import { healthCheck, idempotencyMiddleware, retryWithBackoff } from "@microservices/shared";
import pool from "@microservices/shared/db.js";
import express from "express";
import { jwtAuthMiddleware } from "./auth-middleware.js";

const app = express();
app.use(express.json());

app.get("/healthz", async (_req, res) => {
  try {
    await healthCheck();
    res.status(200).send("ok\n");
  } catch (_err) {
    res.status(503).json({ error: "db health check failed" });
  }
});

app.post("/invoke", jwtAuthMiddleware(), idempotencyMiddleware(), async (req, res) => {
  const name = (req.body?.name || "World").toString();

  let statusCode;
  let message;

  if (name === "unauthorized") {
    statusCode = 401;
    message = null;
  } else if (name === "forbidden") {
    statusCode = 403;
    message = null;
  } else if (name === "notfound") {
    statusCode = 404;
    message = null;
  } else if (name === "conflict") {
    statusCode = 409;
    message = null;
  } else if (name === "ratelimit") {
    statusCode = 429;
    message = null;
  } else if (name === "unavailable") {
    statusCode = 503;
    message = null;
  } else {
    statusCode = 200;
    message = `Hello ${name} from custom-lang-service!`;
  }

  if (pool) {
    const client = await pool.connect();
    try {
      await client.query("BEGIN");
      await retryWithBackoff(() =>
        client.query(
          "INSERT INTO executions (name, result_message, status_code) VALUES ($1, $2, $3)",
          [name, message, statusCode],
        ),
      );
      if (statusCode === 200) {
        await req.app.locals.outbox.insertEvent(client, "invocation.created", {
          payload: { name, message, statusCode, timestamp: new Date().toISOString() },
        });
      }
      await client.query("COMMIT");
    } catch (err) {
      await client.query("ROLLBACK");
      console.error("failed to record execution:", err);
    } finally {
      client.release();
    }
  }

  if (statusCode === 200) {
    return res.status(200).json({ message });
  }

  const errorMessages = {
    401: "unauthorized",
    403: "forbidden",
    404: "not found",
    409: "conflict",
    429: "rate limited",
    503: "service unavailable",
  };
  return res.status(statusCode).json({ error: errorMessages[statusCode] });
});

export default app;
