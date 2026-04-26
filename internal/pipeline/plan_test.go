package pipeline

import (
	"testing"

	"k8ace-matrix/internal/matrix"
)

func TestBuildPlansResolvesStageDependencies(t *testing.T) {
	m := testMatrix()

	plans, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"app_image"},
		VersionSuffix: "dev",
		Builder:       "kaniko",
		KanikoImage:   "kaniko:test",
	})
	if err != nil {
		t.Fatalf("BuildPlans() error = %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("BuildPlans() plans = %d, want 1", len(plans))
	}

	gotStages := make([]string, 0, len(plans[0].Tasks))
	taskByStage := map[string]Task{}
	for _, task := range plans[0].Tasks {
		gotStages = append(gotStages, task.Stage)
		taskByStage[task.Stage] = task
	}

	wantStages := []string{"host_driver", "base_image", "app_image"}
	if len(gotStages) != len(wantStages) {
		t.Fatalf("resolved stages = %v, want %v", gotStages, wantStages)
	}
	for i := range wantStages {
		if gotStages[i] != wantStages[i] {
			t.Fatalf("resolved stages = %v, want %v", gotStages, wantStages)
		}
	}

	if len(taskByStage["host_driver"].DependsOn) != 0 {
		t.Fatalf("host_driver depends_on = %v, want none", taskByStage["host_driver"].DependsOn)
	}
	if deps := taskByStage["base_image"].DependsOn; len(deps) != 1 || deps[0] != taskByStage["host_driver"].Name {
		t.Fatalf("base_image depends_on = %v, want [%s]", deps, taskByStage["host_driver"].Name)
	}
	if deps := taskByStage["app_image"].DependsOn; len(deps) != 1 || deps[0] != taskByStage["base_image"].Name {
		t.Fatalf("app_image depends_on = %v, want [%s]", deps, taskByStage["base_image"].Name)
	}

	if got := taskByStage["host_driver"].Kaniko.Dockerfile; got != "" {
		t.Fatalf("host_driver dockerfile = %q, want empty", got)
	}
	if got := taskByStage["host_driver"].Kaniko.Image; got != "alpine:3.20" {
		t.Fatalf("host_driver image = %q, want alpine:3.20", got)
	}
	if taskByStage["host_driver"].Kaniko.Cache.Enabled {
		t.Fatalf("host_driver cache enabled = true, want false")
	}
}

func TestBuildPlansDockerfileOverrideAppliesToSingleExplicitStage(t *testing.T) {
	m := testMatrix()

	plans, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"app_image"},
		Dockerfile:    "custom/Dockerfile",
		VersionSuffix: "dev",
		Builder:       "kaniko",
		KanikoImage:   "kaniko:test",
	})
	if err != nil {
		t.Fatalf("BuildPlans() error = %v", err)
	}

	taskByStage := map[string]Task{}
	for _, task := range plans[0].Tasks {
		taskByStage[task.Stage] = task
	}

	if got := taskByStage["app_image"].Kaniko.Dockerfile; got != "custom/Dockerfile" {
		t.Fatalf("app_image dockerfile = %q, want custom/Dockerfile", got)
	}
	if got := taskByStage["base_image"].Kaniko.Dockerfile; got == "custom/Dockerfile" {
		t.Fatalf("base_image dockerfile unexpectedly overridden: %q", got)
	}
}

func TestBuildPlansDockerfileOverrideRejectsAmbiguousStageSelection(t *testing.T) {
	m := testMatrix()

	_, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"all"},
		Dockerfile:    "custom/Dockerfile",
		VersionSuffix: "dev",
		Builder:       "kaniko",
		KanikoImage:   "kaniko:test",
	})
	if err == nil {
		t.Fatalf("BuildPlans() error = nil, want override validation error")
	}
}

func TestBuildPlansRejectsUnsupportedBuilder(t *testing.T) {
	m := testMatrix()

	_, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"app_image"},
		VersionSuffix: "dev",
		Builder:       "buildkit",
		KanikoImage:   "kaniko:test",
	})
	if err == nil {
		t.Fatalf("BuildPlans() error = nil, want unsupported builder error")
	}
}

func TestBuildPlansRespectsArgoCacheEnabled(t *testing.T) {
	m := testMatrix()
	m.CICD.ArgoWorkflows.Cache.Enabled = false

	plans, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"app_image"},
		VersionSuffix: "dev",
		Builder:       "kaniko",
		KanikoImage:   "kaniko:test",
	})
	if err != nil {
		t.Fatalf("BuildPlans() error = %v", err)
	}

	for _, task := range plans[0].Tasks {
		if task.Kaniko.Cache.Enabled {
			t.Fatalf("task %s cache enabled = true, want false", task.Name)
		}
	}
}

func testMatrix() *matrix.Matrix {
	return &matrix.Matrix{
		RegistryPrefix: "k8ace",
		BuildPipeline: matrix.BuildPipeline{
			Stages: []matrix.Stage{
				{Name: "host_driver"},
				{Name: "base_image", DependsOn: []string{"host_driver"}},
				{Name: "app_image", DependsOn: []string{"base_image"}},
				{Name: "test", DependsOn: []string{"app_image"}},
			},
			Cache: matrix.Cache{
				Type:        "registry",
				TTL:         "168h",
				KeyTemplate: "cache",
			},
		},
		BaseImageMatrix: map[string]matrix.BaseImageDef{
			"cuda_base": {
				Source: "nvidia/cuda",
				Variants: []matrix.BaseVariant{
					{
						TagSuffix:       "cuda12.4-devel-ubuntu22.04",
						K8AceCompatible: []string{"nvidia"},
					},
				},
			},
		},
		ApplicationMatrix: matrix.ApplicationMatrix{
			"demo": {
				"demo": {
					Versions: []string{"1.0.0"},
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
		CICD: matrix.CICD{
			ArgoWorkflows: matrix.ArgoWorkflows{
				Cache: matrix.ArgoCacheConfig{
					Enabled:      true,
					RepoTemplate: "{{ .RegistryPrefix }}/cache/{{ .Hardware }}/{{ .Stage }}",
				},
			},
		},
	}
}
