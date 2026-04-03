#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

K8ACE_HACK_ROOT="${K8ACE_HACK_ROOT:-/opt/k8ace/hack}"
source "${K8ACE_HACK_ROOT}/lib/init.sh"

APP_ROOT="${COMFYUI_ROOT:-/opt/comfyui}"
PORT="${PORT:-8188}"
LISTEN="${LISTEN:-0.0.0.0}"

if [[ ! -d "${APP_ROOT}" ]]; then
	k8ace::log::error "comfyui root not found: ${APP_ROOT}"
	exit 1
fi

k8ace::gpu::print-nvidia

cd "${APP_ROOT}"

if [[ -f "./main.py" ]]; then
	exec python3 -u ./main.py --listen "${LISTEN}" --port "${PORT}"
fi

k8ace::log::error "comfyui entrypoint could not find main.py"
exit 1

