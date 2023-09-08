#!/usr/bin/env bash

set -euo pipefail

CONTROLLER_GEN=${CONTROLLER_GEN:-"controller-gen"}

declare cluster rbac paths output

while [[ $# -gt 0 ]]; do
  case $1 in
  cluster*)
    cluster="$1"
    cluster="${cluster#cluster=}"
    shift
    ;;
  rbac*)
    rbac="$1"
    shift
    ;;
  paths*)
    paths="$1"
    # Remove 'paths=' prefix and '/...' suffix.
    paths="${paths#paths=}"
    paths="${paths%/...}"
    shift
    ;;
  output:rbac:artifacts:config*)
    output="$1"
    output="${output#output:rbac:artifacts:config=}"
    shift
    ;;
  *)
    echo "Unknown option/arg $1"
    exit 1
    ;;
  esac
done

tmp_dir="$(mktemp -d)"
trap 'rm -rf $tmp_dir' EXIT

combined="$tmp_dir/combined.go"

echo "package extracted" > "$combined"
echo "module extracted" > "$tmp_dir/go.mod"

grep -rh "//+cluster=$cluster" "$paths" | sed -e "s/cluster=$cluster://" >> "$combined"

"$CONTROLLER_GEN" "$rbac" "paths=$tmp_dir" "output:rbac:artifacts:config=$output"
sed -i 's/ClusterRole/Role/g' "$output/role.yaml"
sed -i '/creationTimestamp: null/a\  namespace: system' "$output/role.yaml"
