import app from './app.js';
import { createKafkaClient, createOutbox } from '@microservices/shared';
import pool from '@microservices/shared/db.js';

const kafka = createKafkaClient('auth-service');
const outbox = createOutbox('auth-service', pool, kafka);

const port = process.env.PORT || 8090;

app.locals.kafka = kafka;
app.locals.outbox = outbox;

app.listen(port, () => {
  console.log(`auth-service listening on :${port}`);
  outbox.startPoller();
});
