package render

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"k8ace-matrix/internal/matrix"
	"k8ace-matrix/internal/pipeline"
)

func TestBuildWorkflowYAML_Golden(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", ".."))

	mPath := filepath.Join(root, "testdata", "minimal-matrix.yaml")
	gPath := filepath.Join(root, "testdata", "workflowtemplate.golden.yaml")

	m, err := matrix.Load(mPath)
	if err != nil {
		t.Fatalf("load matrix: %v", err)
	}

	p, err := pipeline.BuildPlan(m, pipeline.Selection{
		Hardwares:     []string{"nvidia"},
		AppName:       "pytorch",
		AppVersion:    "2.5.1",
		Variant:       "pytorch-cuda",
		Stages:        []string{"all"},
		VersionSuffix: "dev",
		Builder:       "kaniko",
		ContextDir:    m.CICD.ArgoWorkflows.BuildContext.Default,
		KanikoImage:   "kaniko:test",
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	contextEnv, err := BuildContextEnvVars(
		m.CICD.ArgoWorkflows.BuildContext.Env,
		m.CICD.ArgoWorkflows.BuildContext.SecretName,
		m.CICD.ArgoWorkflows.BuildContext.SecretEnv,
	)
	if err != nil {
		t.Fatalf("build context env: %v", err)
	}

	got, err := BuildWorkflowYAML(p, Options{
		Kind:                    "workflowtemplate",
		Namespace:               "default",
		ServiceAccountName:      "argo-workflow",
		ContextEnv:              contextEnv,
		InsecureRegistries:      m.CICD.ArgoWorkflows.InsecureRegistries,
		SkipPushPermissionCheck: m.CICD.ArgoWorkflows.SkipPushPermissionCheck,
		RegistrySecretName:      "regcred",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	want, err := os.ReadFile(gPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s\n", string(got), string(want))
	}
}

func TestBuildWorkflowYAML_RendersNoopStageWithoutKaniko(t *testing.T) {
	p := &pipeline.Plan{
		Name: "demo",
		Tasks: []pipeline.Task{
			{
				Name:  "host-driver-demo",
				Stage: "host_driver",
				Kaniko: pipeline.KanikoSpec{
					Image: "alpine:3.20",
				},
			},
		},
	}

	got, err := BuildWorkflowYAML(p, Options{
		Kind: "workflowtemplate",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)
	if !strings.Contains(out, "image: alpine:3.20") {
		t.Fatalf("noop template should use lightweight container, got:\n%s", out)
	}
	if strings.Contains(out, "/kaniko/executor") {
		t.Fatalf("noop template should not invoke kaniko, got:\n%s", out)
	}
	if strings.Contains(out, "--dockerfile=") {
		t.Fatalf("noop template should not render dockerfile args, got:\n%s", out)
	}
}

func TestBuildWorkflowYAML_RendersHostDriverGpuCheck(t *testing.T) {
	p := &pipeline.Plan{
		Name: "demo",
		Tasks: []pipeline.Task{
			{
				Name:  "host-driver-nvidia",
				Stage: "host_driver",
				HostDriver: pipeline.HostDriverSpec{
					Image: "nvidia/cuda:test",
					Commands: []string{
						"nvidia-smi -L",
					},
					ResourceLimits: map[string]string{
						"nvidia.com/gpu": "1",
					},
					ResourceRequests: map[string]string{
						"nvidia.com/gpu": "1",
					},
				},
			},
		},
	}

	got, err := BuildWorkflowYAML(p, Options{
		Kind: "workflowtemplate",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)
	for _, want := range []string{
		"image: nvidia/cuda:test",
		"nvidia-smi -L",
		"nvidia.com/gpu: \"1\"",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered workflow missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "/kaniko/executor") {
		t.Fatalf("host_driver should not invoke kaniko, got:\n%s", out)
	}
}

func TestBuildWorkflowYAML_RendersInsecureRegistries(t *testing.T) {
	p := &pipeline.Plan{
		Name: "demo",
		Tasks: []pipeline.Task{
			{
				Name:  "base",
				Stage: "base_image",
				Kaniko: pipeline.KanikoSpec{
					Image:       "kaniko:test",
					ContextDir:  ".",
					Dockerfile:  "Dockerfile",
					Destination: "registry.local:5000/k8ace/base:dev",
				},
			},
		},
	}

	got, err := BuildWorkflowYAML(p, Options{
		Kind:               "workflowtemplate",
		InsecureRegistries: []string{"registry.local:5000", "", "registry.local:5000"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)
	if count := strings.Count(out, "--insecure-registry=registry.local:5000"); count != 1 {
		t.Fatalf("insecure registry arg count = %d, want 1\n%s", count, out)
	}
}

func TestBuildWorkflowYAML_RendersKanikoSnapshotOptions(t *testing.T) {
	p := &pipeline.Plan{
		Name: "demo",
		Tasks: []pipeline.Task{
			{
				Name:  "app",
				Stage: "app_image",
				Kaniko: pipeline.KanikoSpec{
					Image:       "kaniko:test",
					ContextDir:  "s3://bucket/context.tar.gz",
					Dockerfile:  "dockerfiles/app/Dockerfile",
					Destination: "registry.local:5000/k8ace/app:dev",
				},
			},
		},
	}

	got, err := BuildWorkflowYAML(p, Options{
		Kind: "workflowtemplate",
		Kaniko: matrix.ArgoKanikoConfig{
			SnapshotMode:   "redo",
			SingleSnapshot: true,
			UseNewRun:      true,
			Cleanup:        true,
			Reproducible:   true,
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)
	for _, want := range []string{
		"--snapshot-mode=redo",
		"--single-snapshot",
		"--use-new-run",
		"--cleanup",
		"--reproducible",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered workflow missing %q:\n%s", want, out)
		}
	}
}
