_:
let
  images = import ./images.nix;
  labels = {
    "app.kubernetes.io/name" = "item";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.item-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.item-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.item-service = {
            image = images.ghcrImage "item";
            imagePullPolicy = "Always";
            ports.http.containerPort = 8080;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "item-service";
              PORT.value = "8080";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "item-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "item-secrets";
                key = "KAFKA_BROKERS";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8080;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8080;
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

      services.item-service.spec = {
        selector = labels;
        ports.http = {
          port = 8080;
          targetPort = 8080;
          protocol = "TCP";
        };
      };
    };
  };
}
