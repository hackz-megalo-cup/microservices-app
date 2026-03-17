_:
let
  labels = {
    "app.kubernetes.io/name" = "auth-service";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.auth-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.auth-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.auth-service = {
            image = "ghcr.io/hackz-megalo-cup/auth-service:latest";
            imagePullPolicy = "Always";
            ports.http.containerPort = 8090;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "auth-service";
              PORT.value = "8090";
              DATABASE_URL.valueFrom.secretKeyRef = {
                name = "auth-secrets";
                key = "DATABASE_URL";
              };
              KAFKA_BROKERS.valueFrom.secretKeyRef = {
                name = "auth-secrets";
                key = "KAFKA_BROKERS";
              };
              JWT_SECRET.valueFrom.secretKeyRef = {
                name = "auth-secrets";
                key = "JWT_SECRET";
              };
            };

            livenessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8090;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/healthz";
                port = 8090;
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

      services.auth-service.spec = {
        selector = labels;
        ports.http = {
          port = 8090;
          targetPort = 8090;
          protocol = "TCP";
        };
      };
    };
  };
}
