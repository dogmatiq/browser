package github

import (
	"context"
	"fmt"

	"github.com/dogmatiq/browser/internal/githubutils"
	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
)

// RepositoryWatcher publishes messages about the repositories that are
// accessible to the GitHub application.
type RepositoryWatcher struct {
	Clients *githubutils.ClientSet
}

// Run starts the watcher.
func (w *RepositoryWatcher) Run(ctx context.Context) error {
	minibus.Subscribe[*github.InstallationEvent](ctx)
	minibus.Subscribe[*github.InstallationRepositoriesEvent](ctx)
	minibus.Ready(ctx)

	if err := githubutils.Range(
		ctx,
		w.Clients.App().Apps.ListInstallations,
		w.addInstallation,
	); err != nil {
		return fmt.Errorf("github: unable to list installations: %w", err)
	}

	for m := range minibus.Inbox(ctx) {
		if err := w.handleMessage(ctx, m); err != nil {
			return err
		}
	}

	return ctx.Err()
}

func (w *RepositoryWatcher) handleMessage(
	ctx context.Context,
	m any,
) error {
	switch m := m.(type) {
	case *github.InstallationEvent:
		return w.handleInstallationEvent(ctx, m)
	case *github.InstallationRepositoriesEvent:
		return w.handleInstallationRepositoriesEvent(ctx, m)
	default:
		panic("unexpected message type")
	}
}

func (w *RepositoryWatcher) handleInstallationEvent(
	ctx context.Context,
	m *github.InstallationEvent,
) error {
	c := w.Clients.Installation(m.Installation.GetID())

	switch m.GetAction() {
	case "created":
		return w.foundRepos(ctx, c, m.Repositories...)
	case "deleted":
		return w.lostRepos(ctx, c, m.Repositories...)
	}

	return nil
}

func (w *RepositoryWatcher) handleInstallationRepositoriesEvent(
	ctx context.Context,
	m *github.InstallationRepositoriesEvent,
) error {
	c := w.Clients.Installation(m.Installation.GetID())

	if err := w.lostRepos(ctx, c, m.RepositoriesRemoved...); err != nil {
		return err
	}

	if err := w.foundRepos(ctx, c, m.RepositoriesAdded...); err != nil {
		return err
	}

	return nil
}

func (w *RepositoryWatcher) addInstallation(ctx context.Context, in *github.Installation) error {
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
			return w.foundRepos(ctx, c, r)
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

func (w *RepositoryWatcher) foundRepos(
	ctx context.Context,
	_ *github.Client,
	repos ...*github.Repository,
) error {
	for _, r := range repos {
		return minibus.Send(
			ctx,
			&messages.RepoFound{
				Source: "github",
				Name:   r.GetFullName(),
			})
	}

	return nil
}

func (w *RepositoryWatcher) lostRepos(
	ctx context.Context,
	_ *github.Client,
	repos ...*github.Repository,
) error {
	for _, r := range repos {
		return minibus.Send(
			ctx,
			&messages.RepoLost{
				Source: "github",
				Name:   r.GetFullName(),
			})
	}

	return nil
}
