package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dogmatiq/browser/internal/githubutils"
	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
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
	c *github.Client,
	repos ...*github.Repository,
) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, r := range repos {
		g.Go(func() error {
			return w.foundRepo(ctx, c, r)
		})
	}

	return g.Wait()
}

func (*RepositoryWatcher) foundRepo(
	ctx context.Context,
	c *github.Client,
	r *github.Repository,
) error {
	if err := minibus.Send(
		ctx,
		messages.RepoFound{
			RepoSource: "github",
			RepoName:   r.GetFullName(),
		},
	); err != nil {
		return err
	}

	var version string

	if err := githubutils.Range(
		ctx,
		func(ctx context.Context, opts *github.ListOptions) ([]*github.RepositoryTag, *github.Response, error) {
			return c.Repositories.ListTags(ctx, r.GetOwner().GetLogin(), r.GetName(), opts)
		},
		func(ctx context.Context, t *github.RepositoryTag) error {
			v := semver.Canonical(t.GetName())
			if v == "" {
				return nil // invalid
			}

			if version == "" || semver.Compare(v, version) > 0 {
				version = v
			}

			return nil
		},
	); err != nil {
		return err
	}

	if version == "" {
		// TODO: analyze default branch
		return nil
	}

	u, _, err := c.Repositories.GetArchiveLink(
		ctx,
		r.GetOwner().GetLogin(),
		r.GetName(),
		github.Tarball,
		&github.RepositoryContentGetOptions{
			Ref: version,
		},
		1,
	)
	if err != nil {
		return err
	}

	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if err := githubutils.DownloadArchive(
		ctx,
		c.Client(),
		u,
		dir,
	); err != nil {
		return err
	}

	return filepath.WalkDir(
		dir,
		func(path string, info os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if filepath.Base(path) != "go.mod" {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			mod, err := modfile.Parse(
				path,
				content,
				nil, // version fixer
			)
			if err != nil {
				return err
			}

			return minibus.Send(
				ctx,
				messages.GoModuleFound{
					ModulePath:    mod.Module.Mod.Path,
					ModuleVersion: version,
				},
			)
		},
	)
}

func (w *RepositoryWatcher) lostRepos(
	ctx context.Context,
	_ *github.Client,
	repos ...*github.Repository,
) error {
	for _, r := range repos {
		if err := minibus.Send(
			ctx,
			messages.RepoLost{
				RepoSource: "github",
				RepoName:   r.GetFullName(),
			},
		); err != nil {
			return err
		}
	}

	return nil
}
