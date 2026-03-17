import { Code, ConnectError } from "@connectrpc/connect";
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
    throw new ConnectError("email and password are required", Code.InvalidArgument);
  }

  if (!pool) {
    throw new ConnectError("database not configured", Code.Unavailable);
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
      throw new ConnectError("email already exists", Code.AlreadyExists);
    }
    if (err instanceof ConnectError) {
      throw err;
    }
    console.error("registerUser error:", err);
    throw new ConnectError("internal server error", Code.Internal);
  }
}

/**
 * LoginUser RPC ハンドラー
 * 既存の REST /auth/login と同じロジック
 */
export async function loginUser(req, context) {
  const { email, password } = req;

  if (!email || !password) {
    throw new ConnectError("email and password are required", Code.InvalidArgument);
  }

  if (!pool) {
    throw new ConnectError("database not configured", Code.Unavailable);
  }

  try {
    const result = await retryWithBackoff(() =>
      pool.query("SELECT * FROM users WHERE email = $1", [email]),
    );

    const user = result.rows[0];
    if (!user || !(await bcrypt.compare(password, user.password_hash))) {
      throw new ConnectError("invalid email or password", Code.Unauthenticated);
    }

    // Determine is_first_today before updating last_login_at
    const lastLoginAtDate = user.last_login_at ? new Date(user.last_login_at) : null;
    const today = new Date();
    const isFirstToday =
      !lastLoginAtDate ||
      lastLoginAtDate.getDate() !== today.getDate() ||
      lastLoginAtDate.getMonth() !== today.getMonth() ||
      lastLoginAtDate.getFullYear() !== today.getFullYear();

    const client = await pool.connect();
    let lastLoginAt;
    try {
      await client.query("BEGIN");

      const updateResult = await client.query(
        "UPDATE users SET last_login_at = NOW() WHERE id = $1 RETURNING last_login_at",
        [user.id],
      );
      lastLoginAt = updateResult.rows[0].last_login_at;

      await context.outbox.insertEvent(client, "user.logged_in", {
        payload: {
          userId: user.id,
          isFirstToday,
          timestamp: new Date().toISOString(),
        },
      });

      await client.query("COMMIT");
    } catch (innerErr) {
      await client.query("ROLLBACK");
      throw innerErr;
    } finally {
      client.release();
    }

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
    if (err instanceof ConnectError) {
      throw err;
    }
    console.error("loginUser error:", err);
    throw new ConnectError("authentication failed", Code.Internal);
  }
}

/**
 * GetUserProfile RPC ハンドラー
 */
export async function getUserProfile(req, _context) {
  const { userId } = req;

  if (!userId) {
    throw new ConnectError("user_id is required", Code.InvalidArgument);
  }

  if (!pool) {
    throw new ConnectError("database not configured", Code.Unavailable);
  }

  try {
    const result = await pool.query(
      "SELECT id, email, role, created_at, last_login_at FROM users WHERE id = $1",
      [userId],
    );

    if (result.rows.length === 0) {
      throw new ConnectError("user not found", Code.NotFound);
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
    if (err instanceof ConnectError) {
      throw err;
    }
    console.error("getUserProfile error:", err);
    throw new ConnectError("internal server error", Code.Internal);
  }
}
