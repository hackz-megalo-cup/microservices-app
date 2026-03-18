_:
let
  labels = {
    "app.kubernetes.io/name" = "lobby";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.lobby-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.lobby-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.lobby-service = {
            image = "lobby:latest";
            imagePullPolicy = "Never";
            ports.http.containerPort = 8089;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "lobby-service";
              PORT.value = "8089";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "lobby-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "lobby-secrets";
                key = "KAFKA_BROKERS";
              };
              ITEM_DATABASE_URL.valueFrom.secretKeyRef = {
                name = "lobby-secrets";
                key = "ITEM_DATABASE_URL";
              };
              RAID_LOBBY_DATABASE_URL.valueFrom.secretKeyRef = {
                name = "lobby-secrets";
                key = "RAID_LOBBY_DATABASE_URL";
              };
              MASTERDATA_URL.valueFrom.secretKeyRef = {
                name = "lobby-secrets";
                key = "MASTERDATA_URL";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8089;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8089;
              };
              initialDelaySeconds = 3;
              periodSeconds = 5;
            };

            resources = {
              requests = {
                cpu = "50m";
                memory = "64Mi";
              };
              limits = {
                cpu = "200m";
                memory = "128Mi";
              };
            };
          };
        };
      };

      services.lobby-service.spec = {
        selector = labels;
        ports.http = {
          port = 8089;
          targetPort = 8089;
          protocol = "TCP";
        };
      };
    };
  };
}
