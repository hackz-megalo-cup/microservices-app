{ ... }:
{
  imports = [
    ../../k8s/greeter.nix
    ../../k8s/caller.nix
    ../../k8s/gateway.nix
    ../../k8s/custom-lang-service.nix
    ../../k8s/auth-service.nix
    ../../k8s/frontend.nix
    ./traefik.nix
  ];

  nixidy = {
    target = {
      repository = "https://github.com/thirdlf03/microservice-app";
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
}
