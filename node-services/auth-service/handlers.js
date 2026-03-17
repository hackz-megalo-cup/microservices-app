import { retryWithBackoff } from "@microservices/shared";
import pool from "@microservices/shared/db.js";
import bcrypt from "bcrypt";
import jwt from "jsonwebtoken";

/**
 * Convert a Date or ISO string to protobuf Timestamp format
 * @param {Date | string | null} date
 * @returns {{seconds: number, nanos: number} | null}
 */
function toTimestamp(date) {
  if (!date) return null;
  const d = typeof date === "string" ? new Date(date) : date;
  const seconds = Math.floor(d.getTime() / 1000);
  const nanos = (d.getTime() % 1000) * 1000000;
  return { seconds, nanos };
}

/**
 * RegisterUser RPC ハンドラー
 * 既存の REST /auth/register と同じロジック
 */
export async function registerUser(req, context) {
  const { email, password } = req;

  if (!email || !password) {
    throw new Error("email and password are required");
  }

  if (!pool) {
    throw new Error("database not configured");
  }

  try {
    const passwordHash = await bcrypt.hash(password, 10);
    const client = await pool.connect();

    try {
      await client.query("BEGIN");

      const result = await client.query(
        "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, role, created_at, updated_at",
        [email, passwordHash],
      );

      const user = result.rows[0];

      // Outbox パターンでイベント発行
      await context.outbox.insertEvent(client, "user.registered", {
        payload: {
          userId: user.id,
          email: user.email,
          role: user.role,
          timestamp: new Date().toISOString(),
        },
      });

      await client.query("COMMIT");

      return {
        id: user.id,
        email: user.email,
        role: user.role,
        createdAt: toTimestamp(user.created_at),
      };
    } catch (innerErr) {
      await client.query("ROLLBACK");
      throw innerErr;
    } finally {
      client.release();
    }
  } catch (err) {
    if (err.code === "23505") {
      throw new Error("email already exists");
    }
    console.error("registerUser error:", err);
    throw new Error("internal server error");
  }
}

/**
 * LoginUser RPC ハンドラー
 * 既存の REST /auth/login と同じロジック
 */
export async function loginUser(req, context) {
  const { email, password } = req;

  if (!email || !password) {
    throw new Error("email and password are required");
  }

  if (!pool) {
    throw new Error("database not configured");
  }

  try {
    const result = await retryWithBackoff(() =>
      pool.query("SELECT * FROM users WHERE email = $1", [email]),
    );

    const user = result.rows[0];
    if (!user || !(await bcrypt.compare(password, user.password_hash))) {
      throw new Error("invalid email or password");
    }

    // Update last_login_at and get the updated value
    const updateResult = await pool.query(
      "UPDATE users SET last_login_at = NOW() WHERE id = $1 RETURNING last_login_at",
      [user.id],
    );
    const lastLoginAt = updateResult.rows[0].last_login_at;

    const token = jwt.sign(
      {
        sub: user.id,
        role: user.role,
      },
      context.privateKey,
      {
        algorithm: "RS256",
        issuer: "auth-service",
        expiresIn: "24h",
        keyid: context.kid,
      },
    );

    return {
      token,
      user: {
        id: user.id,
        email: user.email,
        role: user.role,
        createdAt: toTimestamp(user.created_at),
        lastLoginAt: toTimestamp(lastLoginAt),
      },
    };
  } catch (err) {
    if (err instanceof Error && err.message === "invalid email or password") {
      throw err;
    }
    console.error("loginUser error:", err);
    throw new Error("authentication failed");
  }
}

/**
 * GetUserProfile RPC ハンドラー
 */
export async function getUserProfile(req, _context) {
  const { userId } = req;

  if (!userId) {
    throw new Error("user_id is required");
  }

  if (!pool) {
    throw new Error("database not configured");
  }

  try {
    const result = await pool.query(
      "SELECT id, email, role, created_at, last_login_at FROM users WHERE id = $1",
      [userId],
    );

    if (result.rows.length === 0) {
      throw new Error("user not found");
    }

    const user = result.rows[0];
    return {
      id: user.id,
      email: user.email,
      role: user.role,
      createdAt: toTimestamp(user.created_at),
      lastLoginAt: user.last_login_at ? toTimestamp(user.last_login_at) : null,
    };
  } catch (err) {
    if (err instanceof Error && err.message === "user not found") {
      throw err;
    }
    console.error("getUserProfile error:", err);
    throw new Error("internal server error");
  }
}
