import { connectNodeAdapter } from "@connectrpc/connect-node";
import { createKafkaClient, createOutbox } from "@microservices/shared";
import pool from "@microservices/shared/db.js";
import { AuthService } from "../gen/auth/v1/auth_pb.js";
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

// gRPC ハンドラーを Express に統合
app.use(connectNodeAdapter({ routes }));

app.listen(port, () => {
  console.log(`auth-service listening on :${port}`);
  console.log(`  gRPC API: grpc://localhost:${port} (auth.v1.AuthService)`);
  console.log(`  REST API (limited): /verify, /jwks.json, /healthz`);
  outbox.startPoller();
});
