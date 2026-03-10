{
  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    devenv.url = "github:cachix/devenv/v2.0.1";
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    devenv.inputs.nixpkgs.follows = "nixpkgs";
    devenv-root = {
      url = "file+file:///dev/null";
      flake = false;
    };
    nix2container.url = "github:nlewo/nix2container";
    nix2container.inputs.nixpkgs.follows = "nixpkgs";
    mk-shell-bin.url = "github:rrbutani/nix-mk-shell-bin";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    nixidy = {
      url = "github:arnarg/nixidy";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nixhelm = {
      url = "github:farcaller/nixhelm";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  nixConfig = {
    extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=";
    extra-substituters = "https://devenv.cachix.org";
  };

  outputs =
    inputs@{ flake-parts, ... }:
    let
      devenvRootFileContent = builtins.readFile inputs.devenv-root.outPath;
      devenvRoot = if devenvRootFileContent != "" then devenvRootFileContent else builtins.toString ./.;
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devenv.flakeModule
        inputs.treefmt-nix.flakeModule
      ];

      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      perSystem =
        { pkgs, system, ... }:
        let
          nix2containerPkgs = inputs.nix2container.packages.${system};

          # Microservices (connect-go) — go.mod requires go 1.26
          buildGoModule = pkgs.buildGo126Module;

          caller = buildGoModule {
            pname = "caller";
            version = "0.1.0";
            src = ./services;
            vendorHash = null;
            env.CGO_ENABLED = 0;
            ldflags = [
              "-s"
              "-w"
            ];
            subPackages = [ "cmd/caller" ];
          };

          caller-image = nix2containerPkgs.nix2container.buildImage {
            name = "caller";
            tag = "latest";
            config = {
              entrypoint = [ "${caller}/bin/caller" ];
            };
            layers = [
              (nix2containerPkgs.nix2container.buildLayer { deps = [ caller ]; })
            ];
          };

          greeter = buildGoModule {
            pname = "greeter";
            version = "0.1.0";
            src = ./services;
            vendorHash = null;
            env.CGO_ENABLED = 0;
            ldflags = [
              "-s"
              "-w"
            ];
            subPackages = [ "cmd/greeter" ];
          };

          greeter-image = nix2containerPkgs.nix2container.buildImage {
            name = "greeter";
            tag = "latest";
            config = {
              entrypoint = [ "${greeter}/bin/greeter" ];
            };
            layers = [
              (nix2containerPkgs.nix2container.buildLayer { deps = [ greeter ]; })
            ];
          };

          gateway = buildGoModule {
            pname = "gateway";
            version = "0.1.0";
            src = ./services;
            vendorHash = null;
            env.CGO_ENABLED = 0;
            ldflags = [
              "-s"
              "-w"
            ];
            subPackages = [ "cmd/gateway" ];
          };

          gateway-image = nix2containerPkgs.nix2container.buildImage {
            name = "gateway";
            tag = "latest";
            config = {
              entrypoint = [ "${gateway}/bin/gateway" ];
            };
            layers = [
              (nix2containerPkgs.nix2container.buildLayer { deps = [ gateway ]; })
            ];
          };
        in
        {
          devenv.shells.default = {
            devenv.root = devenvRoot;
            imports = [ ./devenv.nix ];
          };

          treefmt = {
            projectRootFile = "flake.nix";
            programs = import ./treefmt-programs.nix;
          };

          # Nixidy environments
          legacyPackages.nixidyEnvs = {
            local = inputs.nixidy.lib.mkEnv {
              inherit pkgs;
              charts = inputs.nixhelm.chartsDerivations.${system};
              modules = [ ./deploy/nixidy/env/local.nix ];
            };
            prod = inputs.nixidy.lib.mkEnv {
              inherit pkgs;
              charts = inputs.nixhelm.chartsDerivations.${system};
              modules = [ ./deploy/nixidy/env/prod.nix ];
            };
          };

          # Nixidy CLI
          packages.nixidy = inputs.nixidy.packages.${system}.default;

          # Microservices
          packages.caller = caller;
          packages.caller-image = caller-image;
          packages.greeter = greeter;
          packages.greeter-image = greeter-image;
          packages.gateway = gateway;
          packages.gateway-image = gateway-image;
        };
    };
}
