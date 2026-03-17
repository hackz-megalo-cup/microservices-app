{ lib, ... }:
{
  imports = [
    ../../k8s/greeter.nix
    ../../k8s/caller.nix
    ../../k8s/gateway.nix
    ../../k8s/custom-lang-service.nix
    ../../k8s/auth-service.nix
    ../../k8s/frontend.nix
    ../../k8s/item.nix
    ../../k8s/masterdata.nix
    # No secrets.nix — prod secrets managed by SOPS
    # No traefik.nix — prod ingress managed by infra repo
  ];

  nixidy = {
    target = {
      repository = "https://github.com/hackz-megalo-cup/microservices-app";
      branch = "main";
      rootPath = "./deploy/manifests/prod";
    };

    defaults = {
      destination.server = "https://kubernetes.default.svc";

      syncPolicy = {
        autoSync = {
          enable = true;
          prune = true;
          selfHeal = true;
        };
      };
    };

    appOfApps = {
      name = "apps";
      namespace = "argocd";
    };
  };

  applications = {
    greeter-service.resources.deployments.greeter-service.spec = {
      replicas = lib.mkForce 2;
      template.spec.containers.greeter-service = {
        resources = lib.mkForce {
          requests = {
            cpu = "100m";
            memory = "128Mi";
          };
          limits = {
            cpu = "500m";
            memory = "384Mi";
          };
        };
      };
    };

    caller-service.resources.deployments.caller-service.spec = {
      replicas = lib.mkForce 2;
      template.spec.containers.caller-service = {
        resources = lib.mkForce {
          requests = {
            cpu = "100m";
            memory = "128Mi";
          };
          limits = {
            cpu = "500m";
            memory = "384Mi";
          };
        };
      };
    };

    gateway-service.resources.deployments.gateway.spec = {
      replicas = lib.mkForce 3;
      template.spec.containers.gateway = {
        resources = lib.mkForce {
          requests = {
            cpu = "100m";
            memory = "128Mi";
          };
          limits = {
            cpu = "500m";
            memory = "512Mi";
          };
        };
      };
    };

    custom-lang-service.resources.deployments.custom-lang-service.spec = {
      replicas = lib.mkForce 2;
      template.spec.containers.custom-lang-service = {
        resources = lib.mkForce {
          requests = {
            cpu = "50m";
            memory = "128Mi";
          };
          limits = {
            cpu = "200m";
            memory = "384Mi";
          };
        };
      };
    };

    auth-service.resources.deployments.auth-service.spec = {
      replicas = lib.mkForce 2;
      template.spec.containers.auth-service = {
        resources = lib.mkForce {
          requests = {
            cpu = "50m";
            memory = "128Mi";
          };
          limits = {
            cpu = "200m";
            memory = "384Mi";
          };
        };
      };
    };
  };
}
