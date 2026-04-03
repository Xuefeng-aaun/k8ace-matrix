package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8ace-matrix/internal/version"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "matrix-ci",
		Short: "Generate and submit Argo Workflows from images-matrix.yaml",
	}
)

func Execute() {
	rootCmd.Version = fmt.Sprintf("%s (%s) %s", version.Version, version.Commit, version.BuildDate)
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.AddCommand(newRenderCmd())
	rootCmd.AddCommand(newSubmitCmd())
	rootCmd.AddCommand(newScaffoldCmd())
}
