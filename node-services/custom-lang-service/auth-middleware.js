import { createRemoteJWKSet, jwtVerify } from 'jose';

let jwks = null;

function getJWKS() {
  if (!jwks) {
    const jwksURL = process.env.JWKS_URL;
    if (!jwksURL) return null;
    jwks = createRemoteJWKSet(new URL(jwksURL));
  }
  return jwks;
}

export function jwtAuthMiddleware() {
  return async (req, res, next) => {
    const keySet = getJWKS();
    if (!keySet) return next(); // JWT verification disabled

    const authHeader = req.headers.authorization;
    if (!authHeader?.startsWith('Bearer ')) {
      return res.status(401).json({ error: 'Missing or invalid authorization header' });
    }

    const token = authHeader.slice(7);
    try {
      const { payload } = await jwtVerify(token, keySet, {
        issuer: 'auth-service',
      });
      req.user = { id: payload.sub, role: payload.role };
      next();
    } catch (_err) {
      return res.status(401).json({ error: 'Invalid token' });
    }
  };
}
