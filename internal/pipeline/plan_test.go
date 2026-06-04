package pipeline

import (
	"strings"
	"testing"

	"k8ace-matrix/internal/matrix"
)

func TestBuildPlansAppAliasDoesNotImplicitlyBuildBase(t *testing.T) {
	m := testMatrix()

	plans, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"app"},
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

	wantStages := []string{"app_image", "app_test"}
	if len(gotStages) != len(wantStages) {
		t.Fatalf("resolved stages = %v, want %v", gotStages, wantStages)
	}
	for i := range wantStages {
		if gotStages[i] != wantStages[i] {
			t.Fatalf("resolved stages = %v, want %v", gotStages, wantStages)
		}
	}

	if _, ok := taskByStage["host_driver"]; ok {
		t.Fatalf("host_driver should be disabled during demo validation")
	}
	if _, ok := taskByStage["base_image"]; ok {
		t.Fatalf("app alias should not include base_image")
	}
	if _, ok := taskByStage["base_test"]; ok {
		t.Fatalf("app alias should not include base_test")
	}
	if deps := taskByStage["app_image"].DependsOn; len(deps) != 0 {
		t.Fatalf("app_image depends_on = %v, want none because base is explicit in batch plan", deps)
	}
	if deps := taskByStage["app_test"].DependsOn; len(deps) != 1 || deps[0] != taskByStage["app_image"].Name {
		t.Fatalf("app_test depends_on = %v, want [%s]", deps, taskByStage["app_image"].Name)
	}
}

func TestBuildPlansAppTestStageRunsSmokeHelperWhenAvailable(t *testing.T) {
	m := testMatrix()

	plans, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"app_test"},
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

	testTask, ok := taskByStage["app_test"]
	if !ok {
		t.Fatalf("app_test stage not generated; stages=%v", taskByStage)
	}
	if len(testTask.TestCommands) != 1 {
		t.Fatalf("app_test commands = %v, want one smoke command", testTask.TestCommands)
	}
	if !strings.Contains(testTask.TestCommands[0], "/opt/k8ace/hack/test/smoke.sh") {
		t.Fatalf("app_test command should call smoke helper, got %q", testTask.TestCommands[0])
	}
	if !strings.Contains(testTask.TestCommands[0], "'L3' 'demo' 'nvidia' 'runtime'") {
		t.Fatalf("app_test command should pass level+app+hardware+type to smoke helper, got %q", testTask.TestCommands[0])
	}
	if len(testTask.TestResourceLimits) != 0 || len(testTask.TestResourceRequests) != 0 {
		t.Fatalf("app_test should not request GPU resources in demo mode")
	}
}

func TestBuildPlansBaseAliasExpandsToBaseAndBaseTest(t *testing.T) {
	m := testMatrix()

	plans, err := BuildPlans(m, Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "demo",
		AppVersion:    "1.0.0",
		Variant:       "demo-cuda",
		Stages:        []string{"base"},
		VersionSuffix: "dev",
		Builder:       "kaniko",
		KanikoImage:   "kaniko:test",
	})
	if err != nil {
		t.Fatalf("BuildPlans() error = %v", err)
	}

	var gotStages []string
	for _, task := range plans[0].Tasks {
		gotStages = append(gotStages, task.Stage)
	}
	wantStages := []string{"base_image", "base_test"}
	if len(gotStages) != len(wantStages) {
		t.Fatalf("resolved stages = %v, want %v", gotStages, wantStages)
	}
	for i := range wantStages {
		if gotStages[i] != wantStages[i] {
			t.Fatalf("resolved stages = %v, want %v", gotStages, wantStages)
		}
	}

	taskByStage := map[string]Task{}
	for _, task := range plans[0].Tasks {
		taskByStage[task.Stage] = task
	}
	if deps := taskByStage["base_image"].DependsOn; len(deps) != 0 {
		t.Fatalf("base_image depends_on = %v, want none while host_driver is disabled", deps)
	}
	if deps := taskByStage["base_test"].DependsOn; len(deps) != 1 || deps[0] != taskByStage["base_image"].Name {
		t.Fatalf("base_test depends_on = %v, want [%s]", deps, taskByStage["base_image"].Name)
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
	if _, ok := taskByStage["base_image"]; ok {
		t.Fatalf("explicit app_image stage should not implicitly include base_image")
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
				{Name: "base_test", DependsOn: []string{"base_image"}},
				{Name: "app_image", DependsOn: []string{"base_test"}},
				{Name: "app_test", DependsOn: []string{"app_image"}},
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
					Type:     "runtime",
					Versions: []string{"1.0.0"},
					Variants: []matrix.AppVariant{
						{
							Name:        "demo-cuda",
							Accelerator: "cuda-124",
							BaseRef:     "cuda_base",
							Hardware:    []string{"nvidia"},
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
