#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

CLUSTER_NAME="microservice-app"

echo "==> Building caller Docker image..."
docker build -t "caller:latest" -f "${REPO_ROOT}/deploy/docker/caller/Dockerfile" "${REPO_ROOT}"

echo "==> Loading caller into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "caller:latest" --name "${CLUSTER_NAME}"

echo "==> Building greeter Docker image..."
docker build -t "greeter:latest" -f "${REPO_ROOT}/deploy/docker/greeter/Dockerfile" "${REPO_ROOT}"

echo "==> Loading greeter into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "greeter:latest" --name "${CLUSTER_NAME}"

echo "==> Building gateway Docker image..."
docker build -t "gateway:latest" -f "${REPO_ROOT}/deploy/docker/gateway/Dockerfile" "${REPO_ROOT}"

echo "==> Loading gateway into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "gateway:latest" --name "${CLUSTER_NAME}"

echo "==> Building custom-lang-service Docker image..."
docker build -t "custom-lang-service:latest" -f "${REPO_ROOT}/deploy/docker/custom-lang-service/Dockerfile" "${REPO_ROOT}"

echo "==> Loading custom-lang-service into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "custom-lang-service:latest" --name "${CLUSTER_NAME}"

echo "==> Building auth-service Docker image..."
docker build -t "auth-service:latest" -f "${REPO_ROOT}/deploy/docker/auth-service/Dockerfile" "${REPO_ROOT}"

echo "==> Loading auth-service into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "auth-service:latest" --name "${CLUSTER_NAME}"

echo "==> Building frontend Docker image..."
docker build -t "frontend:latest" -f "${REPO_ROOT}/deploy/docker/frontend/Dockerfile" "${REPO_ROOT}"

echo "==> Loading frontend into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "frontend:latest" --name "${CLUSTER_NAME}"

echo "==> Done. All microservice images loaded."
