#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

CLUSTER_NAME="microservice-app"

echo "==> Building gateway Docker image..."
docker build -t "gateway:latest" -f "${REPO_ROOT}/deploy/docker/gateway/Dockerfile" "${REPO_ROOT}"

echo "==> Loading gateway into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "gateway:latest" --name "${CLUSTER_NAME}"

echo "==> Building auth-service Docker image..."
docker build -t "auth-service:latest" -f "${REPO_ROOT}/deploy/docker/auth-service/Dockerfile" "${REPO_ROOT}"

echo "==> Loading auth-service into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "auth-service:latest" --name "${CLUSTER_NAME}"

echo "==> Building frontend Docker image..."
docker build -t "frontend:latest" -f "${REPO_ROOT}/deploy/docker/frontend/Dockerfile" "${REPO_ROOT}"

echo "==> Loading frontend into kind cluster '${CLUSTER_NAME}'..."
kind load docker-image "frontend:latest" --name "${CLUSTER_NAME}"

echo "==> Done. All microservice images loaded."
