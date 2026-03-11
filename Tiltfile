# -*- mode: Python -*-
load('ext://restart_process', 'docker_build_with_restart')

use_nix = str(os.getenv('USE_NIX', 'false')).lower() == 'true'
skip_cluster_up = str(os.getenv('TILT_SKIP_CLUSTER_UP', 'false')).lower() == 'true'

# Detect host architecture for cross-compilation targeting kind cluster
_arch = str(local('uname -m', quiet=True)).strip()
go_arch = 'arm64' if _arch == 'arm64' or _arch == 'aarch64' else 'amd64'

# --- Service configuration (auto-watched by read_json) ---
services_config = read_json('tilt-services.json', default={})
go_svcs = services_config.get('go_services', {})
custom_svcs = services_config.get('custom_services', {})
special_svcs = services_config.get('special', {})

# --- Cluster bootstrap ---
cluster_bootstrap_deps = []
if not skip_cluster_up:
    local_resource(
        'cluster-up',
        cmd='kind create cluster --name microservice-app 2>/dev/null || true',
        trigger_mode=TRIGGER_MODE_AUTO,
        labels=['bootstrap'],
    )
    cluster_bootstrap_deps = ['cluster-up']


def find_yaml(path, exclude=''):
    cmd = 'find %s -name "*.yaml"' % path
    if exclude:
        cmd = '%s ! -name "%s"' % (cmd, exclude)
    cmd = cmd + ' 2>/dev/null || true'
    out = str(local(cmd, quiet=True)).strip()
    if not out:
        return []
    return [line for line in out.split('\n') if line]


# --- gen-manifests with dynamic nix deps ---
watch_file('deploy/k8s')
_nix_deps = str(local('find deploy/k8s -name "*.nix" 2>/dev/null | sort', quiet=True)).strip()
nix_files = [f for f in _nix_deps.split('\n') if f] if _nix_deps else []

gen_manifest_deps = [
    'flake.nix',
    'deploy/nixidy/env/local.nix',
    'deploy/nixidy/env/traefik.nix',
    'scripts/gen-manifests.sh',
] + nix_files

local_resource(
    'gen-manifests',
    cmd='bash scripts/gen-manifests.sh',
    deps=gen_manifest_deps,
    resource_deps=cluster_bootstrap_deps,
    trigger_mode=TRIGGER_MODE_AUTO,
    labels=['bootstrap'],
)

local_resource(
    'buf-generate',
    cmd='buf generate',
    deps=['proto/', 'buf.yaml', 'buf.gen.yaml'],
    ignore=['services/gen/go/**', 'frontend/src/gen/**'],
    resource_deps=cluster_bootstrap_deps,
    labels=['codegen'],
)

# --- Dynamic manifest discovery (exclude apps/ which contains ArgoCD Applications) ---
watch_file('deploy/manifests')
_manifest_dirs = str(local(
    'find deploy/manifests -mindepth 1 -maxdepth 1 -type d ! -name apps 2>/dev/null | sort',
    quiet=True,
)).strip()
manifests = []
for d in _manifest_dirs.split('\n'):
    if d:
        manifests += find_yaml(d)

namespaces = [m for m in manifests if '/Namespace-' in m]
# Gateway API CRDs are pre-installed at v1.5.0+; skip the older ones bundled in the Traefik chart.
crds = [m for m in manifests if '/CustomResourceDefinition-' in m and 'gateway-networking-k8s-io' not in m]
others = [m for m in manifests if '/Namespace-' not in m and '/CustomResourceDefinition-' not in m]
manifests = namespaces + crds + others

if manifests:
    k8s_yaml(manifests)
else:
    print('No Kubernetes manifests found yet. Wait for gen-manifests to finish.')


