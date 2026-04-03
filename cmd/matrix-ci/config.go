package main

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetEnvPrefix("MATRIX_CI")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
	if _, err := os.Stat("images-matrix.yaml"); err == nil {
		viper.SetDefault("matrix", "images-matrix.yaml")
	}
}

