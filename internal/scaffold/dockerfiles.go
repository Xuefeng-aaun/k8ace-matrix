package scaffold

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"k8ace-matrix/internal/pipeline"
)

type MirrorConfig struct {
	AptMirror        string
	PipIndexURL      string
	PipExtraIndexURL string
}

func WriteDockerfiles(root string, unit pipeline.BuildUnit, stages []string, mirrors MirrorConfig) error {
	stageSet := map[string]bool{}
	for _, s := range stages {
		stageSet[strings.TrimSpace(s)] = true
	}

	if stageSet["all"] || stageSet["base_image"] {
		p := filepath.Join(root, "dockerfiles", "base_image", unit.Hardware, unit.BaseRef, unit.BaseVariant.TagSuffix, "Dockerfile")
		if err := writeFile(p, baseDockerfile(mirrors)); err != nil {
			return err
		}
	}

	if stageSet["all"] || stageSet["app_image"] {
		p := filepath.Join(root, "dockerfiles", "app_image", unit.Hardware, unit.AppName, unit.AppVersion, pipelineName(unit.VariantName), "Dockerfile")
		if err := writeFile(p, appDockerfile(unit, mirrors)); err != nil {
			return err
		}
	}

	for _, st := range stages {
		st = strings.TrimSpace(st)
		if st == "" || st == "all" || st == "base_image" || st == "app_image" {
			continue
		}
		p := filepath.Join(root, "dockerfiles", st, "noop", "Dockerfile")
		if err := writeFile(p, noopDockerfile()); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func baseDockerfile(m MirrorConfig) string {
	apt := strings.TrimSpace(m.AptMirror)
	pip := strings.TrimSpace(m.PipIndexURL)
	extra := strings.TrimSpace(m.PipExtraIndexURL)

	var b strings.Builder
	b.WriteString("ARG BASE_IMAGE\n")
	b.WriteString("FROM ${BASE_IMAGE}\n\n")

	if apt != "" {
		b.WriteString("RUN if [ -f /etc/apt/sources.list ]; then \\\n")
		b.WriteString("  sed -i 's|http://archive.ubuntu.com/ubuntu/|" + apt + "/ubuntu/|g; s|http://security.ubuntu.com/ubuntu/|" + apt + "/ubuntu/|g; s|https://archive.ubuntu.com/ubuntu/|" + apt + "/ubuntu/|g; s|https://security.ubuntu.com/ubuntu/|" + apt + "/ubuntu/|g' /etc/apt/sources.list; \\\n")
		b.WriteString("fi\n\n")
	}

	b.WriteString("RUN if command -v apt-get >/dev/null 2>&1; then \\\n")
	b.WriteString("  apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git python3 python3-pip && rm -rf /var/lib/apt/lists/*; \\\n")
	b.WriteString("fi\n\n")

	if pip != "" {
		b.WriteString("RUN mkdir -p /etc && \\\n")
		b.WriteString("  printf '[global]\\nindex-url = " + pip + "\\n' > /etc/pip.conf\n\n")
		if extra != "" {
			b.WriteString("RUN printf 'extra-index-url = " + extra + "\\n' >> /etc/pip.conf\n\n")
		}
	}

	return b.String()
}

func appDockerfile(u pipeline.BuildUnit, m MirrorConfig) string {
	pip := strings.TrimSpace(m.PipIndexURL)
	extra := strings.TrimSpace(m.PipExtraIndexURL)

	var b strings.Builder
	b.WriteString("ARG BASE_IMAGE\n")
	b.WriteString("FROM ${BASE_IMAGE}\n\n")

	keys := make([]string, 0, len(u.BuildArgs))
	for k := range u.BuildArgs {
		if k == "BASE_IMAGE" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("ARG " + k + "\n")
	}
	if len(keys) > 0 {
		b.WriteString("\n")
	}

	var pkgs []string
	var pipArgs []string
	for _, p := range u.AdditionalPackages {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "-") {
			pipArgs = append(pipArgs, p)
			continue
		}
		pkgs = append(pkgs, p)
	}

	if pip != "" {
		pipArgs = append(pipArgs, "--index-url", pip)
	}
	if extra != "" {
		pipArgs = append(pipArgs, "--extra-index-url", extra)
	}

	if len(pkgs) > 0 || len(pipArgs) > 0 {
		args := strings.Join(append(pipArgs, pkgs...), " ")
		b.WriteString("RUN python3 -m pip install -U pip && \\\n")
		b.WriteString("  python3 -m pip install " + args + "\n")
	}

	return b.String()
}

func noopDockerfile() string {
	return "ARG BASE_IMAGE\nFROM ${BASE_IMAGE}\n\nRUN echo noop\n"
}

func pipelineName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if s == "" {
		return "default"
	}
	return s
}
