#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

k8ace::gpu::has-nvidia-smi() {
	command -v nvidia-smi >/dev/null 2>&1
}

k8ace::gpu::print-nvidia() {
	if k8ace::gpu::has-nvidia-smi; then
		nvidia-smi || true
		return 0
	fi
	k8ace::log::warn "nvidia-smi not found"
	return 0
}

