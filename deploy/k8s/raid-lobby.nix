_:
let
  images = import ./images.nix;
  labels = {
    "app.kubernetes.io/name" = "raid-lobby";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.raid-lobby-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.raid-lobby-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.raid-lobby-service = {
            image = images.ghcrImage "raid-lobby";
            imagePullPolicy = "Always";
            ports.http.containerPort = 8086;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "raid-lobby-service";
              PORT.value = "8086";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "raid-lobby-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "raid-lobby-secrets";
                key = "KAFKA_BROKERS";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8086;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8086;
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

      services.raid-lobby-service.spec = {
        selector = labels;
        ports.http = {
          port = 8086;
          targetPort = 8086;
          protocol = "TCP";
        };
      };
    };
  };
}
