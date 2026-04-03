#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

K8ACE_HACK_ROOT="${K8ACE_HACK_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)}"
K8ACE_ROOT="${K8ACE_ROOT:-$(cd "${K8ACE_HACK_ROOT}/.." && pwd -P)}"

source "${K8ACE_HACK_ROOT}/lib/init.sh"

MATRIX_PATH="${MATRIX_PATH:-${K8ACE_ROOT}/images-matrix.yaml}"
HARDWARE="${HARDWARE:-nvidia}"
APP_NAME="${APP_NAME:-sd_webui}"
APP_VERSION="${APP_VERSION:-1.10.0}"
VARIANT="${VARIANT:-sd-webui-cuda}"
OUT_DIR="${OUT_DIR:-${K8ACE_ROOT}/dist/argo}"

k8ace::util::require-command go

k8ace::log::info "matrix=${MATRIX_PATH}"
k8ace::log::info "hardware=${HARDWARE} app=${APP_NAME} version=${APP_VERSION} variant=${VARIANT}"

cd "${K8ACE_ROOT}"

go run ./cmd/matrix-ci render \
  --matrix "${MATRIX_PATH}" \
  --hardware "${HARDWARE}" \
  --app-name "${APP_NAME}" \
  --app-version "${APP_VERSION}" \
  --variant "${VARIANT}" \
  --out-dir "${OUT_DIR}"

k8ace::log::info "workflow output dir=${OUT_DIR}"

APP_DOCKERFILE="dockerfiles/app_image/${HARDWARE}/${APP_NAME}/${APP_VERSION}/${VARIANT}/Dockerfile"
k8ace::log::info "app dockerfile=${APP_DOCKERFILE}"
if [[ -f "${APP_DOCKERFILE}" ]]; then
	k8ace::log::info "app dockerfile head:"
	head -n 25 "${APP_DOCKERFILE}"
else
	k8ace::log::warn "app dockerfile not found"
fi

