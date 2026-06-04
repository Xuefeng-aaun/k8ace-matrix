#!/usr/bin/env bash
# smoke.sh - 镜像冒烟测试入口
#
# 用法：
#   smoke.sh [L0|L1|L2|L3] [app_name] [hardware] [app_type]
#
# 当前生产规则：
# - L0：所有镜像都必须通过基础环境检查。
# - L1：确认应用类型和最低测试契约存在。
# - L2：runtime/cli 执行最高有效检查；service 只做启动材料检查。
# - L3：service 必须真实启动并 curl 成功；runtime/cli 复用 L2 的最高有效检查。
#
# 三类应用的验收口径：
# - service：启动服务，等待端口，curl 成功。
# - runtime：直接 import 关键包成功。
# - cli：执行版本/帮助命令并成功返回。

set -o errexit
set -o nounset
set -o pipefail

LEVEL="${1:-L0}"
APP="${2:-}"
HW="${3:-}"
ARG_TYPE="${4:-}"

PASS=0
FAIL=0
SERVICE_PID=""
SERVICE_LOG="/tmp/k8ace-service-smoke.log"

cleanup() {
    if [[ -n "${SERVICE_PID}" ]] && kill -0 "${SERVICE_PID}" >/dev/null 2>&1; then
        kill "${SERVICE_PID}" >/dev/null 2>&1 || true
        wait "${SERVICE_PID}" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT

run_check() {
    local name="$1"
    shift
    local safe_name
    safe_name="$(echo "${name}" | tr -c '[:alnum:]_.-' '_')"
    local log="/tmp/k8ace-smoke-${safe_name}.log"

    echo "[smoke] checking: ${name}"
    if "$@" >"${log}" 2>&1; then
        echo "[smoke] PASS: ${name}"
        PASS=$((PASS + 1))
    else
        echo "[smoke] FAIL: ${name}"
        echo "[smoke] ---- ${name} output ----"
        cat "${log}" || true
        echo "[smoke] ---- end output ----"
        FAIL=$((FAIL + 1))
    fi
}

note_skip() {
    echo "[smoke] SKIP: $1"
}

app_type_fallback() {
    case "$1" in
        vllm|comfyui|sd_webui|fooocus|r_stats|jupyter_sdr|llama_factory|llamafactory)
            echo "service"
            ;;
        xtuner|yolo|alphafold|cuda_samples|openmpi|oneapi|gromacs|gnuradio)
            echo "cli"
            ;;
        pytorch|tensorflow|jax|paddlepaddle|langchain|stable_diffusion|sd|pandas_stack|pyspark|rapids|opencv|detectron2|biopython|blender|julia)
            echo "runtime"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

resolve_type() {
    if [[ -n "${K8ACE_SMOKE_TYPE:-}" ]]; then
        echo "${K8ACE_SMOKE_TYPE}"
        return
    fi
    if [[ -n "${ARG_TYPE}" ]]; then
        echo "${ARG_TYPE}"
        return
    fi
    app_type_fallback "${APP}"
}

default_service_command() {
    case "${APP}" in
        comfyui)
            echo "/opt/k8ace/hack/images/comfyui/entrypoint.sh"
            ;;
        llama_factory|llamafactory)
            echo "/opt/k8ace/hack/images/llama-factory/entrypoint.sh"
            ;;
        *)
            echo ""
            ;;
    esac
}

default_cli_command() {
    case "${APP}" in
        llama_factory|llamafactory)
            echo "llamafactory-cli version || llamafactory-cli --help"
            ;;
        yolo)
            echo "yolo --help"
            ;;
        *)
            echo ""
            ;;
    esac
}

default_runtime_imports() {
    case "${APP}" in
        stable_diffusion|sd)
            echo "torch,diffusers,transformers,accelerate"
            ;;
        pytorch)
            echo "torch,torchvision,torchaudio"
            ;;
        langchain)
            echo "langchain,langchain_community"
            ;;
        pandas_stack)
            echo "pandas,numpy,scipy,sklearn"
            ;;
        opencv)
            echo "cv2,numpy"
            ;;
        *)
            echo ""
            ;;
    esac
}

runtime_import_check() {
    if [[ -n "${K8ACE_SMOKE_RUNTIME_CMD:-}" ]]; then
        run_check "runtime custom command" sh -c "${K8ACE_SMOKE_RUNTIME_CMD}"
        return
    fi

    local imports="${K8ACE_SMOKE_IMPORTS:-$(default_runtime_imports)}"
    if [[ -z "${imports}" ]]; then
        note_skip "runtime app '${APP}' has no import contract"
        FAIL=$((FAIL + 1))
        return
    fi

    run_check "runtime imports: ${imports}" python3 - "${imports}" <<'PY'
import importlib
import sys

mods = [x.strip() for x in sys.argv[1].split(",") if x.strip()]
for mod in mods:
    imported = importlib.import_module(mod)
    print(f"{mod}: {getattr(imported, '__version__', 'ok')}")
PY

    case "${APP}" in
        stable_diffusion|sd)
            run_check "import StableDiffusion3Pipeline" python3 - <<'PY'
