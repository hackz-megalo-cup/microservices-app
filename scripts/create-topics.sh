#!/usr/bin/env bash
set -euo pipefail

BROKER="${KAFKA_BROKERS:-localhost:19092}"

topics=(
  "greeting.created:3"
  "call.completed:3"
  "invocation.created:3"
  "user.registered:3"
  "greeting.created.dlq:1"
  "call.completed.dlq:1"
  "invocation.created.dlq:1"
  "user.registered.dlq:1"
)

for entry in "${topics[@]}"; do
  topic="${entry%%:*}"
  partitions="${entry##*:}"
  echo "Creating topic: $topic (partitions: $partitions)"
  docker compose exec redpanda rpk topic create "$topic" -p "$partitions" 2>/dev/null || true
done

echo "All topics created."
