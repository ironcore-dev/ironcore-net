#!/usr/bin/env bash

set -euo pipefail

INPUT_ROLE="$1"
OUTPUT_CLUSTER_ROLE="$2"
NAME="$3"

cp "$INPUT_ROLE" "$OUTPUT_CLUSTER_ROLE"
sed -i "s/name: .*/name: $NAME/g" "$OUTPUT_CLUSTER_ROLE"
sed -i 's/Role/ClusterRole/g' "$OUTPUT_CLUSTER_ROLE"
sed -i '/namespace: system/d' "$OUTPUT_CLUSTER_ROLE"
