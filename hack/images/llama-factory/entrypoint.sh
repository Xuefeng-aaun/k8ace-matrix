#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

APP_ROOT="${LLAMAFACTORY_ROOT:-/opt/LLaMA-Factory}"
PORT="${PORT:-7860}"
HOST="${HOST:-0.0.0.0}"
export GRADIO_SERVER_NAME="${GRADIO_SERVER_NAME:-${HOST}}"
export GRADIO_SERVER_PORT="${GRADIO_SERVER_PORT:-${PORT}}"

if [[ ! -d "${APP_ROOT}" ]]; then
    echo "[llama-factory] app root not found: ${APP_ROOT}" >&2
    exit 1
fi

cd "${APP_ROOT}"
exec llamafactory-cli webui
