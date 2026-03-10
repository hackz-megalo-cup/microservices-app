# -*- mode: Python -*-
load('ext://restart_process', 'docker_build_with_restart')

use_nix = str(os.getenv('USE_NIX', 'false')).lower() == 'true'
skip_cluster_up = str(os.getenv('TILT_SKIP_CLUSTER_UP', 'false')).lower() == 'true'

# Detect host architecture for cross-compilation targeting kind cluster
_arch = str(local('uname -m', quiet=True)).strip()
go_arch = 'arm64' if _arch == 'arm64' or _arch == 'aarch64' else 'amd64'

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

watch_file('deploy/manifests')


local_resource(
    'gen-manifests',
    cmd='bash scripts/gen-manifests.sh',
    deps=[
        'flake.nix',
        'deploy/nixidy/env/local.nix',
        'deploy/nixidy/env/traefik.nix',
        'deploy/k8s/greeter.nix',
        'deploy/k8s/caller.nix',
        'deploy/k8s/gateway.nix',
        'deploy/k8s/custom-lang-service.nix',
        'deploy/k8s/auth-service.nix',
        'deploy/k8s/frontend.nix',
        'scripts/gen-manifests.sh',
    ],
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

manifests = []

# Edge routing and middleware.
manifests += find_yaml('deploy/manifests/traefik')

# Application services.
manifests += find_yaml('deploy/manifests/greeter-service')
manifests += find_yaml('deploy/manifests/caller-service')
manifests += find_yaml('deploy/manifests/gateway-service')
manifests += find_yaml('deploy/manifests/custom-lang-service')
manifests += find_yaml('deploy/manifests/auth-service')
manifests += find_yaml('deploy/manifests/frontend')
manifests += find_yaml('deploy/manifests/microservices-secrets')

namespaces = [m for m in manifests if '/Namespace-' in m]
# Gateway API CRDs are pre-installed at v1.5.0+; skip the older ones bundled in the Traefik chart.
crds = [m for m in manifests if '/CustomResourceDefinition-' in m and 'gateway-networking-k8s-io' not in m]
others = [m for m in manifests if '/Namespace-' not in m and '/CustomResourceDefinition-' not in m]
manifests = namespaces + crds + others

if manifests:
    k8s_yaml(manifests)
else:
    print('No Kubernetes manifests found yet. Wait for gen-manifests to finish.')


def go_service(name, cmd_path):
    compile_deps = [
        'services/go.mod',
        'services/go.sum',
        'services/internal/',
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
            'rm -f services/%s/build/%s && cd services && CGO_ENABLED=0 GOOS=linux GOARCH=%s go build -o %s/build/%s ./%s'
            % (name, name, go_arch, name, name, cmd_path),
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


go_service('caller', 'cmd/caller')
go_service('greeter', 'cmd/greeter')
go_service('gateway', 'cmd/gateway')
docker_build(
    'custom-lang-service',
    context='.',
    dockerfile='deploy/docker/custom-lang-service/Dockerfile',
)

docker_build(
    'auth-service',
    context='.',
    dockerfile='deploy/docker/auth-service/Dockerfile',
)

docker_build(
    'frontend',
    context='.',
    dockerfile='deploy/docker/frontend/Dockerfile',
    build_args={'VITE_API_BASE_URL': 'http://localhost:30081'},
)

if manifests:
    caller_deps = cluster_bootstrap_deps + ['gen-manifests', 'buf-generate']
    greeter_deps = cluster_bootstrap_deps + ['gen-manifests', 'buf-generate']
    gateway_deps = cluster_bootstrap_deps + ['gen-manifests', 'buf-generate']
    if not use_nix:
        caller_deps += ['caller-compile']
        greeter_deps += ['greeter-compile']
        gateway_deps += ['gateway-compile']

    k8s_resource('traefik', resource_deps=cluster_bootstrap_deps + ['gen-manifests'])
    k8s_resource('caller-service', port_forwards=8081, resource_deps=caller_deps)
    k8s_resource('greeter-service', port_forwards=8080, resource_deps=greeter_deps)
    k8s_resource('gateway', port_forwards=8082, resource_deps=gateway_deps)
    k8s_resource('custom-lang-service', port_forwards=3000, resource_deps=cluster_bootstrap_deps + ['gen-manifests'])
    k8s_resource('auth-service', port_forwards=8090, resource_deps=cluster_bootstrap_deps + ['gen-manifests'])
    k8s_resource('frontend', port_forwards=5173, resource_deps=cluster_bootstrap_deps + ['gen-manifests', 'buf-generate'])

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
