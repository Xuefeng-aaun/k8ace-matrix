#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

K8ACE_HACK_ROOT="${K8ACE_HACK_ROOT:-/opt/k8ace/hack}"
source "${K8ACE_HACK_ROOT}/lib/init.sh"

APP_ROOT="${SD_WEBUI_ROOT:-/opt/sd-webui}"
PORT="${PORT:-7860}"
LISTEN="${LISTEN:-0.0.0.0}"

if [[ ! -d "${APP_ROOT}" ]]; then
	k8ace::log::error "sd-webui root not found: ${APP_ROOT}"
	exit 1
fi

k8ace::gpu::print-nvidia

cd "${APP_ROOT}"

if [[ -f "./launch.py" ]]; then
	exec python3 -u ./launch.py --listen "${LISTEN}" --port "${PORT}"
fi

if [[ -f "./webui.sh" ]]; then
	exec bash ./webui.sh --listen --port "${PORT}"
fi

k8ace::log::error "sd-webui entrypoint could not find launch.py or webui.sh"
exit 1

