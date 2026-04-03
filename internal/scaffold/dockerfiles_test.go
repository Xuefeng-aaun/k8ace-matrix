package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"k8ace-matrix/internal/matrix"
	"k8ace-matrix/internal/pipeline"
)

func TestWriteDockerfiles(t *testing.T) {
	dir := t.TempDir()

	u := pipeline.BuildUnit{
		Hardware:    "nvidia",
		AppName:     "pytorch",
		AppVersion:  "2.5.1",
		VariantName: "pytorch-cuda",
		BaseRef:     "cuda_base",
		BaseVariant: matrix.BaseVariant{TagSuffix: "cuda12.4-devel-ubuntu22.04"},
		BuildArgs: map[string]string{
			"PYTORCH_VERSION": "2.5.1",
		},
		AdditionalPackages: []string{"torch==2.5.1", "torchvision"},
	}

	m := MirrorConfig{
		AptMirror:    "https://mirrors.tuna.tsinghua.edu.cn",
		PipIndexURL:  "https://pypi.tuna.tsinghua.edu.cn/simple",
	}

	if err := WriteDockerfiles(dir, u, []string{"base_image", "app_image", "test"}, m); err != nil {
		t.Fatalf("WriteDockerfiles: %v", err)
	}

	basePath := filepath.Join(dir, "dockerfiles", "base_image", "nvidia", "cuda_base", "cuda12.4-devel-ubuntu22.04", "Dockerfile")
	baseBytes, err := os.ReadFile(basePath)
	if err != nil {
		t.Fatalf("read base dockerfile: %v", err)
	}
	if string(baseBytes) == "" {
		t.Fatalf("base dockerfile empty")
	}

	appPath := filepath.Join(dir, "dockerfiles", "app_image", "nvidia", "pytorch", "2.5.1", "pytorch-cuda", "Dockerfile")
	appBytes, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatalf("read app dockerfile: %v", err)
	}
	if string(appBytes) == "" {
		t.Fatalf("app dockerfile empty")
	}

	noopPath := filepath.Join(dir, "dockerfiles", "test", "noop", "Dockerfile")
	if _, err := os.Stat(noopPath); err != nil {
		t.Fatalf("noop dockerfile not found: %v", err)
	}
}

