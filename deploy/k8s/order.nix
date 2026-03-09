_:
let
  labels = {
    "app.kubernetes.io/name" = "order";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.order-service = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.order-service.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.order-service = {
            image = "order:latest";
            imagePullPolicy = "Never";
            ports.http.containerPort = 8084;

            env = {
              OTEL_EXPORTER_OTLP_ENDPOINT.value = "http://otel-collector.observability:4317";
              OTEL_SERVICE_NAME.value = "order-service";
              PORT.value = "8084";
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

      services.order-service.spec = {
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
