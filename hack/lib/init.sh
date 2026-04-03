#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

K8ACE_HACK_ROOT="${K8ACE_HACK_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)}"
K8ACE_ROOT="${K8ACE_ROOT:-$(cd "${K8ACE_HACK_ROOT}/.." && pwd -P)}"

source "${K8ACE_HACK_ROOT}/lib/logging.sh"
source "${K8ACE_HACK_ROOT}/lib/util.sh"
source "${K8ACE_HACK_ROOT}/lib/gpu.sh"

