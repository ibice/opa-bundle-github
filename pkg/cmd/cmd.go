package cmd

import (
	"log/slog"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ibice/opa-bundle-github/pkg/log"
	"github.com/ibice/opa-bundle-github/pkg/repository"
	"github.com/ibice/opa-bundle-github/pkg/server"
)

func New(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use: name,
		Run: run,
	}
	cmd.Flags().StringP("listen-address", "a", "", "address to listen")
	cmd.Flags().UintP("listen-port", "p", 8080, "port to listen")
	cmd.Flags().StringP("github-base-url", "u", "", "base URL for GitHub API")
	cmd.Flags().StringP("github-token", "t", "", "GitHub token")
	cmd.Flags().StringP("repo-name", "r", "", "repository name")
	cmd.Flags().StringP("repo-owner", "o", "", "repository organization or owner")
	cmd.Flags().StringP("repo-dir", "d", ".", "path to directory containing bundle files inside repository")
	cmd.Flags().StringP("repo-branch", "b", "HEAD", "path to directory containing bundle files inside repository")
	cmd.Flags().BoolP("verbose", "v", false, "show verbose logs")
	return cmd
}

func run(cmd *cobra.Command, args []string) {
	initLogging()

	var opts []repository.Option

	if token := viper.GetString("github-token"); token != "" {
		opts = append(opts, repository.WithToken(token))
	}

	if baseURL := viper.GetString("github-base-url"); baseURL != "" {
		u, err := url.Parse(baseURL)
		if err != nil {
			slog.Error("Parse GitHub base URL", "url", baseURL, "error", err)
		}
		opts = append(opts, repository.WithGitHubURL(u))
	}

	repository, err := repository.New(
		viper.GetString("repo-name"),
		viper.GetString("repo-owner"),
		viper.GetString("repo-dir"),
		viper.GetString("repo-branch"),
		opts...,
	)
	if err != nil {
		slog.Error("Creating repository service", "error", err)
		os.Exit(1)
	}

	server := server.New(
		viper.GetString("listen-address"),
		viper.GetUint("listen-port"),
		repository,
	)

	slog.Error(server.Run().Error())
}

func initLogging() {
	level := slog.LevelInfo
	if viper.GetBool("verbose") {
		level = slog.LevelDebug
	}
	var (
		handlerOpts = slog.HandlerOptions{Level: level}
		handler     = slog.NewTextHandler(os.Stderr, &handlerOpts)
		logger      = slog.New(handler)
	)
	slog.SetDefault(logger)
	log.Logger = logger
}