from diffusers import StableDiffusion3Pipeline
print("StableDiffusion3Pipeline: ok")
PY
            ;;
    esac
}

cli_command_check() {
    local cmd="${K8ACE_SMOKE_CLI_CMD:-$(default_cli_command)}"
    if [[ -z "${cmd}" ]]; then
        note_skip "cli app '${APP}' has no command contract"
        FAIL=$((FAIL + 1))
        return
    fi
    run_check "cli command: ${cmd}" sh -c "${cmd}"
}

service_contract_check() {
    local cmd="${K8ACE_SMOKE_SERVICE_CMD:-$(default_service_command)}"
    if [[ -z "${cmd}" ]]; then
        note_skip "service app '${APP}' has no start command contract"
        FAIL=$((FAIL + 1))
        return
    fi
    run_check "service start command exists" sh -c "command -v ${cmd%% *} >/dev/null 2>&1 || test -x ${cmd%% *} || test -f ${cmd%% *}"
}

service_curl_check() {
    local cmd="${K8ACE_SMOKE_SERVICE_CMD:-$(default_service_command)}"
    local port="${K8ACE_SMOKE_PORT:-${PORT:-}}"
    local path="${K8ACE_SMOKE_HEALTH_PATH:-/}"
    local timeout="${K8ACE_SMOKE_TIMEOUT_SECONDS:-120}"

    if [[ -z "${cmd}" ]]; then
        note_skip "service app '${APP}' has no start command contract"
        FAIL=$((FAIL + 1))
        return
    fi
    if [[ -z "${port}" ]]; then
        case "${APP}" in
            comfyui) port="8188" ;;
            llama_factory|llamafactory) port="7860" ;;
            *) port="80" ;;
        esac
    fi

    local url="${K8ACE_SMOKE_HEALTH_URL:-http://127.0.0.1:${port}${path}}"
    echo "[smoke] starting service: ${cmd}"
    echo "[smoke] waiting for URL: ${url}"

    rm -f "${SERVICE_LOG}"
    sh -c "${cmd}" >"${SERVICE_LOG}" 2>&1 &
    SERVICE_PID="$!"

    for _ in $(seq 1 "${timeout}"); do
        if curl -fsS "${url}" >/dev/null 2>&1; then
            echo "[smoke] PASS: service curl ${url}"
            PASS=$((PASS + 1))
            return
        fi
        if ! kill -0 "${SERVICE_PID}" >/dev/null 2>&1; then
            echo "[smoke] FAIL: service exited before curl succeeded"
            echo "[smoke] ---- service output ----"
            cat "${SERVICE_LOG}" || true
            echo "[smoke] ---- end output ----"
            FAIL=$((FAIL + 1))
            return
        fi
        sleep 1
    done

    echo "[smoke] FAIL: service did not become ready within ${timeout}s"
    echo "[smoke] ---- service output ----"
    cat "${SERVICE_LOG}" || true
    echo "[smoke] ---- end output ----"
    FAIL=$((FAIL + 1))
}

run_l1_contract() {
    case "${TYPE}" in
        service)
            service_contract_check
            ;;
        runtime)
            local imports="${K8ACE_SMOKE_IMPORTS:-$(default_runtime_imports)}"
            if [[ -n "${K8ACE_SMOKE_RUNTIME_CMD:-}" || -n "${imports}" ]]; then
                echo "[smoke] PASS: runtime contract exists"
                PASS=$((PASS + 1))
            else
                note_skip "runtime app '${APP}' has no import contract"
                FAIL=$((FAIL + 1))
            fi
            ;;
        cli)
            local cmd="${K8ACE_SMOKE_CLI_CMD:-$(default_cli_command)}"
            if [[ -n "${cmd}" ]]; then
                echo "[smoke] PASS: cli contract exists"
                PASS=$((PASS + 1))
            else
                note_skip "cli app '${APP}' has no command contract"
                FAIL=$((FAIL + 1))
            fi
            ;;
        *)
            note_skip "unknown app type '${TYPE}'"
            FAIL=$((FAIL + 1))
            ;;
    esac
}

run_highest_effective_check() {
    case "${TYPE}" in
        service)
            service_curl_check
            ;;
        runtime)
            runtime_import_check
            ;;
        cli)
            cli_command_check
            ;;
        *)
            note_skip "unknown app type '${TYPE}'"
            FAIL=$((FAIL + 1))
            ;;
    esac
}

run_check "python3 available" python3 --version
run_check "pip available" sh -c "pip --version || pip3 --version"

if [[ "${LEVEL}" == "L0" ]]; then
    echo "[smoke] L0 checks done: ${PASS} passed, ${FAIL} failed"
    exit "${FAIL}"
fi

TYPE="$(resolve_type)"
echo "[smoke] app=${APP} type=${TYPE} hardware=${HW} level=${LEVEL}"

run_l1_contract

case "${LEVEL}" in
    L1)
        ;;
    L2|L3)
        run_highest_effective_check
        ;;
    *)
        echo "[smoke] unknown level: ${LEVEL}" >&2
        FAIL=$((FAIL + 1))
        ;;
esac

echo "[smoke] ${LEVEL} checks done: ${PASS} passed, ${FAIL} failed"
exit "${FAIL}"
