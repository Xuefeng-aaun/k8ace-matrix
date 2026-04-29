package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8ace-matrix/internal/argo/render"
	"k8ace-matrix/internal/argo/submit"
	"k8ace-matrix/internal/matrix"
	"k8ace-matrix/internal/pipeline"

	"github.com/spf13/cobra"
)

func newSubmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Render and submit an Argo Workflow",
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

			nsFlag, _ := cmd.Flags().GetString("namespace")
			ns := firstNonEmpty(nsFlag, aw.Namespace)
			if ns == "" {
				return fmt.Errorf("--namespace is required for submit")
			}

			hardwares, _ := cmd.Flags().GetStringSlice("hardware")
			apps, _ := cmd.Flags().GetStringSlice("app")
			appName, _ := cmd.Flags().GetString("app-name")
			appVersion, _ := cmd.Flags().GetString("app-version")
			variant, _ := cmd.Flags().GetString("variant")
			baseTagSuffix, _ := cmd.Flags().GetString("base-tag-suffix")
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

			name, _ := cmd.Flags().GetString("name")
			serviceAccount, _ := cmd.Flags().GetString("service-account")
			registrySecretName, _ := cmd.Flags().GetString("registry-secret-name")
			labels, _ := cmd.Flags().GetStringSlice("label")
			submitMode, _ := cmd.Flags().GetString("submit-mode")
			argoServerFlag, _ := cmd.Flags().GetString("argo-server")
			argoTokenFlag, _ := cmd.Flags().GetString("argo-token")

			sel := pipeline.Selection{
				Hardwares:      hardwares,
				Apps:           apps,
				AppName:        appName,
				AppVersion:     appVersion,
				Variant:        variant,
				BaseTagSuffix:  baseTagSuffix,
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

			plans, err := pipeline.BuildPlans(m, sel)
			if err != nil {
				return err
			}
			if len(plans) != 1 {
				return fmt.Errorf("submit requires exactly 1 build unit, got %d (use render --out-dir to generate multiple)", len(plans))
			}

			p := plans[0]
			yamlBytes, err := render.BuildWorkflowYAML(p, render.Options{
				Kind:                    "workflow",
				Name:                    name,
				Namespace:               ns,
				ServiceAccountName:      firstNonEmpty(serviceAccount, aw.ServiceAccount),
				ContextEnv:              contextEnv,
				InsecureRegistries:      aw.InsecureRegistries,
				SkipPushPermissionCheck: aw.SkipPushPermissionCheck,
				RegistrySecretName:      firstNonEmpty(registrySecretName, aw.RegistrySecret),
				Labels:                  labelsFromCSV(labels),
			})
			if err != nil {
				return err
			}

			mode := firstNonEmpty(submitMode, aw.SubmitModeDefault, "argo-server")
			if mode != "argo-server" {
				return fmt.Errorf("unsupported submit-mode: %s", mode)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			c := submit.ArgoServerClient{
				BaseURL: firstNonEmpty(argoServerFlag, aw.ArgoServer),
				Token:   firstNonEmpty(argoTokenFlag, os.Getenv("MATRIX_CI_ARGO_TOKEN")),
			}

			resp, err := c.SubmitWorkflowYAML(ctx, ns, yamlBytes)
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(resp)
			return err
		},
	}

	cmd.Flags().String("matrix", "", "path to images-matrix.yaml")
	cmd.Flags().StringSlice("hardware", nil, "hardware vendors (repeatable or comma-separated)")
	cmd.Flags().StringSlice("app", nil, "apps (repeatable or comma-separated)")
	cmd.Flags().String("app-name", "", "application name from application_matrix (e.g. pytorch)")
	cmd.Flags().String("app-version", "", "application version (e.g. 2.5.1)")
	cmd.Flags().String("variant", "", "variant name (e.g. pytorch-cuda)")
	cmd.Flags().String("base-tag-suffix", "", "base image tag_suffix override from base_image_matrix")
	cmd.Flags().StringSlice("stage", []string{"all"}, "stages (repeatable)")
	cmd.Flags().String("priority", "", "priority tier from priority_build_list")

	cmd.Flags().String("name", "", "workflow name override")
	cmd.Flags().String("namespace", "", "kubernetes namespace")
	cmd.Flags().String("service-account", "", "argo service account name")
	cmd.Flags().String("registry-secret-name", "", "k8s secret name for docker registry credentials (dockerconfigjson)")
	cmd.Flags().StringSlice("label", nil, "extra labels k=v (repeatable)")

	cmd.Flags().String("submit-mode", "", "argo-server")
	cmd.Flags().String("argo-server", "", "argo server base url")
	cmd.Flags().String("argo-token", "", "argo server bearer token (or MATRIX_CI_ARGO_TOKEN)")

	cmd.Flags().String("builder", "kaniko", "builder engine")
	cmd.Flags().String("context", "", "build context override (defaults to ci_cd.argo_workflows.build_context.default or '.')")
	cmd.Flags().String("dockerfile", "", "dockerfile path override")
	cmd.Flags().String("kaniko-image", "", "kaniko executor image")
	cmd.Flags().String("registry-prefix", "", "registry prefix override")
	cmd.Flags().String("version-suffix", "dev", "image tag/version suffix")

	return cmd
}
