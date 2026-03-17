_:
let
  labels = {
    "app.kubernetes.io/name" = "greeter";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.greeter-service = {
    namespace = "microservices";
    createNamespace = true;

    resources = {
      deployments.greeter-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.greeter-service = {
            image = "ghcr.io/hackz-megalo-cup/greeter:latest";
            imagePullPolicy = "Always";
            ports.http.containerPort = 8080;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "greeter-service";
              CALLER_BASE_URL.value = "http://caller-service.microservices:8081";
              EXTERNAL_API_URL.value = "https://httpbin.org/get";
              PORT.value = "8080";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "greeter-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "greeter-secrets";
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
                memory = "192Mi";
              };
            };
          };
        };
      };

      services.greeter-service.spec = {
        selector = labels;
        ports.http = {
          port = 80;
          targetPort = 8080;
          protocol = "TCP";
        };
      };
    };
  };
}
