#!/bin/bash

if [ $# -lt 1 ]; then
  echo "Usage: $0 <number_of_nodes>"
  exit 1
fi

N=$1

if [ "$N" -lt 2 ]; then
  echo "Need at least 2 nodes"
  exit 1
fi

# Génération des IDs
IDS=()
for ((i=1; i<=N; i++)); do
  IDS+=("Site$i")
done

echo "Starting ring with $N nodes: ${IDS[*]}"

# Lancement des nodes
for id in "${IDS[@]}"; do
  echo "Starting $id"

  go run main.go "$id" "${IDS[@]}" > "${id}.log" 2>&1 &

done

echo "All nodes started"
echo "Logs in Site*.log"
wait