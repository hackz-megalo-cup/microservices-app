import { createKafkaClient, createOutbox } from "@microservices/shared";
import pool from "@microservices/shared/db.js";
import app from "./app.js";

const kafka = createKafkaClient("custom-lang-service");
const outbox = createOutbox("custom-lang-service", pool, kafka);

const port = process.env.PORT || 3000;

app.locals.kafka = kafka;
app.locals.outbox = outbox;

app.listen(port, () => {
  console.log(`custom-lang-service listening on :${port}`);
  outbox.startPoller();
});
