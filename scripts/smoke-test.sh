#!/usr/bin/env bash
set -euo pipefail

echo "==> Checking deployments"
kubectl -n microservices rollout status deploy/gateway --timeout=180s >/dev/null
kubectl -n microservices rollout status deploy/custom-lang-service --timeout=180s >/dev/null
kubectl -n microservices rollout status deploy/auth-service --timeout=180s >/dev/null
kubectl -n microservices rollout status deploy/frontend --timeout=180s >/dev/null

cleanup() {
  if [[ -n "${PF_PIDS:-}" ]]; then
    # shellcheck disable=SC2086
    kill ${PF_PIDS} >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "==> Port-forwarding services"
kubectl -n microservices port-forward svc/gateway 18082:8082 >/tmp/pf-gateway.log 2>&1 &
PF1=$!
kubectl -n microservices port-forward svc/custom-lang-service 13000:3000 >/tmp/pf-custom.log 2>&1 &
PF2=$!
kubectl -n microservices port-forward svc/auth-service 18090:8090 >/tmp/pf-auth.log 2>&1 &
PF3=$!
kubectl -n microservices port-forward svc/frontend 18081:80 >/tmp/pf-frontend.log 2>&1 &
PF4=$!
PF_PIDS="$PF1 $PF2 $PF3 $PF4"

for _ in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:18082/healthz >/dev/null 2>&1 \
    && curl -fsS http://127.0.0.1:13000/healthz >/dev/null 2>&1 \
    && curl -fsS http://127.0.0.1:18090/healthz >/dev/null 2>&1 \
    && curl -fsS http://127.0.0.1:18081/ >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "==> Health checks"
curl -fsS http://127.0.0.1:18082/healthz >/dev/null
curl -fsS http://127.0.0.1:13000/healthz >/dev/null
curl -fsS http://127.0.0.1:18090/healthz >/dev/null
curl -fsS http://127.0.0.1:18081/ >/dev/null

echo "==> gRPC checks"
grpcurl -plaintext -d '{"name":"Smoke"}' 127.0.0.1:18082 gateway.v1.GatewayService/InvokeCustom >/dev/null

echo "==> Auth check"
TOKEN="$(curl -fsS -X POST http://127.0.0.1:18090/auth/login -H 'content-type: application/json' -d '{\"email\":\"demo@example.com\",\"password\":\"password\"}' | sed -n 's/.*\"token\":\"\\([^\"]*\\)\".*/\\1/p')"
if [[ -z "$TOKEN" ]]; then
  echo "failed to get JWT token from auth-service" >&2
  exit 1
fi
curl -fsS http://127.0.0.1:18090/verify -H "authorization: Bearer $TOKEN" >/dev/null

echo "Smoke test passed"
