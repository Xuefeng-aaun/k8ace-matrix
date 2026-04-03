#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

k8ace::log::stamp() {
	date -u +%Y-%m-%dT%H:%M:%SZ
}

k8ace::log::info() {
	echo "I$(k8ace::log::stamp) $*"
}

k8ace::log::warn() {
	echo "W$(k8ace::log::stamp) $*" 1>&2
}

k8ace::log::error() {
	echo "E$(k8ace::log::stamp) $*" 1>&2
}

