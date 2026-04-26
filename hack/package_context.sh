#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_PATH="${1:-${ROOT_DIR}/dist/context/context.tar.gz}"
OUTPUT_DIR="$(dirname "${OUTPUT_PATH}")"

mkdir -p "${OUTPUT_DIR}"

tar -C "${ROOT_DIR}" \
  --exclude=.git \
  --exclude=.gocache \
  --exclude=.gomodcache \
  --exclude=.gotmp \
  --exclude=.verifydist \
  --exclude=dist/context \
  --exclude=dist/argo \
  -zcf "${OUTPUT_PATH}" .

printf '%s\n' "${OUTPUT_PATH}"
