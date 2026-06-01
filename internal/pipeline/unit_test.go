package pipeline

import (
	"testing"

	"k8ace-matrix/internal/matrix"
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
