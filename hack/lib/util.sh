#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

k8ace::util::require-command() {
	local cmd="${1}"
	if ! command -v "${cmd}" >/dev/null 2>&1; then
		k8ace::log::error "missing required command: ${cmd}"
		return 1
	fi
}

k8ace::util::ensure-dir() {
	local d="${1}"
	mkdir -p "${d}"
}

k8ace::util::is-true() {
	case "${1:-}" in
	true|TRUE|True|1|yes|YES|Yes|y|Y) return 0 ;;
	*) return 1 ;;
	esac
}

