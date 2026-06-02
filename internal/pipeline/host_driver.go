package pipeline

import (
	"fmt"
	"regexp"
	"strings"
)

func buildHostDriverSpec(u BuildUnit) HostDriverSpec {
	if strings.EqualFold(strings.TrimSpace(u.Hardware), "nvidia") {
		return buildNvidiaHostDriverSpec(u)
	}

	return HostDriverSpec{
		Image: "alpine:3.20",
		Commands: []string{
			fmt.Sprintf("echo '[host_driver] unsupported hardware=%s'; exit 1", shellQuote(u.Hardware)),
		},
	}
}

func buildNvidiaHostDriverSpec(u BuildUnit) HostDriverSpec {
	accelerator := normalizeCudaAccelerator(u.Accelerator, u.BaseVariant.TagSuffix, getBaseString(u, "cuda_short"), getBaseString(u, "cuda_version"))
	minDriver := getBaseString(u, "min_driver_linux")
	if minDriver == "" {
		minDriver = minDriverForCuda(accelerator)
	}

	image := strings.TrimSpace(u.BaseSourceImage)
	if image == "" {
		image = "nvidia/cuda:12.4.1-base-ubuntu22.04"
	}

	return HostDriverSpec{
		Image: image,
		Commands: []string{
			nvidiaHostDriverScript(accelerator, minDriver),
		},
		ResourceLimits: map[string]string{
			"nvidia.com/gpu": "1",
		},
		ResourceRequests: map[string]string{
			"nvidia.com/gpu": "1",
		},
	}
}

func getBaseString(u BuildUnit, key string) string {
	v, _ := u.BaseVariant.GetString(key)
	return strings.TrimSpace(v)
}

func minDriverForCuda(accelerator string) string {
	switch strings.ToLower(strings.TrimSpace(accelerator)) {
	case "cuda-118":
		return "520.61.05"
	case "cuda-121":
		return "530.30.02"
	case "cuda-124":
		return "550.54.15"
	case "cuda-131":
		return "590.44.01"
	default:
		return ""
	}
}

func normalizeCudaAccelerator(values ...string) string {
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		value = strings.ReplaceAll(value, "_", "-")
		if strings.HasPrefix(value, "cuda-") {
			return value
		}
		if strings.HasPrefix(value, "cuda") {
			suffix := strings.TrimPrefix(value, "cuda")
			suffix = strings.TrimLeft(suffix, "-")
			if normalized := normalizeCudaDigits(suffix); normalized != "" {
				return normalized
			}
		}
		if normalized := normalizeCudaDigits(value); normalized != "" {
			return normalized
		}
	}
	return ""
}

var cudaVersionRe = regexp.MustCompile(`^([0-9]+)(?:\.([0-9]+))?`)

func normalizeCudaDigits(value string) string {
	value = strings.TrimSpace(value)
	m := cudaVersionRe.FindStringSubmatch(value)
	if len(m) == 0 {
		return ""
	}
	major := m[1]
	minor := m[2]
	if minor == "" && len(major) >= 3 {
		minor = major[len(major)-1:]
		major = major[:len(major)-1]
	}
	if major == "" || minor == "" {
		return ""
	}
	return "cuda-" + major + minor
}

func nvidiaHostDriverScript(accelerator, minDriver string) string {
	return fmt.Sprintf(`set -eu
TARGET_ACCELERATOR=%s
MIN_DRIVER=%s

echo "[host_driver] vendor=nvidia runtime=cuda target=${TARGET_ACCELERATOR}"

if [ -z "${TARGET_ACCELERATOR}" ] || [ -z "${MIN_DRIVER}" ]; then
  echo "[host_driver] ERROR: cannot determine target CUDA accelerator or minimum driver"
  echo "[host_driver] Please set application runtime/accelerator or base min_driver_linux"
  exit 1
fi

if ! command -v nvidia-smi >/dev/null 2>&1; then
  echo "[host_driver] ERROR: nvidia-smi not found; NVIDIA driver/device plugin is unavailable in this pod"
  exit 1
fi

GPU_LIST="$(nvidia-smi -L || true)"
echo "${GPU_LIST}"
if [ -z "${GPU_LIST}" ] || ! echo "${GPU_LIST}" | grep -q '^GPU '; then
  echo "[host_driver] ERROR: no NVIDIA GPU visible in this pod"
  exit 1
fi

DRIVER_VERSION="$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -n1 | tr -d '[:space:]')"
if [ -z "${DRIVER_VERSION}" ]; then
  echo "[host_driver] ERROR: cannot read NVIDIA driver version"
  exit 1
fi

version_ge() {
  awk -v a="$1" -v b="$2" 'BEGIN {
    n = split(a, A, ".");
    m = split(b, B, ".");
    max = n > m ? n : m;
    for (i = 1; i <= max; i++) {
      ai = A[i] + 0;
      bi = B[i] + 0;
      if (ai > bi) exit 0;
      if (ai < bi) exit 1;
    }
    exit 0;
  }'
}

echo "[host_driver] detected_driver=${DRIVER_VERSION} required_min_driver=${MIN_DRIVER}"

if ! version_ge "${DRIVER_VERSION}" "${MIN_DRIVER}"; then
  echo "[host_driver] ERROR: driver ${DRIVER_VERSION} is too old for ${TARGET_ACCELERATOR}; later smoke tests would be meaningless"
  exit 1
fi

echo "[host_driver] PASS: NVIDIA driver is compatible with ${TARGET_ACCELERATOR}"
`, shellQuote(accelerator), shellQuote(minDriver))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
