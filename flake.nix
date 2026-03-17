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
    extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw= hackz-megalo-cup.cachix.org-1:21679nTC27hKWUad5U5+MGAxkw1+8y0/9RGAbuvlmUY=";
    extra-substituters = "https://devenv.cachix.org https://hackz-megalo-cup.cachix.org";
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
          buildGoService =
            name:
            buildGoModule {
              pname = name;
              version = "0.1.0";
              src = ./services;
              vendorHash = "sha256-fzgNa+0Y5biTxqcK6VelnCzzIElzxeiLb653GhKKR7E=";
              env.CGO_ENABLED = 0;
              ldflags = [
                "-s"
                "-w"
              ];
              subPackages = [ "cmd/${name}" ];
            };

          buildGoServiceImage =
            name: package:
            nix2containerPkgs.nix2container.buildImage {
              inherit name;
              tag = "latest";
              config = {
                entrypoint = [ "${package}/bin/${name}" ];
              };
              layers = [
                (nix2containerPkgs.nix2container.buildLayer { deps = [ package ]; })
              ];
            };

          caller = buildGoService "caller";
          caller-image = buildGoServiceImage "caller" caller;

          gateway = buildGoService "gateway";
          gateway-image = buildGoServiceImage "gateway" gateway;

          greeter = buildGoService "greeter";
          greeter-image = buildGoServiceImage "greeter" greeter;

          item = buildGoService "item";
          item-image = buildGoServiceImage "item" item;

          masterdata = buildGoService "masterdata";
          masterdata-image = buildGoServiceImage "masterdata" masterdata;

          projector = buildGoService "projector";
          projector-image = buildGoServiceImage "projector" projector;

          raid-lobby = buildGoService "raid-lobby";
          raid-lobby-image = buildGoServiceImage "raid-lobby" raid-lobby;
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
          packages.gateway = gateway;
          packages.gateway-image = gateway-image;
          packages.greeter = greeter;
          packages.greeter-image = greeter-image;
          packages.item = item;
          packages.item-image = item-image;
          packages.masterdata = masterdata;
          packages.masterdata-image = masterdata-image;
          packages.projector = projector;
          packages.projector-image = projector-image;
          packages.raid-lobby = raid-lobby;
          packages.raid-lobby-image = raid-lobby-image;
        };
    };
}
