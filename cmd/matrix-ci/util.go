package main

import (
	"strings"

	"k8ace-matrix/internal/argo/render"
	"k8ace-matrix/internal/matrix"
)

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func resolveContextDir(flagValue string, flagChanged bool, aw matrix.ArgoWorkflows) string {
	if flagChanged {
		return flagValue
	}
	return firstNonEmpty(aw.BuildContext.Default, flagValue, ".")
}

func buildContextEnv(aw matrix.ArgoWorkflows) ([]render.EnvVar, error) {
	return render.BuildContextEnvVars(
		aw.BuildContext.Env,
		aw.BuildContext.SecretName,
		aw.BuildContext.SecretEnv,
	)
}
