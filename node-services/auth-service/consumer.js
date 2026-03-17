import pool from "@microservices/shared/db.js";

/**
 * Start Kafka consumer for capture.completed events
 * Listens for successful captures and registers them in user_pokemon table
 * @param {object} kafkaClient - Kafka client instance
 */
export async function startCaptureConsumer(kafkaClient) {
  if (!kafkaClient || !kafkaClient.isKafkaEnabled()) {
    console.log("Kafka not enabled, skipping capture consumer");
    return;
  }

  try {
    const eachMessageHandler = async ({ message }) => {
      try {
        const payload =
          typeof message.value === "string"
            ? JSON.parse(message.value)
            : JSON.parse(message.value.toString());

        // Only process successful captures
        if (payload.result !== "success") {
          return;
        }

        const { user_id, pokemon_id } = payload;

        if (!user_id || !pokemon_id) {
          throw new Error(`Invalid payload: missing user_id or pokemon_id`);
        }

        if (!pool) {
          throw new Error("Database pool not available");
        }

        // Insert into user_pokemon (or ignore on duplicate)
        await pool.query(
          `INSERT INTO user_pokemon (user_id, pokemon_id, caught_at)
           VALUES ($1, $2, NOW())
           ON CONFLICT (user_id, pokemon_id) DO NOTHING`,
          [user_id, pokemon_id],
        );

        console.log(`Registered pokemon ${pokemon_id} for user ${user_id}`);
      } catch (err) {
        console.error("Error processing capture event:", err);
        throw err;
      }
    };

    await kafkaClient.getConsumer(
      "auth-service-consumer",
      ["capture.completed"],
      eachMessageHandler,
    );
  } catch (err) {
    console.error("Failed to start capture consumer:", err);
  }
}
