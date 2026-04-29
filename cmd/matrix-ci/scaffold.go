package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8ace-matrix/internal/matrix"
	"k8ace-matrix/internal/pipeline"
	"k8ace-matrix/internal/scaffold"
)

func newScaffoldCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate project files from images-matrix.yaml",
	}
	cmd.AddCommand(newScaffoldDockerfilesCmd())
	return cmd
}

func newScaffoldDockerfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dockerfiles",
		Short: "Generate dockerfiles/ directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			mPath := viper.GetString("matrix")
			if mPath == "" {
				return fmt.Errorf("--matrix is required")
			}
			m, err := matrix.Load(mPath)
			if err != nil {
				return err
			}

			sel := pipeline.Selection{
				Hardwares:     viper.GetStringSlice("hardware"),
				Apps:          viper.GetStringSlice("app"),
				AppName:       viper.GetString("app-name"),
				AppVersion:    viper.GetString("app-version"),
				Variant:       viper.GetString("variant"),
				BaseTagSuffix: viper.GetString("base-tag-suffix"),
				Stages:        viper.GetStringSlice("stage"),
				PriorityTier:  viper.GetString("priority"),

				RegistryPrefix: viper.GetString("registry-prefix"),
				VersionSuffix:  viper.GetString("version-suffix"),
			}

			units, err := pipeline.DeriveUnits(m, sel)
			if err != nil {
				return err
			}

			stages := viper.GetStringSlice("stage")
			if len(stages) == 0 {
				stages = []string{"all"}
			}

			aptMirror := strings.TrimSpace(viper.GetString("apt-mirror"))
			if aptMirror == "" {
				aptMirror = "https://mirrors.tuna.tsinghua.edu.cn"
			}
			pipIndex := strings.TrimSpace(viper.GetString("pip-index-url"))
			if pipIndex == "" {
				pipIndex = "https://pypi.tuna.tsinghua.edu.cn/simple"
			}

			mirrors := scaffold.MirrorConfig{
				AptMirror:        aptMirror,
				PipIndexURL:      pipIndex,
				PipExtraIndexURL: strings.TrimSpace(viper.GetString("pip-extra-index-url")),
			}

			for _, u := range units {
				if err := scaffold.WriteDockerfiles(".", u, stages, mirrors); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().String("matrix", "", "path to images-matrix.yaml")
	cmd.Flags().StringSlice("hardware", nil, "hardware vendors (repeatable or comma-separated)")
	cmd.Flags().StringSlice("app", nil, "apps (repeatable or comma-separated)")
	cmd.Flags().String("app-name", "", "application name from application_matrix (e.g. pytorch)")
	cmd.Flags().String("app-version", "", "application version (e.g. 2.5.1)")
	cmd.Flags().String("variant", "", "variant name (e.g. pytorch-cuda)")
	cmd.Flags().String("base-tag-suffix", "", "base image tag_suffix override from base_image_matrix")
	cmd.Flags().StringSlice("stage", []string{"base_image", "app_image"}, "stages to scaffold (or 'all')")
	cmd.Flags().String("priority", "", "priority tier from priority_build_list")

	cmd.Flags().String("apt-mirror", "", "apt mirror base url (e.g. https://mirrors.tuna.tsinghua.edu.cn)")
	cmd.Flags().String("pip-index-url", "", "pip index url (e.g. https://pypi.tuna.tsinghua.edu.cn/simple)")
	cmd.Flags().String("pip-extra-index-url", "", "pip extra index url")

	cmd.Flags().String("registry-prefix", "", "registry prefix override")
	cmd.Flags().String("version-suffix", "dev", "image tag/version suffix")

	_ = viper.BindPFlag("matrix", cmd.Flags().Lookup("matrix"))
	_ = viper.BindPFlag("hardware", cmd.Flags().Lookup("hardware"))
	_ = viper.BindPFlag("app", cmd.Flags().Lookup("app"))
	_ = viper.BindPFlag("app-name", cmd.Flags().Lookup("app-name"))
	_ = viper.BindPFlag("app-version", cmd.Flags().Lookup("app-version"))
	_ = viper.BindPFlag("variant", cmd.Flags().Lookup("variant"))
	_ = viper.BindPFlag("base-tag-suffix", cmd.Flags().Lookup("base-tag-suffix"))
	_ = viper.BindPFlag("stage", cmd.Flags().Lookup("stage"))
	_ = viper.BindPFlag("priority", cmd.Flags().Lookup("priority"))

	_ = viper.BindPFlag("apt-mirror", cmd.Flags().Lookup("apt-mirror"))
	_ = viper.BindPFlag("pip-index-url", cmd.Flags().Lookup("pip-index-url"))
	_ = viper.BindPFlag("pip-extra-index-url", cmd.Flags().Lookup("pip-extra-index-url"))

	_ = viper.BindPFlag("registry-prefix", cmd.Flags().Lookup("registry-prefix"))
	_ = viper.BindPFlag("version-suffix", cmd.Flags().Lookup("version-suffix"))

	return cmd
}
