#!/usr/bin/env bash
set -euo pipefail

BOOTSTRAP="kafka:9092"
TOPICS=(
  "orders:3:1"
  "orders_dead_letter:1:1"
)

for i in $(seq 1 30); do
  echo "Checking kafka availability (attempt $i)..."
  if kafka-topics --bootstrap-server ${BOOTSTRAP} --list >/dev/null 2>&1; then
    echo "Kafka is up"
    break
  fi
  sleep 2
done

for spec in "${TOPICS[@]}"; do
  IFS=':' read -r topic partitions replication <<< "$spec"
  echo "Creating topic $topic (partitions=$partitions, replication=$replication) if not exists..."
  kafka-topics --bootstrap-server ${BOOTSTRAP} --create --topic "$topic" \
    --partitions "$partitions" --replication-factor "$replication" || true
done

echo "Topics creation finished."
