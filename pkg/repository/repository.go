package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-github/v56/github"
	"github.com/open-policy-agent/opa/compile"

	"github.com/ibice/opa-bundle-github/pkg/log"
)

var _ Interface = &Repository{}

type Interface interface {
	Get(context.Context, string) (io.Reader, string, error)
}

type Repository struct {
	repo    string
	owner   string
	dir     string
	branch  string
	baseURL *url.URL
	token   string
	client  *github.Client
	logger  *slog.Logger
}

type Option func(*Repository)

func New(repo, owner, dir, branch string, opts ...Option) (*Repository, error) {
	if repo == "" {
		return nil, fmt.Errorf("repo is required")
	}
	if owner == "" {
		return nil, fmt.Errorf("org is required")
	}
	if dir == "" {
		dir = "."
	}
	if branch == "" {
		branch = "HEAD"
	}

	logger := log.Logger.With("repo", repo, "owner", owner, "dir", dir, "branch", branch)
	logger.Debug("Creating repository service")

	repository := &Repository{
		repo:    repo,
		owner:   owner,
		dir:     dir,
		branch:  branch,
		client:  github.NewClient(&http.Client{Transport: newLoggerRoundTripper("githubClient")}),
		logger:  logger,
		baseURL: &url.URL{Scheme: "https", Host: "api.github.com"},
	}

	for _, opt := range opts {
		opt(repository)
	}
	return repository, nil
}

func WithToken(token string) Option {
	return func(r *Repository) {
		r.token = token
		r.client = r.client.WithAuthToken(token)
		slog.Debug("Set token for GitHub client")
	}
}

func WithGitHubURL(base *url.URL) Option {
	return func(r *Repository) {
		c, err := r.client.WithEnterpriseURLs(base.String(), "")
		if err != nil {
			slog.Error("Creating GitHub Enterprise client", "baseURL", base.String(), "error", err)
			return
		}
		r.client = c
		r.baseURL = base
		slog.Debug("Using GitHub Enterprise client", "baseURL", base.String())
	}
}

func (repository Repository) Get(ctx context.Context, lastRevision string) (data io.Reader, revision string, err error) {
	repository.logger.Debug("Get branch information")
	branch, res, err := repository.client.Repositories.GetBranch(ctx,
		repository.owner,
		repository.repo,
		repository.branch,
		5,
	)
	if err != nil {
		return nil, "", fmt.Errorf("get branch: %v", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", fmt.Errorf("unexpected status code %d getting branch", res.StatusCode)
	}

	revision = branch.Commit.GetSHA()
	repository.logger.Debug("Comparing revisions", "last", lastRevision, "current", revision)
	if lastRevision == revision {
		return
	}

	repository.logger.Debug("Downloading repo content", "revision", revision)
	clone, err := repository.clone(ctx, revision)
	if err != nil {
		return nil, "", fmt.Errorf("clone: %v", err)
	}

	// Update revision in case a commit was pushed while we cloned
	h, err := clone.Head()
	if err == nil {
		revision = h.Hash().String()
	}

	repository.logger.Debug("Building bundle")
	data, err = repository.bundle(ctx, clone)
	return
}

func (repository Repository) clone(ctx context.Context, revision string) (*git.Repository, error) {
	u := repository.baseURL
	u.Path = path.Join(repository.owner, repository.repo)
	u.Host = strings.TrimPrefix(u.Host, "api.")

	if repository.token != "" {
		u.User = url.UserPassword("git", repository.token)
	}

	return git.CloneContext(ctx, memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           u.String(),
		Depth:         1,
		ReferenceName: plumbing.NewBranchReferenceName(repository.branch),
		SingleBranch:  true,
	})
}

func (repository Repository) bundle(ctx context.Context, r *git.Repository) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)

	tree, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("worktree: %v", err)
	}

	compiler := compile.New().
		WithOutput(buf).
		WithPaths(repository.dir).
		WithFS(&fsAdapter{tree.Filesystem})

	err = compiler.Build(ctx)

	return buf, err
}
