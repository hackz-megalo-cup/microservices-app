import { getPool } from './db.js';

export function idempotencyMiddleware() {
  return async (req, res, next) => {
    const key = req.headers['idempotency-key'];
    if (!key) return next();

    const pool = getPool();
    if (!pool) return next();

    try {
      const result = await pool.query(
        'SELECT response, status_code FROM idempotency_keys WHERE key = $1 AND expires_at > NOW()',
        [key],
      );
      if (result.rows.length > 0) {
        const cached = result.rows[0];
        return res.status(cached.status_code).json(JSON.parse(cached.response));
      }
    } catch {
      // store unavailable, proceed without idempotency
    }

    const originalJson = res.json.bind(res);
    res.json = (body) => {
      pool
        ?.query(
          `INSERT INTO idempotency_keys (key, response, status_code, expires_at)
         VALUES ($1, $2, $3, NOW() + INTERVAL '24 hours')
         ON CONFLICT (key) DO UPDATE SET response = $2, status_code = $3`,
          [key, JSON.stringify(body), res.statusCode],
        )
        .catch(() => {});
      return originalJson(body);
    };
    next();
  };
}
