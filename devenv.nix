{
  pkgs,
  lib,
  ...
}:

{
  # Disable devenv container outputs (shadow package is Linux-only, breaks nix flake check on macOS)
  containers = lib.mkForce { };
  packages = [
    pkgs.nh
    pkgs.nix-output-monitor

    # Git
    pkgs.git

    # Kubernetes
    pkgs.kind
    pkgs.kubectl
    pkgs.kubernetes-helm

    # Go
    pkgs.go_1_26

    # Protobuf / gRPC
    pkgs.buf
    pkgs.protobuf
    pkgs.grpcurl

    # Multi-language microservices
    pkgs.nodejs_24

    # Local K8s development
    pkgs.tilt
    pkgs.watchexec

    # Container image operations
    pkgs.skopeo

    # JSON manipulation (used by new-service.sh)
    pkgs.jq

    # Nix tooling
    pkgs.nix-tree
    pkgs.nurl
  ];

  treefmt = {
    enable = true;
    config.programs = import ./treefmt-programs.nix;
  };

  git-hooks.hooks = {
    treefmt.enable = true;

    golangci-lint = {
      enable = true;
      name = "golangci-lint";
      entry = "bash -c 'export PATH=\"$DEVENV_ROOT/.devenv/go-bin:$PATH\" && cd services && golangci-lint run --modules-download-mode=mod'";
      files = "\\.go$";
      excludes = [
        "^services/gen/go/"
        "^services/vendor/"
      ];
      language = "system";
      pass_filenames = false;
    };

    gofmt = {
      enable = true;
      name = "goimports";
      entry = "bash -c 'export PATH=\"$DEVENV_ROOT/.devenv/go-bin:$PATH\" && cd services && find . -name \"*.go\" -not -path \"./vendor/*\" -not -path \"./gen/*\" | xargs goimports -l -w'";
      files = "\\.go$";
      excludes = [
        "^services/gen/go/"
        "^services/vendor/"
      ];
      language = "system";
      pass_filenames = false;
    };

    biome = {
      enable = true;
      name = "biome check (frontend)";
      entry = "bash -c 'cd frontend && npx biome check src/'";
      files = "\\.(ts|tsx)$";
      language = "system";
      pass_filenames = false;
    };

    biome-node = {
      enable = true;
      name = "biome check (node-services)";
      entry = "bash -c 'cd node-services && npx biome check --write .'";
      files = "^node-services/.*\\.js$";
      language = "system";
      pass_filenames = false;
    };

    go-test = {
      enable = true;
      name = "go test";
      entry = "bash -c 'cd services && go test -mod=mod ./...'";
      files = "\\.go$";
      excludes = [
        "^services/gen/go/"
        "^services/vendor/"
      ];
      language = "system";
      pass_filenames = false;
    };
  };

  scripts = {
    fmt.exec = ''
      echo "=== Formatting all ==="
      export PATH="$DEVENV_ROOT/.devenv/go-bin:$PATH"
      (cd services && goimports -l -w . && golangci-lint fmt ./...)
      (cd frontend && npx biome format --write src/)
      (cd node-services && npx biome check --write .)
      treefmt --no-cache
      echo "=== Done. Staging formatted files ==="
      git add -u
      echo "Ready to commit."
    '';
    lint.exec = ''
      echo "=== Linting all ==="
      export PATH="$DEVENV_ROOT/.devenv/go-bin:$PATH"
      (cd services && golangci-lint run ./...)
      (cd frontend && npx biome check src/)
      (cd node-services && npx biome check .)
      echo "=== Done ==="
    '';
    gen-manifests.exec = ''
      bash "$DEVENV_ROOT/scripts/gen-manifests.sh"
    '';
    load-microservice-images.exec = ''
      bash "$DEVENV_ROOT/scripts/load-microservice-images.sh"
    '';
    watch-manifests.exec = ''
      echo "Watching nixidy modules for changes..."
      watchexec --exts nix --restart -- bash -lc 'bash scripts/gen-manifests.sh && kubectl apply -f deploy/manifests/'
    '';
    nix-check.exec = ''
      SYSTEM="$(nix eval --raw --impure --expr 'builtins.currentSystem')"
      echo "Evaluating nix..."
      nix eval ".#legacyPackages.''${SYSTEM}.nixidyEnvs.local.environmentPackage" --raw >/dev/null \
        && echo "✓ nix eval OK" \
        || echo "✗ nix eval FAILED"
    '';
    buf-check.exec = ''
      buf lint
      if git cat-file -e main:proto 2>/dev/null; then
        buf breaking --against '.git#branch=main'
      else
        echo "Skipping buf breaking: proto baseline does not exist on main yet"
      fi
    '';
    debug-k8s.exec = ''
      echo "=== Pod status ==="
      kubectl get pods -A
      echo "=== Recent events ==="
      kubectl get events -A --sort-by=.lastTimestamp | tail -10
    '';
    debug-grpc.exec = ''
      echo "=== Greeter gRPC check ==="
      grpcurl -plaintext localhost:8080 list
      echo "=== Gateway gRPC check ==="
      grpcurl -plaintext localhost:8082 list
    '';
    test-smoke.exec = ''
      bash "$DEVENV_ROOT/scripts/smoke-test.sh"
    '';
    fix-chart-hash.exec = ''
      bash "$DEVENV_ROOT/scripts/fix-chart-hash.sh"
    '';
    sync-vendor.exec = ''
      bash "$DEVENV_ROOT/scripts/sync-vendor.sh"
    '';
    new-service.exec = ''
      bash "$DEVENV_ROOT/scripts/new-service.sh" "$@"
    '';
  };

  enterShell = ''
    # Go tools directory (not managed by nix - needs Go 1.26 built binaries)
    export GOBIN="$DEVENV_ROOT/.devenv/go-bin"
    export PATH="$GOBIN:$PATH"

    # golangci-lint: official binary (pre-built with Go 1.26, nixpkgs lags behind)
    if [ ! -x "$GOBIN/golangci-lint" ]; then
      echo "Installing golangci-lint (official binary)..."
      mkdir -p "$GOBIN"
      curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
        | sh -s -- -b "$GOBIN" latest 2>/dev/null \
        && echo "golangci-lint installed" \
        || echo "Warning: golangci-lint install failed (network issue?), skipping"
    fi

    # goimports: install via go install
    if [ ! -x "$GOBIN/goimports" ]; then
      echo "Installing goimports..."
      go install golang.org/x/tools/cmd/goimports@latest 2>/dev/null
    fi

    # protoc-gen-go: protobuf Go code generator
    if [ ! -x "$GOBIN/protoc-gen-go" ]; then
      echo "Installing protoc-gen-go..."
      go install google.golang.org/protobuf/cmd/protoc-gen-go@latest 2>/dev/null
    fi

    # protoc-gen-connect-go: Connect RPC Go code generator
    if [ ! -x "$GOBIN/protoc-gen-connect-go" ]; then
      echo "Installing protoc-gen-connect-go..."
      go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest 2>/dev/null
    fi

    # frontend: install npm deps (includes protoc-gen-es, protoc-gen-connect-query)
    if [ ! -x "$DEVENV_ROOT/frontend/node_modules/.bin/protoc-gen-es" ]; then
      echo "Installing frontend npm dependencies..."
      (cd "$DEVENV_ROOT/frontend" && npm install --no-audit --no-fund 2>/dev/null)
    fi

    echo "microservice-app dev environment loaded"
    echo ""
    echo "Available commands:"
    echo "  fmt              : Format all (Go + TS + Node + Nix) and git add -u"
    echo "  lint             : Lint all (golangci-lint + Biome frontend/node)"
    echo "  gen-manifests    : Regenerate nixidy manifests into deploy/manifests/"
    echo "  load-microservice-images  : Build + load all microservice images into kind"
    echo "  watch-manifests  : Watch nixidy modules and apply changes"
    echo "  nix-check        : Fast nix expression sanity check"
    echo "  buf-check        : Run buf lint + breaking checks"
    echo "  debug-k8s        : Kubernetes pod/event debug"
    echo "  debug-grpc       : grpcurl checks for local services"
    echo "  fix-chart-hash   : Auto-fix empty chartHash in nixidy modules"
    echo "  test-smoke       : Smoke test for health and RPC endpoints"
    echo "  sync-vendor      : Sync Go vendor/ with go.mod (tidy + vendor + stage)"
    echo "  new-service      : Scaffold a new service (new-service <go|custom> <name> [port])"
    echo ""
    echo "Microservice tools:"
    echo "  buf                      : Protobuf tooling (lint, breaking, generate)"
    echo "  grpcurl                  : gRPC CLI debugger"
    echo "  tilt                     : Local K8s dev environment"
    echo "  skopeo                   : Container image operations"
  '';
}
