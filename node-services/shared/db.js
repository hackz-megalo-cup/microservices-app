import pg from 'pg';

const databaseUrl = process.env.DATABASE_URL;

const pool = databaseUrl ? new pg.Pool({ connectionString: databaseUrl, max: 10 }) : null;

export async function healthCheck() {
  if (!pool) return;
  const client = await pool.connect();
  try {
    await client.query('SELECT 1');
  } finally {
    client.release();
  }
}

export function getPool() {
  return pool;
}

export default pool;
