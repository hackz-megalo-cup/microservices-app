_: {
  applications.microservices-secrets = {
    namespace = "microservices";
    createNamespace = false;

    resources.secrets = {
      gateway-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/gateway_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
        };
      };

      custom-lang-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/lang_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
        };
      };

      auth-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/auth_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
          JWT_SECRET = "dev-secret";
        };
      };

      item-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/item_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
        };
      };

      masterdata-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/masterdata_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
        };
      };

      raid-lobby-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/raid_lobby_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
        };
      };
      lobby-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/lobby_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
          AUTH_URL = "http://auth-service.microservices:8090";
          ITEM_URL = "http://item-service.microservices:8080";
          RAID_LOBBY_URL = "http://raid-lobby-service.microservices:8086";
          MASTERDATA_URL = "http://masterdata-service.microservices:8084";
        };
      };

      capture-secrets = {
        type = "Opaque";
        stringData = {
          DATABASE_URL = "postgresql://devuser:devpass@postgresql.database:5432/capture_db";
          KAFKA_BROKERS = "redpanda.messaging:9092";
        };
      };

    };
  };
}
