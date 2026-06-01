#!/usr/bin/env bash
# smoke.sh - ?????????????
# ??: smoke.sh [L0|L1|L2] [app_name] [hardware]
# ??:
# - L0 ??????????
# - L1 ????????????runtime / cli / service / workspace
# - L2 ?????????
# - ?? docker run ??????????????? L1 ??

set -o errexit
set -o nounset
set -o pipefail

LEVEL="${1:-L0}"
APP="${2:-}"
HW="${3:-}"

PASS=0
FAIL=0

run_check() {
    local name="$1"
    shift
    echo "[smoke] checking: ${name}"
    if "$@" >/dev/null 2>&1; then
        echo "[smoke] PASS: ${name}"
        PASS=$((PASS + 1))
    else
        echo "[smoke] FAIL: ${name}"
        FAIL=$((FAIL + 1))
    fi
}

note_skip() {
    echo "[smoke] SKIP: $1"
}

app_type() {
    case "$1" in
        pytorch|tensorflow|jax|paddlepaddle|langchain|stable_diffusion|sd|pandas_stack|pyspark|rapids|opencv|detectron2|biopython)
            echo "runtime"
            ;;
        llama_factory|llamafactory|xtuner|yolo|alphafold|cuda_samples|openmpi|oneapi|gromacs)
            echo "cli"
            ;;
        vllm|comfyui|sd_webui|fooocus)
            echo "service"
            ;;
        ros_noetic|ros2_humble|rl_robotics|gnuradio|jupyter_sdr|blender|r_stats|julia)
            echo "workspace"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

l1_runtime() {
    case "${APP}" in
        stable_diffusion|sd)
            run_check "diffusers version" python3 -c 'import diffusers; print(diffusers.__version__)'
            run_check "import StableDiffusion3Pipeline" python3 -c 'from diffusers import StableDiffusion3Pipeline; print("ok")'
            ;;
        *)
            note_skip "runtime app '${APP}' ???????????? L1 ??"
            ;;
    esac
}

l1_cli() {
    case "${APP}" in
        llama_factory|llamafactory)
            run_check "llamafactory-cli version/help" sh -c "llamafactory-cli version >/dev/null 2>&1 || llamafactory-cli --help >/dev/null"
            ;;
        *)
            note_skip "cli app '${APP}' ???????????? L1 ??"
            ;;
    esac
}

l1_service() {
    case "${APP}" in
        comfyui)
            run_check "comfyui entrypoint exists" test -f /opt/k8ace/hack/images/comfyui/entrypoint.sh
            run_check "import folder_paths" python3 -c 'import folder_paths; print("ok")'
            ;;
        *)
            note_skip "service app '${APP}' ???????????? L1 ??"
            ;;
    esac
}

l1_workspace() {
    note_skip "workspace app '${APP}' ???????????? L1 ??"
}

run_check "python3 available" python3 --version
run_check "pip available" sh -c "pip --version || pip3 --version"

if [[ "${LEVEL}" == "L0" ]]; then
    echo "[smoke] L0 checks done: ${PASS} passed, ${FAIL} failed"
    exit ${FAIL}
fi

TYPE="$(app_type "${APP}")"
echo "[smoke] app=${APP} type=${TYPE}"

case "${TYPE}" in
    runtime)
        l1_runtime
        ;;
    cli)
        l1_cli
        ;;
    service)
        l1_service
        ;;
    workspace)
        l1_workspace
        ;;
    *)
        note_skip "????? '${APP}' ?????? L1"
        ;;
esac

if [[ "${LEVEL}" == "L1" ]]; then
    echo "[smoke] L1 checks done: ${PASS} passed, ${FAIL} failed"
    exit ${FAIL}
fi

case "${HW}" in
    nvidia)
        run_check "nvidia-smi" nvidia-smi
        run_check "torch.cuda.is_available" python3 -c "import torch; assert torch.cuda.is_available(), 'CUDA not available'"
        ;;
    *)
        note_skip "L2 ????? '${HW}' ????"
        ;;
esac

echo "[smoke] ${LEVEL} checks done: ${PASS} passed, ${FAIL} failed"
exit ${FAIL}
