#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

EXTERNAL_API_DIR="$SCRIPT_DIR"/../api/core/v1alpha1
INTERNAL_API_DIR="$SCRIPT_DIR"/../internal/apis/core

rsync -a --exclude="{doc.go zz_generated.deepcopy.go}" "$EXTERNAL_API_DIR"/ "$INTERNAL_API_DIR"

function sed-all() {
  exp="$1"
  find "$INTERNAL_API_DIR" -maxdepth 1 -name "*.go" -exec sed -i -E "$exp" {} +
}

function sed-single() {
  f="$1"
  exp="$2"
  sed -i -E "$exp" "$INTERNAL_API_DIR/$f"
}

sed-all 's/[[:space:]]+`json:[^$]*//g'
sed-all 's/package v1alpha1/package core/g'
sed-all 's/Version: "v1alpha1"/Version: runtime.APIVersionInternal/g'
sed-all 's/Package v1alpha1/Package core/g'
sed-all 's/github.com\/onmetal\/onmetal-api-net\/api\/v1alpha1/github.com\/onmetal\/onmetal-api-net\/internal\/apis\/core/g'
sed-all 's/v1alpha1/core/g'
sed-all '/metav1\.AddToGroupVersion\(scheme, SchemeGroupVersion\)/d'

sed-single "register.go" '/metav1 "k8s.io\/apimachinery\/pkg\/apis\/meta\/v1"/d'

goimports -w "$INTERNAL_API_DIR"
