import crypto from 'node:crypto';
import bcrypt from 'bcrypt';
import cors from 'cors';
import express from 'express';
import jwt from 'jsonwebtoken';
import { healthCheck } from '@microservices/shared';
import pool from '@microservices/shared/db.js';
import { idempotencyMiddleware } from '@microservices/shared';
import { retryWithBackoff } from '@microservices/shared';

const app = express();
app.use(cors());
app.use(express.json());

// Load RSA key pair from env (prod) or generate at startup (dev)
let publicKey, privateKey;

if (process.env.RSA_PRIVATE_KEY && process.env.RSA_PUBLIC_KEY) {
  privateKey = process.env.RSA_PRIVATE_KEY;
  publicKey = process.env.RSA_PUBLIC_KEY;
} else {
  const pair = crypto.generateKeyPairSync('rsa', {
    modulusLength: 2048,
    publicKeyEncoding: { type: 'spki', format: 'pem' },
    privateKeyEncoding: { type: 'pkcs8', format: 'pem' },
  });
  publicKey = pair.publicKey;
  privateKey = pair.privateKey;
}

// Extract key components for JWKS
const publicKeyObj = crypto.createPublicKey(publicKey);
const jwk = publicKeyObj.export({ format: 'jwk' });
const kid = crypto
  .createHash('sha256')
  .update(JSON.stringify({ e: jwk.e, kty: jwk.kty, n: jwk.n }))
  .digest('base64url');

app.get('/healthz', async (_req, res) => {
  try {
    await healthCheck();
    res.status(200).send('ok\n');
  } catch (_err) {
    res.status(503).json({ error: 'db health check failed' });
  }
});

app.get('/.well-known/jwks.json', (_req, res) => {
  res.json({
    keys: [
      {
        kty: 'RSA',
        use: 'sig',
        alg: 'RS256',
        kid,
        n: jwk.n,
        e: jwk.e,
      },
    ],
  });
});

app.post('/auth/register', idempotencyMiddleware(), async (req, res) => {
  if (!pool) {
    return res.status(503).json({ error: 'database not configured' });
  }
  const email = req.body?.email;
  const password = req.body?.password;
  if (!email || !password) {
    return res.status(400).json({ error: 'email and password are required' });
  }

  try {
    const passwordHash = await bcrypt.hash(password, 10);
    const client = await pool.connect();
    try {
      await client.query('BEGIN');
      const result = await client.query(
        'INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, role',
        [email, passwordHash],
      );
      const user = result.rows[0];

      await req.app.locals.outbox.insertEvent(client, 'user.registered', {
        payload: {
          userId: user.id,
          email: user.email,
          role: user.role,
          timestamp: new Date().toISOString(),
        },
      });

      await client.query('COMMIT');
      return res.status(201).json(user);
    } catch (innerErr) {
      await client.query('ROLLBACK');
      throw innerErr;
    } finally {
      client.release();
    }
  } catch (err) {
    if (err.code === '23505') {
      return res.status(409).json({ error: 'email already exists' });
    }
    console.error('register error:', err);
    return res.status(500).json({ error: 'internal server error' });
  }
});

app.post('/auth/login', async (req, res) => {
  const email = req.body?.email;
  const password = req.body?.password;
  if (!email || !password) {
    return res.status(400).json({ error: 'email and password are required' });
  }

  // DB がなければ旧来の簡易ログイン (JWT だけ発行)
  if (!pool) {
    const token = jwt.sign({ sub: email, role: 'user' }, privateKey, {
      algorithm: 'RS256',
      issuer: 'auth-service',
      expiresIn: '24h',
      keyid: kid,
    });
    return res.status(200).json({ token });
  }

  try {
    const result = await retryWithBackoff(() =>
      pool.query('SELECT * FROM users WHERE email = $1', [email]),
    );
    const user = result.rows[0];
    if (!user || !(await bcrypt.compare(password, user.password_hash))) {
      return res.status(401).json({ error: 'invalid email or password' });
    }

    const token = jwt.sign(
      {
        sub: user.id,
        role: user.role,
      },
      privateKey,
      { algorithm: 'RS256', issuer: 'auth-service', expiresIn: '24h', keyid: kid },
    );

    return res.status(200).json({ token });
  } catch (err) {
    console.error('login error:', err);
    return res.status(500).json({ error: 'internal server error' });
  }
});

const verifyHandler = (req, res) => {
  const authHeader = req.header('authorization') || '';
  if (!authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'missing bearer token' });
  }

  const token = authHeader.slice('Bearer '.length);

  try {
    const payload = jwt.verify(token, publicKey, {
      algorithms: ['RS256'],
      issuer: 'auth-service',
    });
    const userId = payload.sub?.toString() || 'unknown';
    res.setHeader('X-User-Id', userId);
    return res.status(200).json({ ok: true, userId });
  } catch (_err) {
    return res.status(401).json({ error: 'invalid token' });
  }
};

app.get('/verify', verifyHandler);
app.get('/auth/verify', verifyHandler);

export default app;