# --- Go service builder (narrowed compile_deps per service) ---
def go_service(name, cmd_path):
    snake_name = name.replace('-', '_')
    compile_deps = [
        'services/go.mod',
        'services/go.sum',
        'services/internal/platform/',
        'services/internal/%s/' % snake_name,
        'services/gen/go/',
        'services/%s/' % cmd_path,
    ]
    if use_nix:
        custom_build(
            name,
            'nix build .#%s-image && kind load docker-image %s:latest --name microservice-app' % (name, name),
            deps=compile_deps,
            skips_local_docker=True,
        )
    else:
        local_resource(
            '%s-compile' % name,
            'mkdir -p services/%s/build && rm -f services/%s/build/%s && cd services && CGO_ENABLED=0 GOOS=linux GOARCH=%s go build -o %s/build/%s ./%s'
            % (name, name, name, go_arch, name, name, cmd_path),
            deps=compile_deps,
            ignore=['services/%s/build/' % name],
            resource_deps=['buf-generate'],
            labels=['compile'],
        )
        docker_build_with_restart(
            name,
            'services/%s' % name,
            entrypoint='/%s' % name,
            dockerfile='deploy/docker/%s/Dockerfile.dev' % name,
            live_update=[sync('services/%s/build/%s' % (name, name), '/%s' % name)],
        )


# --- Build services from config ---
for name, cfg in go_svcs.items():
    go_service(name, cfg['cmd_path'])

for name, cfg in custom_svcs.items():
    docker_build(
        name,
        context='.',
        dockerfile='deploy/docker/%s/Dockerfile' % name,
    )

for name, cfg in special_svcs.items():
    build_args = cfg.get('build_args', {})
    if build_args:
        docker_build(
            name,
            context='.',
            dockerfile='deploy/docker/%s/Dockerfile' % name,
            build_args=build_args,
        )
    else:
        docker_build(
            name,
            context='.',
            dockerfile='deploy/docker/%s/Dockerfile' % name,
        )

# --- k8s resources (resource_deps derived from service type, not JSON) ---
if manifests:
    k8s_resource('traefik', resource_deps=cluster_bootstrap_deps + ['gen-manifests'])

    for name, cfg in go_svcs.items():
        k8s_name = cfg.get('k8s_resource', '%s-service' % name)
        port = cfg.get('port', 8080)
        deps = cluster_bootstrap_deps + ['gen-manifests', 'buf-generate']
        if not use_nix:
            deps += ['%s-compile' % name]
        k8s_resource(k8s_name, port_forwards=port, resource_deps=deps)

    for name, cfg in custom_svcs.items():
        k8s_name = cfg.get('k8s_resource', name)
        port = cfg.get('port', 8080)
        k8s_resource(k8s_name, port_forwards=port,
                     resource_deps=cluster_bootstrap_deps + ['gen-manifests'])

    for name, cfg in special_svcs.items():
        k8s_name = cfg.get('k8s_resource', name)
        port = cfg.get('port', 8080)
        extra_deps = cfg.get('extra_resource_deps', [])
        k8s_resource(k8s_name, port_forwards=port,
                     resource_deps=cluster_bootstrap_deps + ['gen-manifests'] + extra_deps)

local_resource(
    'health-check',
    cmd='''
      echo "=== Cluster ===" && kubectl cluster-info 2>&1 | head -2
      echo "=== Pods ===" && kubectl get pods -A --no-headers 2>&1 | grep -v Running | head -10
      echo "=== Services ===" && kubectl get svc -A --no-headers 2>&1 | head -10
      echo "=== Greeter health ===" && curl -sf http://localhost:8080/healthz >/dev/null && echo ok || echo ng
      echo "=== Gateway health ===" && curl -sf http://localhost:8082/healthz >/dev/null && echo ok || echo ng
      echo "=== Frontend ===" && curl -sf http://localhost:5173 >/dev/null && echo ok || echo ng
      echo "=== Traefik ===" && curl -sf http://localhost:30081 >/dev/null && echo ok || echo ng
    ''',
    trigger_mode=TRIGGER_MODE_MANUAL,
    labels=['debug'],
)
