_:
let
  images = import ./images.nix;
  labels = {
    "app.kubernetes.io/name" = "masterdata";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.masterdata-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.masterdata-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.masterdata-service = {
            image = images.ghcrImage "masterdata";
            imagePullPolicy = "Always";
            ports.http.containerPort = 8084;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "masterdata-service";
              PORT.value = "8084";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "masterdata-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "masterdata-secrets";
                key = "KAFKA_BROKERS";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8084;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8084;
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

      services.masterdata-service.spec = {
        selector = labels;
        ports.http = {
          port = 8084;
          targetPort = 8084;
          protocol = "TCP";
        };
      };
    };
  };
}
