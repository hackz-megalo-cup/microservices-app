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
          inherit (pkgs) lib;
          nix2containerPkgs = inputs.nix2container.packages.${system};
          nodejs = pkgs.nodejs_24;
          repoSrc = lib.cleanSourceWith {
            src = ./.;
            filter =
              path: type:
              let
                pathStr = toString path;
              in
              lib.cleanSourceFilter path type
              && !(lib.hasInfix "/node_modules/" pathStr || lib.hasSuffix "/node_modules" pathStr)
              && !(lib.hasInfix "/dist/" pathStr || lib.hasSuffix "/dist" pathStr)
              && !(lib.hasSuffix ".tsbuildinfo" pathStr)
              && !(lib.hasSuffix ".DS_Store" pathStr)
              && !(lib.hasInfix "/.agents/" pathStr || lib.hasSuffix "/.agents" pathStr);
          };
          nodeServicesRoot = repoSrc + "/node-services";
          frontendRoot = repoSrc + "/frontend";
          nodeServicesPackageLock = builtins.fromJSON (
            builtins.readFile (nodeServicesRoot + "/package-lock.json")
          );
          nodeServiceNodeModules =
            name:
            pkgs.importNpmLock.buildNodeModules {
              npmRoot = nodeServicesRoot;
              package = builtins.fromJSON (builtins.readFile (nodeServicesRoot + "/${name}/package.json"));
              packageLock = nodeServicesPackageLock;
              inherit nodejs;
              derivationArgs = {
                pname = "${name}-node-modules";
                version = "0.1.0";
              };
            };
          frontendNodeModules = pkgs.importNpmLock.buildNodeModules {
            npmRoot = frontendRoot;
            inherit nodejs;
            derivationArgs = {
              pname = "frontend-node-modules";
              version = "0.1.0";
            };
          };

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

          buildNodeService =
            name: nodeModules:
            pkgs.stdenv.mkDerivation {
              pname = name;
              version = "0.1.0";
              src = repoSrc;
              dontBuild = true;
              installPhase = ''
                runHook preInstall
                mkdir -p $out/app
                cp -r node-services/${name} $out/app/${name}
                cp -r node-services/shared $out/app/shared
                cp -r ${nodeModules}/node_modules $out/app/node_modules
                runHook postInstall
              '';
            };

          buildNodeServiceRunner =
            name: package:
            pkgs.writeShellApplication {
              name = "${name}-run";
              runtimeInputs = [ nodejs ];
              text = ''
                cd ${package}/app/${name}
                if [ -n "''${DATABASE_URL:-}" ]; then
                  ../node_modules/.bin/node-pg-migrate up --database-url "''${DATABASE_URL}" --migrations-dir ./migrations
                fi
                exec ${nodejs}/bin/node --import ./tracing.js server.js
              '';
            };

          buildNodeServiceImage =
            name: package:
            let
              runner = buildNodeServiceRunner name package;
            in
            nix2containerPkgs.nix2container.buildImage {
              inherit name;
              tag = "latest";
              config = {
                entrypoint = [ "${runner}/bin/${name}-run" ];
              };
              layers = [
                (nix2containerPkgs.nix2container.buildLayer {
                  deps = [
                    package
                    runner
                  ];
                })
              ];
            };

          auth-service = buildNodeService "auth-service" (nodeServiceNodeModules "auth-service");
          auth-service-image = buildNodeServiceImage "auth-service" auth-service;

          custom-lang-service = buildNodeService "custom-lang-service" (
            nodeServiceNodeModules "custom-lang-service"
          );
          custom-lang-service-image = buildNodeServiceImage "custom-lang-service" custom-lang-service;

          frontend-assets = pkgs.stdenv.mkDerivation {
            pname = "frontend-assets";
            version = "0.1.0";
            src = repoSrc;
            nativeBuildInputs = [ nodejs ];
            buildPhase = ''
              runHook preBuild
              ln -s ${frontendNodeModules}/node_modules frontend/node_modules
              pushd frontend
              npm run build
              popd
              runHook postBuild
            '';
            installPhase = ''
              runHook preInstall
              mkdir -p $out/share/frontend $out/etc/nginx
              cp -r frontend/dist/. $out/share/frontend/
              cp frontend/nginx.conf $out/etc/nginx/server.conf
              runHook postInstall
            '';
          };

          frontend-nginx-conf = pkgs.writeText "frontend-nginx.conf" ''
            user root;
            worker_processes 1;
            error_log /dev/stderr info;
            pid /tmp/nginx.pid;

            events {}

            http {
              access_log /dev/stdout;
              include ${pkgs.nginx}/conf/mime.types;
              default_type application/octet-stream;
              sendfile on;

              server {
                listen 80;
                server_name _;
                root ${frontend-assets}/share/frontend;
                index index.html;

                location / {
                  try_files $uri $uri/ /index.html;
                }
              }
            }
          '';

          frontend-runner = pkgs.writeShellApplication {
            name = "frontend-run";
            runtimeInputs = [ pkgs.nginx ];
            text = ''
              exec ${pkgs.nginx}/bin/nginx -c ${frontend-nginx-conf} -g 'daemon off;'
            '';
          };

          frontend-image = nix2containerPkgs.nix2container.buildImage {
            name = "frontend";
            tag = "latest";
            config = {
              entrypoint = [ "${frontend-runner}/bin/frontend-run" ];
            };
            layers = [
              (nix2containerPkgs.nix2container.buildLayer {
                deps = [
                  frontend-assets
                  frontend-runner
                  pkgs.dockerTools.fakeNss
                  pkgs.nginx
                ];
              })
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
          packages.auth-service = auth-service;
          packages.auth-service-image = auth-service-image;
          packages.custom-lang-service = custom-lang-service;
          packages.custom-lang-service-image = custom-lang-service-image;
          packages.frontend = frontend-assets;
          packages.frontend-image = frontend-image;
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
