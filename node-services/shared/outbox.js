import crypto from 'node:crypto';

export function createOutbox(serviceName, pool, kafkaClient) {
  let pollerTimeout = null;

  async function insertEvent(client, topic, event) {
    const envelope = {
      id: crypto.randomUUID(),
      type: event.type || topic,
      source: serviceName,
      version: 1,
      timestamp: new Date().toISOString(),
      data: event.payload,
    };
    await client.query(
      `INSERT INTO outbox_events (id, event_type, topic, payload, created_at)
       VALUES ($1, $2, $3, $4, NOW())`,
      [envelope.id, envelope.type, topic, JSON.stringify(envelope)],
    );
    return envelope;
  }

  async function publishPending() {
    if (!pool || !kafkaClient.isKafkaEnabled()) return 0;

    const { rows } = await pool.query(
      `SELECT id, event_type, topic, payload
       FROM outbox_events
       WHERE published = FALSE
       ORDER BY created_at ASC
       LIMIT 50
       FOR UPDATE SKIP LOCKED`,
    );

    let published = 0;
    for (const row of rows) {
      try {
        const envelope = typeof row.payload === 'string' ? JSON.parse(row.payload) : row.payload;
        const producer = await kafkaClient.getProducer();
        if (!producer) continue;
        await producer.send({
          topic: row.topic,
          messages: [{ key: row.id, value: JSON.stringify(envelope) }],
        });
        await pool.query(
          `UPDATE outbox_events SET published = TRUE, published_at = NOW() WHERE id = $1`,
          [row.id],
        );
        published++;
      } catch (err) {
        console.error('outbox: failed to publish event', row.id, err);
      }
    }
    return published;
  }

  async function cleanup(maxAgeMs = 24 * 60 * 60 * 1000) {
    if (!pool) return;
    const cutoff = new Date(Date.now() - maxAgeMs);
    await pool.query(
      `DELETE FROM outbox_events WHERE published = TRUE AND published_at < $1`,
      [cutoff],
    );
  }

  const BASE_INTERVAL = 500;
  const MAX_INTERVAL = 5000;
  let currentInterval = BASE_INTERVAL;

  function startPoller() {
    if (pollerTimeout) return;

    async function poll() {
      try {
        const n = await publishPending();
        if (n > 0) {
          console.log(`outbox poller published ${n} events`);
          currentInterval = BASE_INTERVAL;
        } else {
          currentInterval = Math.min(currentInterval * 2, MAX_INTERVAL);
        }
      } catch (err) {
        console.error('outbox poller error:', err);
        currentInterval = Math.min(currentInterval * 2, MAX_INTERVAL);
      }
      pollerTimeout = setTimeout(poll, currentInterval);
    }

    pollerTimeout = setTimeout(poll, currentInterval);
  }

  function stopPoller() {
    if (pollerTimeout) {
      clearTimeout(pollerTimeout);
      pollerTimeout = null;
    }
  }

  return { insertEvent, publishPending, cleanup, startPoller, stopPoller };
}
