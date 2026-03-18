{ lib, ... }:
{
  imports = [
    ../../k8s/gateway.nix
    ../../k8s/auth-service.nix
    ../../k8s/frontend.nix
    ../../k8s/item.nix
    ../../k8s/masterdata.nix
    ../../k8s/raid-lobby.nix
    ../../k8s/lobby.nix
    ../../k8s/capture.nix
    ../../k8s/projector.nix
    # Keep secrets.nix until the existing app path is migrated off repo-managed secrets.
    ../../k8s/secrets.nix
    # No traefik.nix — prod ingress managed by infra repo
  ];

  nixidy = {
    target = {
      repository = "https://github.com/hackz-megalo-cup/microservices-app";
      branch = "main";
      rootPath = "./deploy/manifests";
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
