import http2 from "node:http2";
import { connectNodeAdapter } from "@connectrpc/connect-node";
import { createKafkaClient, createOutbox } from "@microservices/shared";
import pool from "@microservices/shared/db.js";
import app, { kid, privateKey } from "./app.js";
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
  router.rpc({ service: "auth.v1.AuthService", method: "RegisterUser" }, async (req) => {
    return await registerUser(req, grpcContext);
  });

  router.rpc({ service: "auth.v1.AuthService", method: "LoginUser" }, async (req) => {
    return await loginUser(req, grpcContext);
  });

  router.rpc({ service: "auth.v1.AuthService", method: "GetUserProfile" }, async (req) => {
    return await getUserProfile(req, grpcContext);
  });
}

// gRPC ハンドラー（connectNodeAdapter を使用）
const grpcHandler = connectNodeAdapter({
  routes,
});

// http2 サーバー作成（REST + gRPC）
const server = http2.createServer((req, res) => {
  const url = req.url || "";

  // REST エンドポイント（Traefik forward-auth と JWKS のみ）
  if (
    url.startsWith("/verify") ||
    url.startsWith("/auth/verify") ||
    url.startsWith("/.well-known/jwks.json") ||
    url === "/healthz"
  ) {
    // Express で処理
    app(req, res);
  } else {
    // gRPC で処理
    grpcHandler(req, res);
  }
});

server.listen(port, () => {
  console.log(`auth-service listening on :${port}`);
  console.log(`  gRPC API: grpc://localhost:${port} (auth.v1.AuthService)`);
  console.log(`  REST API (limited): /verify, /jwks.json, /healthz`);
  outbox.startPoller();
});
