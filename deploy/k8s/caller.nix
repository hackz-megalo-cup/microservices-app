_:
let
  labels = {
    "app.kubernetes.io/name" = "caller";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.caller-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.caller-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.caller-service = {
            image = "caller:latest";
            imagePullPolicy = "Never";
            ports.http.containerPort = 8081;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "caller-service";
              PORT.value = "8081";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "caller-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "caller-secrets";
                key = "KAFKA_BROKERS";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8081;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8081;
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
                memory = "192Mi";
              };
            };
          };
        };
      };

      services.caller-service.spec = {
        selector = labels;
        ports.http = {
          port = 8081;
          targetPort = 8081;
          protocol = "TCP";
        };
      };
    };
  };
}
