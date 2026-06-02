package pipeline

import (
	"testing"

	"k8ace-matrix/internal/matrix"

	"gopkg.in/yaml.v3"
)

func TestBuildBaseSourceImageUsesUpstreamTagWhenPresent(t *testing.T) {
	baseVar := matrix.BaseVariant{
		TagSuffix: "cuda12.4-devel-ubuntu22.04",
		Extra: map[string]any{
			"upstream_tag": "12.4.1-devel-ubuntu22.04",
		},
	}

	got := buildBaseSourceImage("nvidia/cuda", baseVar)
	want := "nvidia/cuda:12.4.1-devel-ubuntu22.04"
	if got != want {
		t.Fatalf("buildBaseSourceImage() = %q, want %q", got, want)
	}
}

func TestBuildBaseSourceImageUpstreamSourceOverridesSource(t *testing.T) {
	baseVar := matrix.BaseVariant{
		TagSuffix: "cuda12.4-devel-ubuntu22.04",
		Extra: map[string]any{
			"upstream_source": "172.20.47.182:5000/k8ace/upstream/nvidia-cuda",
			"upstream_tag":    "12.4.1-devel-ubuntu22.04",
		},
	}

	got := buildBaseSourceImage("nvidia/cuda", baseVar)
	want := "172.20.47.182:5000/k8ace/upstream/nvidia-cuda:12.4.1-devel-ubuntu22.04"
	if got != want {
		t.Fatalf("buildBaseSourceImage() = %q, want %q", got, want)
	}
}

func TestBuildBaseSourceImageFallsBackToTagSuffix(t *testing.T) {
	baseVar := matrix.BaseVariant{
		TagSuffix: "cuda12.4-devel-ubuntu22.04",
	}

	got := buildBaseSourceImage("nvidia/cuda", baseVar)
	want := "nvidia/cuda:cuda12.4-devel-ubuntu22.04"
	if got != want {
		t.Fatalf("buildBaseSourceImage() = %q, want %q", got, want)
	}
}

func TestNestedApplicationMatrixDerivesPreciseBaseRef(t *testing.T) {
	raw := `
registry_prefix: k8ace
base_image_matrix:
  cuda124_base:
    source: nvidia/cuda
    variants:
      - tag_suffix: cuda12.4-devel-ubuntu22.04
        k8ace_compatible: [nvidia]
application_matrix:
  practical_apps:
    comfyui:
      type: service
      versions:
        "0.22.0":
          runtimes:
            cuda:
              cuda-124:
                name: comfyui-service-cuda124
                base_ref: cuda124_base
                hardware: [nvidia]
`

	var m matrix.Matrix
	if err := yaml.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	units, err := DeriveUnits(&m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "comfyui",
		AppVersion:    "0.22.0",
		Variant:       "comfyui-service-cuda124",
		VersionSuffix: "dev",
	})
	if err != nil {
		t.Fatalf("DeriveUnits() error = %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("DeriveUnits() units = %d, want 1", len(units))
	}

	unit := units[0]
	if unit.BaseRef != "cuda124_base" {
		t.Fatalf("BaseRef = %q, want cuda124_base", unit.BaseRef)
	}
	if unit.BaseImageDest != "k8ace/cuda124_base-cuda12.4-devel-ubuntu22.04" {
		t.Fatalf("BaseImageDest = %q", unit.BaseImageDest)
	}
	if unit.AppImageDest != "k8ace/comfyui0.22.0-nvidia-comfyui-service-cuda124-dev" {
		t.Fatalf("AppImageDest = %q", unit.AppImageDest)
	}
}
