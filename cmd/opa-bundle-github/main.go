package main

import (
	"log/slog"
	"strings"

	"github.com/spf13/viper"

	"github.com/ibice/opa-bundle-github/pkg/cmd"
)

func main() {
	cmd := cmd.New("opa-bundle-github")

	viper.BindPFlags(cmd.Flags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	err := cmd.Execute()
	if err != nil {
		slog.Error("Execute command", "error", err)
	}
}
