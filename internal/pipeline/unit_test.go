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

func TestDeriveUnitsUsesExplicitBaseTagSuffix(t *testing.T) {
	m := &matrix.Matrix{
		RegistryPrefix: "k8ace",
		BaseImageMatrix: map[string]matrix.BaseImageDef{
			"cuda_base": {
				Source: "nvidia/cuda",
				Variants: []matrix.BaseVariant{
					{
						TagSuffix:       "cuda12.4-devel-ubuntu22.04",
						K8AceCompatible: []string{"nvidia"},
						Extra: map[string]any{
							"status":       "recommended",
							"cuda_version": "12.4",
						},
					},
					{
						TagSuffix:       "cuda12.2-devel-ubuntu22.04",
						K8AceCompatible: []string{"nvidia"},
						Extra: map[string]any{
							"status":       "stable",
							"cuda_version": "12.2",
						},
					},
				},
			},
		},
		ApplicationMatrix: matrix.ApplicationMatrix{
			"llm_aigc": {
				"demo": {
					Versions: []string{"1.0"},
					Variants: []matrix.AppVariant{
						{
							Name:          "demo-cuda122",
							BaseRef:       "cuda_base",
							BaseTagSuffix: "cuda12.2-devel-ubuntu22.04",
							Hardware:      []string{"nvidia"},
							BuildArgs: map[string]string{
								"CUDA_VERSION": "${base.cuda_version}",
							},
						},
					},
				},
			},
		},
	}

	units, err := DeriveUnits(m, Selection{
		Hardwares:  []string{"nvidia"},
		AppName:    "demo",
		AppVersion: "1.0",
		Variant:    "demo-cuda122",
	})
	if err != nil {
		t.Fatalf("DeriveUnits: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("units = %d, want 1", len(units))
	}
	if got, want := units[0].BaseVariant.TagSuffix, "cuda12.2-devel-ubuntu22.04"; got != want {
		t.Fatalf("BaseVariant.TagSuffix = %q, want %q", got, want)
	}
	if got, want := units[0].BuildArgs["CUDA_VERSION"], "12.2"; got != want {
		t.Fatalf("CUDA_VERSION = %q, want %q", got, want)
	}
}

func TestDeriveUnitsPrefersRecommendedBaseVariantWhenTagSuffixOmitted(t *testing.T) {
	m := &matrix.Matrix{
		RegistryPrefix: "k8ace",
		BaseImageMatrix: map[string]matrix.BaseImageDef{
			"cuda_base": {
				Source: "nvidia/cuda",
				Variants: []matrix.BaseVariant{
					{
						TagSuffix:       "cuda12.2-devel-ubuntu22.04",
						K8AceCompatible: []string{"nvidia"},
						Extra:           map[string]any{"status": "stable"},
					},
					{
						TagSuffix:       "cuda12.4-devel-ubuntu22.04",
						K8AceCompatible: []string{"nvidia"},
						Extra:           map[string]any{"status": "recommended"},
					},
				},
			},
		},
		ApplicationMatrix: matrix.ApplicationMatrix{
			"llm_aigc": {
				"demo": {
					Versions: []string{"1.0"},
					Variants: []matrix.AppVariant{
						{
							Name:     "demo-cuda",
							BaseRef:  "cuda_base",
							Hardware: []string{"nvidia"},
						},
					},
				},
			},
		},
	}

	units, err := DeriveUnits(m, Selection{
		Hardwares:  []string{"nvidia"},
		AppName:    "demo",
		AppVersion: "1.0",
		Variant:    "demo-cuda",
	})
	if err != nil {
		t.Fatalf("DeriveUnits: %v", err)
	}
	if got, want := units[0].BaseVariant.TagSuffix, "cuda12.4-devel-ubuntu22.04"; got != want {
		t.Fatalf("BaseVariant.TagSuffix = %q, want %q", got, want)
	}
}

func TestDeriveUnitsSelectionBaseTagSuffixOverridesAppVariant(t *testing.T) {
	m := &matrix.Matrix{
		RegistryPrefix: "k8ace",
		BaseImageMatrix: map[string]matrix.BaseImageDef{
			"cuda_base": {
				Source: "nvidia/cuda",
				Variants: []matrix.BaseVariant{
					{TagSuffix: "cuda12.4-devel-ubuntu22.04", K8AceCompatible: []string{"nvidia"}},
					{TagSuffix: "cuda12.2-devel-ubuntu22.04", K8AceCompatible: []string{"nvidia"}},
				},
			},
		},
		ApplicationMatrix: matrix.ApplicationMatrix{
			"llm_aigc": {
				"demo": {
					Versions: []string{"1.0"},
					Variants: []matrix.AppVariant{
						{
							Name:          "demo-cuda",
							BaseRef:       "cuda_base",
							BaseTagSuffix: "cuda12.4-devel-ubuntu22.04",
							Hardware:      []string{"nvidia"},
						},
					},
				},
			},
		},
	}

	units, err := DeriveUnits(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0",
		Variant:       "demo-cuda",
		BaseTagSuffix: "cuda12.2-devel-ubuntu22.04",
	})
	if err != nil {
		t.Fatalf("DeriveUnits: %v", err)
	}
	if got, want := units[0].BaseVariant.TagSuffix, "cuda12.2-devel-ubuntu22.04"; got != want {
		t.Fatalf("BaseVariant.TagSuffix = %q, want %q", got, want)
	}
}
