#!/usr/bin/env bash

set -e

BASEDIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
export TERM="xterm-256color"

bold="$(tput bold)"
red="$(tput setaf 1)"
green="$(tput setaf 2)"
normal="$(tput sgr0)"

shopt -s globstar
for kustomization in "$BASEDIR"/../config/**/kustomization.yaml; do
  [ -e "$kustomization" ] || continue
  path="$(dirname "$kustomization")"
  dir="$(realpath --relative-to "$BASEDIR"/.. "$path")"
  echo "${bold}Validating $dir${normal}"
  if ! kustomize_output="$(kustomize build "$path" 2>&1)"; then
    echo "${red}Kustomize build $dir failed:${normal}"
    echo "$kustomize_output"
    exit 1
  fi
  echo "${green}Successfully validated $dir${normal}"
done
