import { Kafka, logLevel } from "kafkajs";

const brokers = process.env.KAFKA_BROKERS?.split(",").filter(Boolean) || [];

let kafka = null;
let producer = null;
let consumer = null;

export function isKafkaEnabled() {
  return brokers.length > 0;
}

export async function getProducer() {
  if (!isKafkaEnabled()) return null;
  if (producer) return producer;

  kafka = new Kafka({
    clientId: process.env.OTEL_SERVICE_NAME || "unknown-service",
    brokers,
    logLevel: logLevel.WARN,
    retry: { retries: 3 },
  });

  producer = kafka.producer();
  try {
    await producer.connect();
    console.log(`Kafka producer connected to ${brokers.join(",")}`);
  } catch (err) {
    console.warn("Kafka producer connection failed, disabling:", err.message);
    producer = null;
  }
  return producer;
}

export async function getConsumer(groupId, topics, eachMessageHandler) {
  if (!isKafkaEnabled()) return null;

  if (!kafka) {
    kafka = new Kafka({
      clientId: process.env.OTEL_SERVICE_NAME || "unknown-service",
      brokers,
      logLevel: logLevel.WARN,
    });
  }

  const c = kafka.consumer({ groupId });
  try {
    await c.connect();
    for (const topic of topics) {
      await c.subscribe({ topic, fromBeginning: false });
    }
    await c.run({
      eachMessage: eachMessageHandler,
    });
    console.log(`Kafka consumer running: group=${groupId}, topics=${topics}`);
    return c;
  } catch (err) {
    console.warn("Kafka consumer connection failed:", err.message);
    return null;
  }
}

export async function publishEvent(topic, event) {
  const p = await getProducer();
  if (!p) return;
  await p.send({
    topic,
    messages: [{ key: event.key, value: JSON.stringify(event.payload) }],
  });
}

export async function shutdown() {
  if (producer) await producer.disconnect();
  if (consumer) await consumer.disconnect();
}
