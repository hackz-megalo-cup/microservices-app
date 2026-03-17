_:
let
  labels = {
    "app.kubernetes.io/name" = "frontend";
    "app.kubernetes.io/version" = "0.1.0";
  };
in
{
  applications.frontend = {
    namespace = "microservices";
    createNamespace = false;

    resources = {
      deployments.frontend.spec = {
        replicas = 1;
        selector.matchLabels = labels;
        template = {
          metadata.labels = labels;
          spec.containers.frontend = {
            image = "ghcr.io/hackz-megalo-cup/frontend:latest";
            imagePullPolicy = "Always";
            ports.http.containerPort = 80;

            livenessProbe = {
              httpGet = {
                path = "/";
                port = 80;
              };
              initialDelaySeconds = 5;
              periodSeconds = 10;
            };

            readinessProbe = {
              httpGet = {
                path = "/";
                port = 80;
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
                memory = "128Mi";
              };
            };
          };
        };
      };

      services.frontend.spec = {
        selector = labels;
        ports.http = {
          port = 80;
          targetPort = 80;
          protocol = "TCP";
        };
      };
    };
  };
}
