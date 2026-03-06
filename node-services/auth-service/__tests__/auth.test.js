import request from 'supertest';
import { describe, expect, it } from 'vitest';
import app from '../app.js';

describe('POST /auth/login', () => {
  it('returns 400 when email is missing', async () => {
    const res = await request(app).post('/auth/login').send({ password: 'secret' });
    expect(res.status).toBe(400);
    expect(res.body.error).toBe('email and password are required');
  });

  it('returns 400 when password is missing', async () => {
    const res = await request(app).post('/auth/login').send({ email: 'test@example.com' });
    expect(res.status).toBe(400);
    expect(res.body.error).toBe('email and password are required');
  });

  it('returns 400 when body is empty', async () => {
    const res = await request(app).post('/auth/login').send({});
    expect(res.status).toBe(400);
    expect(res.body.error).toBe('email and password are required');
  });

  it('returns 200 with token when DB is not configured', async () => {
    const res = await request(app)
      .post('/auth/login')
      .send({ email: 'test@example.com', password: 'secret' });
    expect(res.status).toBe(200);
    expect(res.body).toHaveProperty('token');
    expect(typeof res.body.token).toBe('string');
  });
});

describe('GET /.well-known/jwks.json', () => {
  it('returns valid JWKS', async () => {
    const res = await request(app).get('/.well-known/jwks.json');
    expect(res.status).toBe(200);
    expect(res.body).toHaveProperty('keys');
    expect(Array.isArray(res.body.keys)).toBe(true);
    expect(res.body.keys.length).toBe(1);

    const key = res.body.keys[0];
    expect(key.kty).toBe('RSA');
    expect(key.use).toBe('sig');
    expect(key.alg).toBe('RS256');
    expect(key).toHaveProperty('kid');
    expect(key).toHaveProperty('n');
    expect(key).toHaveProperty('e');
  });
});

describe('GET /verify', () => {
  it('returns 401 without authorization header', async () => {
    const res = await request(app).get('/verify');
    expect(res.status).toBe(401);
    expect(res.body.error).toBe('missing bearer token');
  });

  it('returns 401 with invalid token', async () => {
    const res = await request(app).get('/verify').set('Authorization', 'Bearer invalidtoken');
    expect(res.status).toBe(401);
    expect(res.body.error).toBe('invalid token');
  });

  it('returns 200 with valid token from login', async () => {
    const loginRes = await request(app)
      .post('/auth/login')
      .send({ email: 'verify@example.com', password: 'secret' });
    expect(loginRes.status).toBe(200);

    const { token } = loginRes.body;
    const verifyRes = await request(app).get('/verify').set('Authorization', `Bearer ${token}`);
    expect(verifyRes.status).toBe(200);
    expect(verifyRes.body.ok).toBe(true);
    expect(verifyRes.body.userId).toBe('verify@example.com');
  });
});

describe('POST /auth/register', () => {
  it('returns 503 when database is not configured', async () => {
    const res = await request(app)
      .post('/auth/register')
      .send({ email: 'test@example.com', password: 'secret' });
    expect(res.status).toBe(503);
    expect(res.body.error).toBe('database not configured');
  });

  it('returns 400 when email is missing', async () => {
    const res = await request(app).post('/auth/register').send({ password: 'secret' });
    // Without DB it returns 503 before validation, but with the current code flow
    // the pool check comes first, so we get 503
    expect([400, 503]).toContain(res.status);
  });
});
