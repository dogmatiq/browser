package github

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/dogmatiq/browser/internal/githubapi"
	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
	"golang.org/x/mod/modfile"
	"golang.org/x/sync/errgroup"
)

// RepositoryWatcher publishes messages about the repositories that are
// accessible to the GitHub application.
type RepositoryWatcher struct {
	Client *githubapi.AppClient
	Logger *slog.Logger
}

// Run starts the watcher.
func (w *RepositoryWatcher) Run(ctx context.Context) (err error) {
	minibus.Subscribe[*github.InstallationEvent](ctx)
	minibus.Subscribe[*github.InstallationRepositoriesEvent](ctx)
	minibus.Subscribe[*github.RepositoryEvent](ctx)
	minibus.Ready(ctx)

	if err := githubapi.Range(
		ctx,
		w.Client.REST().Apps.ListInstallations,
		w.addInstallation,
	); err != nil {
		return err
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
	case *github.RepositoryEvent:
		return w.handleRepositoryEvent(ctx, m)
	default:
		panic("unexpected message type")
	}
}

func (w *RepositoryWatcher) handleInstallationEvent(
	ctx context.Context,
	m *github.InstallationEvent,
) error {
	switch m.GetAction() {
	case "created":
		c := w.Client.InstallationClient(m.GetInstallation().GetID())
		return w.foundRepos(ctx, c, m.Repositories...)
	case "deleted":
		return w.lostRepos(ctx, m.Repositories...)
	}

	return nil
}

func (w *RepositoryWatcher) handleInstallationRepositoriesEvent(
	ctx context.Context,
	m *github.InstallationRepositoriesEvent,
) error {
	c := w.Client.InstallationClient(m.GetInstallation().GetID())

	if err := w.foundRepos(ctx, c, m.RepositoriesAdded...); err != nil {
		return err
	}

	if err := w.lostRepos(ctx, m.RepositoriesRemoved...); err != nil {
		return err
	}

	return nil
}

func (w *RepositoryWatcher) handleRepositoryEvent(
	ctx context.Context,
	m *github.RepositoryEvent,
) error {
	switch m.GetAction() {
	case "unarchived":
		c := w.Client.InstallationClient(m.GetInstallation().GetID())
		return w.foundRepos(ctx, c, m.Repo)
	case "archived":
		return w.lostRepos(ctx, m.Repo)
	default:
		return nil
	}
}

func (w *RepositoryWatcher) addInstallation(ctx context.Context, in *github.Installation) error {
	c := w.Client.InstallationClient(in.GetID())

	if err := githubapi.RangePages(
		ctx,
		func(ctx context.Context, opts *github.ListOptions) ([]*github.Repository, *github.Response, error) {
			repos, res, err := c.REST().Apps.ListRepos(ctx, opts)
			if err != nil {
				return nil, res, err
			}
			return repos.Repositories, res, nil
		},
		func(ctx context.Context, repos []*github.Repository) error {
			return w.foundRepos(ctx, c, repos...)
		},
	); err != nil {
		return fmt.Errorf("unable to add %q installation: %w", in.GetAccount().GetLogin(), err)
	}

	return nil
}

func (w *RepositoryWatcher) foundRepos(
	ctx context.Context,
	c *githubapi.InstallationClient,
	repos ...*github.Repository,
) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, repo := range repos {
		if !isIgnored(repo) {
			g.Go(func() error {
				return w.foundRepo(ctx, c, repo)
			})
		}
	}

	return g.Wait()
}

func (w *RepositoryWatcher) foundRepo(
	ctx context.Context,
	c *githubapi.InstallationClient,
	repo *github.Repository,
) error {
	if err := minibus.Send(
		ctx,
		messages.RepoFound{
			RepoSource: repoSource(w.Client),
			RepoID:     marshalRepoID(repo.GetID()),
			RepoName:   repo.GetFullName(),
		},
	); err != nil {
		return err
	}

	tree, _, err := c.REST().Git.GetTree(
		ctx,
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		"HEAD",
		true,
	)
	if err != nil {
		return err
	}

	if tree.GetTruncated() {
		w.Logger.WarnContext(
			ctx,
			"truncated git tree results, some modules may go undetected",
			slog.Group(
				"repo",
				slog.String("source", repoSource(w.Client)),
				slog.String("id", marshalRepoID(repo.GetID())),
				slog.String("name", repo.GetFullName()),
				slog.String("sha", tree.GetSHA()),
			),
		)
	}

	for _, entry := range tree.Entries {
		if entry.GetType() != "blob" {
			continue
		}

		if filepath.Base(entry.GetPath()) != "go.mod" {
			continue
		}

		data, _, err := c.REST().Git.GetBlobRaw(
			ctx,
			repo.GetOwner().GetLogin(),
			repo.GetName(),
			entry.GetSHA(),
		)
		if err != nil {
			return fmt.Errorf("unable to get go.mod content: %w", err)
		}

		mod, err := modfile.ParseLax(
			entry.GetPath(),
			data,
			nil, /* version fixer*/
		)
		if err != nil {
			return fmt.Errorf("unable to parse go.mod: %w", err)
		}

		if err := minibus.Send(
			ctx,
			messages.GoModuleFound{
				RepoSource:    repoSource(w.Client),
				RepoID:        marshalRepoID(repo.GetID()),
				ModulePath:    mod.Module.Mod.Path,
				ModuleVersion: tree.GetSHA(),
			},
		); err != nil {
			return err
		}
	}

	return nil
}

func (w *RepositoryWatcher) lostRepos(
	ctx context.Context,
	repos ...*github.Repository,
) error {
	for _, repo := range repos {
		if err := minibus.Send(
			ctx,
			messages.RepoLost{
				RepoSource: repoSource(w.Client),
				RepoID:     marshalRepoID(repo.GetID()),
			},
		); err != nil {
			return err
		}
	}

	return nil
}
