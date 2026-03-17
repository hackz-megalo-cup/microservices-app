import { connectNodeAdapter } from "@connectrpc/connect-node";
import { createKafkaClient, createOutbox } from "@microservices/shared";
import pool from "@microservices/shared/db.js";
import express from "express";
import { AuthService } from "../gen/auth/v1/auth_pb.js";
import app, { kid, privateKey } from "./app.js";
import { startCaptureConsumer } from "./consumer.js";
import { getUserProfile, loginUser, registerUser } from "./handlers.js";

const kafka = createKafkaClient("auth-service");
const outbox = createOutbox("auth-service", pool, kafka);

const port = process.env.PORT || 8090;

app.locals.kafka = kafka;
app.locals.outbox = outbox;

// gRPC ハンドラーが使う共通コンテキスト
const grpcContext = {
  outbox,
  privateKey,
  kid,
};

// Connect-RPC ルーター設定
function routes(router) {
  router.service(AuthService, {
    async registerUser(req) {
      return await registerUser(req, grpcContext);
    },
    async loginUser(req) {
      return await loginUser(req, grpcContext);
    },
    async getUserProfile(req) {
      return await getUserProfile(req, grpcContext);
    },
  });
}

// メインサーバー：gRPCとRESTを統合
const server = express();
const grpcHandler = connectNodeAdapter({ routes });

// 1. gRPCパス（/auth.v1.*）の処理
server.use((req, res, next) => {
  if (req.path.startsWith("/auth.v1.")) {
    return grpcHandler(req, res, next);
  }
  next();
});

// 2. RESTエンドポイント（app.jsで定義）
server.use(app);

server.listen(port, async () => {
  console.log(
    `auth-service listening on :${port} (gRPC: auth.v1.AuthService, REST: /verify, /jwks.json, /healthz)`,
  );
  outbox.startPoller();
  await startCaptureConsumer(kafka);
});
