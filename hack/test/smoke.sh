#!/usr/bin/env bash
# smoke.sh - 镜像冒烟测试入口
#
# 用法：
#   smoke.sh [L0|L1|L2] [app_name] [hardware]
#
# 测试边界：
# - L0：基础环境检查，确认 python/pip 这类通用工具可用。
# - L1：应用级轻量检查，按 runtime/cli/service/workspace 做最小验证。
# - L2：硬件 runtime 级检查。当前只覆盖 NVIDIA CUDA，不做 torch、不加载模型、不跑推理。

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
            note_skip "runtime app '${APP}' has no dedicated L1 rule yet"
            ;;
    esac
}

l1_cli() {
    case "${APP}" in
        llama_factory|llamafactory)
            run_check "llamafactory-cli version/help" sh -c "llamafactory-cli version >/dev/null 2>&1 || llamafactory-cli --help >/dev/null"
            ;;
        *)
            note_skip "cli app '${APP}' has no dedicated L1 rule yet"
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
            note_skip "service app '${APP}' has no dedicated L1 rule yet"
            ;;
    esac
}

l1_workspace() {
    note_skip "workspace app '${APP}' has no dedicated L1 rule yet"
}

l2_nvidia_cuda_runtime() {
    echo "[smoke] L2 runtime scope: NVIDIA CUDA driver/runtime visibility only"

    run_check "nvidia-smi available" command -v nvidia-smi
    run_check "nvidia gpu visible" sh -c "nvidia-smi -L | grep -q '^GPU '"
    run_check "nvidia driver version readable" sh -c "nvidia-smi --query-gpu=driver_version --format=csv,noheader | head -n1 | grep -Eq '^[0-9]+(\\.[0-9]+)+'"

    # libcuda.so.1 来自宿主 NVIDIA driver 注入，是容器能调用 CUDA Driver API 的关键。
    run_check "libcuda.so.1 visible" sh -c "ldconfig -p 2>/dev/null | grep -q 'libcuda.so.1' || test -e /usr/lib/x86_64-linux-gnu/libcuda.so.1 || test -e /usr/local/cuda/compat/libcuda.so.1"

    # libcudart 是 CUDA Runtime API。部分 app 镜像可能通过 pip/conda wheel 携带 runtime，
    # 也可能来自 CUDA base image；这里仅检查它在容器内可被动态链接器看到。
    run_check "CUDA runtime library visible" sh -c "ldconfig -p 2>/dev/null | grep -Eq 'libcudart\\.so|libcuda\\.so' || find /usr /opt /workspace -name 'libcudart.so*' -o -name 'libcuda.so*' 2>/dev/null | head -n1 | grep -q ."
}

run_check "python3 available" python3 --version
run_check "pip available" sh -c "pip --version || pip3 --version"

if [[ "${LEVEL}" == "L0" ]]; then
    echo "[smoke] L0 checks done: ${PASS} passed, ${FAIL} failed"
    exit "${FAIL}"
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
        note_skip "unknown app '${APP}', no L1 rule"
        ;;
esac

if [[ "${LEVEL}" == "L1" ]]; then
    echo "[smoke] L1 checks done: ${PASS} passed, ${FAIL} failed"
    exit "${FAIL}"
fi

case "${HW}" in
    nvidia)
        l2_nvidia_cuda_runtime
        ;;
    *)
        note_skip "L2 runtime check only supports nvidia for now, got '${HW}'"
        ;;
esac

echo "[smoke] ${LEVEL} checks done: ${PASS} passed, ${FAIL} failed"
exit "${FAIL}"
