{ charts, ... }:
{
  applications.traefik = {
    namespace = "edge";
    createNamespace = true;

    helm.releases.traefik = {
      chart = charts.traefik.traefik;
      values = {
        image.tag = "v3.2.0";

        service = {
          type = "NodePort";
          spec = {
            externalTrafficPolicy = "Cluster";
          };
        };

        ports = {
          web.nodePort = 30081;
          websecure.nodePort = 30444;
        };

        providers = {
          kubernetesCRD.enabled = true;
          kubernetesIngress.enabled = true;
        };

        logs.general.level = "INFO";

        tracing = {
          otlp = {
            grpc = {
              enabled = true;
              endpoint = "otel-collector.observability:4317";
              insecure = true;
            };
          };
        };

        extraObjects = [
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "Middleware";
            metadata = {
              name = "cors-middleware";
              namespace = "microservices";
            };
            spec.headers = {
              accessControlAllowMethods = [
                "GET"
                "POST"
                "OPTIONS"
              ];
              accessControlAllowHeaders = [
                "Content-Type"
                "Authorization"
                "Connect-Protocol-Version"
                "Connect-Timeout-Ms"
                "Grpc-Timeout"
                "X-Grpc-Web"
                "X-User-Agent"
                "Idempotency-Key"
              ];
              accessControlAllowOriginList = [ "http://localhost:5173" ];
              accessControlExposeHeaders = [
                "Grpc-Status"
                "Grpc-Message"
                "Grpc-Status-Details-Bin"
              ];
              accessControlMaxAge = 7200;
              addVaryHeader = true;
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "Middleware";
            metadata = {
              name = "rate-limit-middleware";
              namespace = "microservices";
            };
            spec.rateLimit = {
              average = 100;
              burst = 50;
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "Middleware";
            metadata = {
              name = "retry-middleware";
              namespace = "microservices";
            };
            spec.retry = {
              attempts = 3;
              initialInterval = "100ms";
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "IngressRoute";
            metadata = {
              name = "greeter-route";
              namespace = "microservices";
            };
            spec = {
              entryPoints = [ "web" ];
              routes = [
                {
                  match = "PathPrefix(`/greeter.v1.GreeterService`)";
                  kind = "Rule";
                  priority = 100;
                  middlewares = [
                    { name = "cors-middleware"; }
                    { name = "rate-limit-middleware"; }
                    { name = "retry-middleware"; }
                  ];
                  services = [
                    {
                      name = "greeter-service";
                      port = 80;
                      scheme = "h2c";
                    }
                  ];
                }
                {
                  match = "PathPrefix(`/greeter.v2.GreeterService`)";
                  kind = "Rule";
                  priority = 100;
                  middlewares = [
                    { name = "cors-middleware"; }
                    { name = "rate-limit-middleware"; }
                    { name = "retry-middleware"; }
                  ];
                  services = [
                    {
                      name = "greeter-service";
                      port = 80;
                      scheme = "h2c";
                    }
                  ];
                }
              ];
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "IngressRoute";
            metadata = {
              name = "gateway-route";
              namespace = "microservices";
            };
            spec = {
              entryPoints = [ "web" ];
              routes = [
                {
                  match = "PathPrefix(`/gateway.v1.GatewayService`)";
                  kind = "Rule";
                  priority = 100;
                  middlewares = [
                    { name = "cors-middleware"; }
                    { name = "rate-limit-middleware"; }
                  ];
                  services = [
                    {
                      name = "gateway";
                      port = 8082;
                      scheme = "h2c";
                    }
                  ];
                }
              ];
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "IngressRoute";
            metadata = {
              name = "auth-route";
              namespace = "microservices";
            };
            spec = {
              entryPoints = [ "web" ];
              routes = [
                {
                  match = "PathPrefix(`/auth`)";
                  kind = "Rule";
                  priority = 90;
                  middlewares = [
                    { name = "cors-middleware"; }
                    { name = "rate-limit-middleware"; }
                  ];
                  services = [
                    {
                      name = "auth-service";
                      port = 8090;
                    }
                  ];
                }
              ];
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "IngressRoute";
            metadata = {
              name = "item-route";
              namespace = "microservices";
            };
            spec = {
              entryPoints = [ "web" ];
              routes = [
                {
                  match = "PathPrefix(`/item.v1.ItemService`)";
                  kind = "Rule";
                  priority = 100;
                  middlewares = [
                    { name = "cors-middleware"; }
                    { name = "rate-limit-middleware"; }
                    { name = "retry-middleware"; }
                  ];
                  services = [
                    {
                      name = "item-service";
                      port = 8080;
                      scheme = "h2c";
                    }
                  ];
                }
              ];
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "IngressRoute";
            metadata = {
              name = "masterdata-route";
              namespace = "microservices";
            };
            spec = {
              entryPoints = [ "web" ];
              routes = [
                {
                  match = "PathPrefix(`/masterdata.v1.MasterdataService`)";
                  kind = "Rule";
                  priority = 100;
                  middlewares = [
                    { name = "cors-middleware"; }
                    { name = "rate-limit-middleware"; }
                    { name = "retry-middleware"; }
                  ];
                  services = [
                    {
                      name = "masterdata-service";
                      port = 8084;
                      scheme = "h2c";
                    }
                  ];
                }
              ];
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "IngressRoute";
            metadata = {
              name = "raid-lobby-route";
              namespace = "microservices";
            };
            spec = {
              entryPoints = [ "web" ];
              routes = [
                {
                  match = "PathPrefix(`/raid_lobby.v1.RaidLobbyService`)";
                  kind = "Rule";
                  priority = 100;
                  middlewares = [
                    { name = "cors-middleware"; }
                    { name = "rate-limit-middleware"; }
                    { name = "retry-middleware"; }
                  ];
                  services = [
                    {
                      name = "raid-lobby-service";
                      port = 8086;
                      scheme = "h2c";
                    }
                  ];
                }
              ];
            };
          }
          {
            apiVersion = "traefik.io/v1alpha1";
            kind = "IngressRoute";
            metadata = {
              name = "frontend-route";
              namespace = "microservices";
            };
            spec = {
              entryPoints = [ "web" ];
              routes = [
                {
                  match = "PathPrefix(`/`)";
                  kind = "Rule";
                  priority = 1;
                  services = [
                    {
                      name = "frontend";
                      port = 80;
                    }
                  ];
                }
              ];
            };
          }
        ];
      };
    };
  };
}
