_:
let
  images = import ./images.nix;
  labels = {
    "app.kubernetes.io/name" = "gateway";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.gateway-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.gateway.spec = {
        replicas = 2;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.gateway = {
            image = images.ghcrImage "gateway";
            imagePullPolicy = "Always";
            ports.http.containerPort = 8082;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "gateway-service";
              CUSTOM_LANG_BASE_URL.value = "http://custom-lang-service.microservices:3000";
              PORT.value = "8082";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "gateway-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "gateway-secrets";
                key = "KAFKA_BROKERS";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8082;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8082;
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
                memory = "256Mi";
              };
            };
          };
        };
      };

      services.gateway.spec = {
        selector = labels;
        ports.http = {
          port = 8082;
          targetPort = 8082;
          protocol = "TCP";
        };
      };

      podDisruptionBudgets.gateway-pdb.spec = {
        minAvailable = 1;
        selector.matchLabels = labels;
      };
    };
  };
}
