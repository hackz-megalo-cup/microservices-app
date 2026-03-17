_:
let
  images = import ./images.nix;
  labels = {
    "app.kubernetes.io/name" = "custom-lang-service";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.custom-lang-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.custom-lang-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.custom-lang-service = {
            image = images.ghcrImage "custom-lang-service";
            imagePullPolicy = "Always";
            ports.http.containerPort = 3000;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "custom-lang-service";
              PORT.value = "3000";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "custom-lang-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "custom-lang-secrets";
                key = "KAFKA_BROKERS";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 3000;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 3000;
              };
              initialDelaySeconds = 3;
              periodSeconds = 5;
            };

            resources = {
              requests = {
                cpu = "25m";
                memory = "64Mi";
              };
              limits = {
                cpu = "100m";
                memory = "192Mi";
              };
            };
          };
        };
      };

      services.custom-lang-service.spec = {
        selector = labels;
        ports.http = {
          port = 3000;
          targetPort = 3000;
          protocol = "TCP";
        };
      };
    };
  };
}
