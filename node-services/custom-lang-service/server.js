import app from "./app.js";
import { createKafkaClient, createOutbox } from "@microservices/shared";
import pool from "@microservices/shared/db.js";

const kafka = createKafkaClient("custom-lang-service");
const outbox = createOutbox("custom-lang-service", pool, kafka);

const port = process.env.PORT || 3000;

app.locals.kafka = kafka;
app.locals.outbox = outbox;

app.listen(port, () => {
  console.log(`custom-lang-service listening on :${port}`);
  outbox.startPoller();
});
