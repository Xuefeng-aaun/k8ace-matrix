package render

import (
	"os"
	"path/filepath"
	"runtime"
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
		ContextDir:    ".",
		KanikoImage:   "kaniko:test",
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	got, err := BuildWorkflowYAML(p, Options{
		Kind:               "workflowtemplate",
		RegistrySecretName: "regcred",
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
