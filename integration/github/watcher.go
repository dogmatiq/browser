package github

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dogmatiq/browser/internal/githubutils"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
)

type Watcher struct {
	Clients *githubutils.ClientSet
	Logger  *slog.Logger
}

func (w *Watcher) Run(ctx context.Context) error {
	minibus.Ready(ctx)

	if err := githubutils.Range(
		ctx,
		w.Clients.App().Apps.ListInstallations,
		w.addInstallation,
	); err != nil {
		return fmt.Errorf("github: unable to list installations: %w", err)
	}

	return nil
}

func (w *Watcher) addInstallation(ctx context.Context, in *github.Installation) error {
	c := w.Clients.Installation(in.GetID())

	if err := githubutils.Range(
		ctx,
		func(ctx context.Context, opts *github.ListOptions) ([]*github.Repository, *github.Response, error) {
			repos, res, err := c.Apps.ListRepos(ctx, opts)
			if err != nil {
				return nil, res, err
			}
			return repos.Repositories, res, nil
		},
		func(ctx context.Context, r *github.Repository) error {
			return w.addRepo(ctx, c, r)
		},
	); err != nil {
		return fmt.Errorf(
			"github: unable to list repositories for %q installation:  %w",
			in.GetAccount().GetName(),
			err,
		)
	}

	return nil
}

func (w *Watcher) addRepo(
	ctx context.Context,
	c *github.Client,
	r *github.Repository,
) error {
	r, _, err := c.Repositories.GetByID(ctx, r.GetID())
	if err != nil {
		return fmt.Errorf("github: unable to get %q repository: %w", r.GetFullName(), err)
	}

	w.Logger.InfoContext(
		ctx,
		"added repository",
		slog.String("repo_name", r.GetFullName()),
	)

	return nil
}
