package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8ace-matrix/internal/argo/render"
	"k8ace-matrix/internal/matrix"
	"k8ace-matrix/internal/pipeline"

	"github.com/spf13/cobra"
)

func newRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render Argo Workflow YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			mPath, _ := cmd.Flags().GetString("matrix")
			if mPath == "" {
				if _, err := os.Stat("images-matrix.yaml"); err == nil {
					mPath = "images-matrix.yaml"
				} else {
					return fmt.Errorf("--matrix is required")
				}
			}

			m, err := matrix.Load(mPath)
			if err != nil {
				return err
			}

			aw := m.CICD.ArgoWorkflows

			hardwares, _ := cmd.Flags().GetStringSlice("hardware")
			apps, _ := cmd.Flags().GetStringSlice("app")
			appName, _ := cmd.Flags().GetString("app-name")
			appVersion, _ := cmd.Flags().GetString("app-version")
			variant, _ := cmd.Flags().GetString("variant")
			stages, _ := cmd.Flags().GetStringSlice("stage")
			priority, _ := cmd.Flags().GetString("priority")
			registryPrefix, _ := cmd.Flags().GetString("registry-prefix")
			versionSuffix, _ := cmd.Flags().GetString("version-suffix")
			builder, _ := cmd.Flags().GetString("builder")
			contextFlagChanged := cmd.Flags().Changed("context")
			contextDir, _ := cmd.Flags().GetString("context")
			contextDir = resolveContextDir(contextDir, contextFlagChanged, aw)
			dockerfile, _ := cmd.Flags().GetString("dockerfile")
			kanikoImage, _ := cmd.Flags().GetString("kaniko-image")

			kind, _ := cmd.Flags().GetString("kind")
			name, _ := cmd.Flags().GetString("name")
			namespace, _ := cmd.Flags().GetString("namespace")
			serviceAccount, _ := cmd.Flags().GetString("service-account")
			labels, _ := cmd.Flags().GetStringSlice("label")
			out, _ := cmd.Flags().GetString("output")
			outDir, _ := cmd.Flags().GetString("out-dir")
			split, _ := cmd.Flags().GetBool("split")
			registrySecretName, _ := cmd.Flags().GetString("registry-secret-name")

			sel := pipeline.Selection{
				Hardwares:      hardwares,
				Apps:           apps,
				AppName:        appName,
				AppVersion:     appVersion,
				Variant:        variant,
				Stages:         stages,
				PriorityTier:   priority,
				RegistryPrefix: registryPrefix,
				VersionSuffix:  versionSuffix,
				Builder:        builder,
				ContextDir:     contextDir,
				Dockerfile:     dockerfile,
				KanikoImage:    kanikoImage,
			}

			if sel.KanikoImage == "" {
				sel.KanikoImage = firstNonEmpty(aw.KanikoImage, "gcr.io/kaniko-project/executor:latest")
			}

			contextEnv, err := buildContextEnv(aw)
			if err != nil {
				return err
			}

			kind = firstNonEmpty(kind, aw.KindDefault, "workflowtemplate")
			registrySecretName = firstNonEmpty(registrySecretName, aw.RegistrySecret)

			if outDir != "" && split {
				plans, err := pipeline.BuildPlans(m, sel)
				if err != nil {
					return err
				}
				baseName := name
				for _, p := range plans {
					name := p.Name
					if baseName != "" {
						name = baseName + "-" + name
					}

					yamlBytes, err := render.BuildWorkflowYAML(p, render.Options{
						Kind:                    kind,
						Name:                    name,
						Namespace:               firstNonEmpty(namespace, aw.Namespace),
						ServiceAccountName:      firstNonEmpty(serviceAccount, aw.ServiceAccount),
						ContextEnv:              contextEnv,
						InsecureRegistries:      aw.InsecureRegistries,
						SkipPushPermissionCheck: aw.SkipPushPermissionCheck,
						RegistrySecretName:      registrySecretName,
						Labels:                  labelsFromCSV(labels),
					})
					if err != nil {
						return err
					}
					if err := writeOutDir(outDir, kind, name, yamlBytes); err != nil {
						return err
					}
				}
				return nil
			}

			p, err := pipeline.BuildPlan(m, sel)
			if err != nil {
				return err
			}
			yamlBytes, err := render.BuildWorkflowYAML(p, render.Options{
				Kind:                    kind,
				Name:                    name,
				Namespace:               firstNonEmpty(namespace, aw.Namespace),
				ServiceAccountName:      firstNonEmpty(serviceAccount, aw.ServiceAccount),
				ContextEnv:              contextEnv,
				InsecureRegistries:      aw.InsecureRegistries,
				SkipPushPermissionCheck: aw.SkipPushPermissionCheck,
				RegistrySecretName:      registrySecretName,
				Labels:                  labelsFromCSV(labels),
			})
			if err != nil {
				return err
			}

			if outDir != "" {
				return writeOutDir(outDir, kind, p.Name, yamlBytes)
			}
			if out == "" || out == "-" {
				_, err := os.Stdout.Write(yamlBytes)
				return err
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			return os.WriteFile(out, yamlBytes, 0o644)
		},
	}

	cmd.Flags().String("matrix", "", "path to images-matrix.yaml")
	cmd.Flags().StringSlice("hardware", nil, "hardware vendors (repeatable or comma-separated)")
	cmd.Flags().StringSlice("app", nil, "apps (repeatable or comma-separated)")
	cmd.Flags().String("app-name", "", "application name from application_matrix (e.g. pytorch)")
	cmd.Flags().String("app-version", "", "application version (e.g. 2.5.1)")
	cmd.Flags().String("variant", "", "variant name (e.g. pytorch-cuda)")
	cmd.Flags().StringSlice("stage", []string{"all"}, "stages (repeatable)")
	cmd.Flags().String("priority", "", "priority tier from priority_build_list")

	cmd.Flags().String("kind", "", "workflowtemplate or workflow")
	cmd.Flags().String("name", "", "object name override")
	cmd.Flags().String("namespace", "", "kubernetes namespace")
	cmd.Flags().String("service-account", "", "argo service account name")
	cmd.Flags().StringSlice("label", nil, "extra labels k=v (repeatable)")

	cmd.Flags().String("output", "-", "output file path or '-' for stdout")
	cmd.Flags().String("out-dir", "", "output directory (writes structured files)")
	cmd.Flags().Bool("split", true, "split output into multiple files when using --out-dir")

	cmd.Flags().String("builder", "kaniko", "builder engine")
	cmd.Flags().String("context", "", "build context override (defaults to ci_cd.argo_workflows.build_context.default or '.')")
	cmd.Flags().String("dockerfile", "", "dockerfile path override")
	cmd.Flags().String("kaniko-image", "", "kaniko executor image")
	cmd.Flags().String("registry-secret-name", "", "k8s secret name for docker registry credentials (dockerconfigjson)")
	cmd.Flags().String("registry-prefix", "", "registry prefix override")
	cmd.Flags().String("version-suffix", "dev", "image tag/version suffix")

	return cmd
}

func labelsFromCSV(kvs []string) map[string]string {
	out := map[string]string{}
	for _, kv := range kvs {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func writeOutDir(outDir, kind, name string, yamlBytes []byte) error {
	kind = strings.ToLower(strings.TrimSpace(kind))
	sub := "workflowtemplates"
	if kind == "workflow" {
		sub = "workflows"
	}

	dir := filepath.Join(outDir, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	p := filepath.Join(dir, name+".yaml")
	return os.WriteFile(p, yamlBytes, 0o644)
}
