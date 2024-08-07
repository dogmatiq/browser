package github

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dogmatiq/browser/internal/githubutils"
	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v63/github"
	"golang.org/x/mod/modfile"
)

// RepositoryWatcher publishes messages about the repositories that are
// accessible to the GitHub application.
type RepositoryWatcher struct {
	Clients *githubutils.ClientSet
	Logger  *slog.Logger
}

// Run starts the watcher.
func (w *RepositoryWatcher) Run(ctx context.Context) (err error) {
	minibus.Subscribe[*github.InstallationEvent](ctx)
	minibus.Subscribe[*github.InstallationRepositoriesEvent](ctx)
	minibus.Subscribe[*github.RepositoryEvent](ctx)
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
		return w.foundRepos(ctx, m.GetInstallation(), m.Repositories...)
	case "deleted":
		return w.lostRepos(ctx, m.Repositories...)
	}

	return nil
}

func (w *RepositoryWatcher) handleInstallationRepositoriesEvent(
	ctx context.Context,
	m *github.InstallationRepositoriesEvent,
) error {
	if err := w.foundRepos(ctx, m.GetInstallation(), m.RepositoriesAdded...); err != nil {
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
		return w.foundRepos(ctx, m.GetInstallation(), m.Repo)
	case "archived":
		return w.lostRepos(ctx, m.Repo)
	default:
		return nil
	}
}

func (w *RepositoryWatcher) addInstallation(ctx context.Context, in *github.Installation) (err error) {
	c := w.Clients.Installation(in.GetID())

	if err := githubutils.RangePages(
		ctx,
		func(ctx context.Context, opts *github.ListOptions) ([]*github.Repository, *github.Response, error) {
			repos, res, err := c.Apps.ListRepos(ctx, opts)
			if err != nil {
				return nil, res, err
			}
			return repos.Repositories, res, nil
		},
		func(ctx context.Context, repos []*github.Repository) error {
			return w.foundRepos(ctx, in, repos...)
		},
	); err != nil {
		return fmt.Errorf(
			"github: unable to list repositories for %q installation: %w",
			in.GetAccount().GetLogin(),
			err,
		)
	}

	return nil
}

func (w *RepositoryWatcher) foundRepos(
	ctx context.Context,
	in *github.Installation,
	repos ...*github.Repository,
) error {
	var repoIDs []int64
	for _, repo := range repos {
		repoIDs = append(repoIDs, repo.GetID())
	}

	token, _, err := w.
		Clients.
		App().
		Apps.
		CreateInstallationToken(
			ctx,
			in.GetID(),
			&github.InstallationTokenOptions{
				RepositoryIDs: repoIDs,
				Permissions: &github.InstallationPermissions{
					Contents: github.String("read"),
				},
			},
		)
	if err != nil {
		return fmt.Errorf("github: unable to create installation token for git clone: %w", err)
	}

	auth := &http.BasicAuth{
		Username: "x-access-token",
		Password: token.GetToken(),
	}

	w.Logger.DebugContext(
		ctx,
		"generated installation token for read-only git operations",
		slog.String("installation", in.GetAccount().GetLogin()),
		slog.Duration("token_ttl", time.Until(token.GetExpiresAt().Time)),
	)

	// g, ctx := errgroup.WithContext(ctx)

	for _, repo := range repos {
		if isIgnored(repo) {
			continue
		}

		// g.Go(func() error {
		if err := minibus.Send(
			ctx,
			messages.RepoFound{
				ID:     generateRepoID(repo),
				Source: "github",
				Name:   repo.GetFullName(),
			},
		); err != nil {
			return err
		}

		if err := w.sync(ctx, repo, auth); err != nil {
			return err
		}
		// })
	}

	return nil
	// return g.Wait()
}

func (w *RepositoryWatcher) lostRepos(
	ctx context.Context,
	repos ...*github.Repository,
) error {
	for _, repo := range repos {
		if err := minibus.Send(
			ctx,
			messages.RepoLost{
				ID: generateRepoID(repo),
			},
		); err != nil {
			return err
		}
	}

	return nil
}

func (w *RepositoryWatcher) sync(
	ctx context.Context,
	repo *github.Repository,
	auth transport.AuthMethod,
) error {
	dir := filepath.Join(
		os.TempDir(),
		"dogma-browser",
		"github",
		strconv.FormatInt(repo.GetID(), 10),
	)

	clone, err := w.fetch(ctx, repo, auth, dir)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		clone, err = w.clone(ctx, repo, auth, dir)
	}
	if err != nil {
		return err
	}

	head, err := clone.Head()
	if err != nil {
		return fmt.Errorf("analyzer: unable to get HEAD of git repository: %w", err)
	}

	return filepath.WalkDir(
		dir,
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() || filepath.Base(path) != "go.mod" {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			mod, err := modfile.Parse(path, content, nil /* version fixer*/)
			if err != nil {
				return err
			}

			return minibus.Send(
				ctx,
				messages.GoModuleFound{
					RepoID:        generateRepoID(repo),
					ModulePath:    mod.Module.Mod.Path,
					ModuleVersion: head.Hash().String(),
				},
			)
		},
	)
}

func (w *RepositoryWatcher) clone(
	ctx context.Context,
	repo *github.Repository,
	auth transport.AuthMethod,
	dir string,
) (*git.Repository, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("analyzer: unable to create directory for git clone: %w", err)
	}

	clone, err := git.PlainCloneContext(
		ctx,
		dir,
		false,
		&git.CloneOptions{
			URL:   repo.GetCloneURL(),
			Auth:  auth,
			Depth: 1,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("analyzer: unable to clone git repository: %w", err)
	}

	return clone, nil
}

func (w *RepositoryWatcher) fetch(
	ctx context.Context,
	_ *github.Repository,
	auth transport.AuthMethod,
	dir string,
) (*git.Repository, error) {
	clone, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("analyzer: unable to open git repository at for fetch: %w", err)
	}

	if err := clone.FetchContext(
		ctx,
		&git.FetchOptions{
			Auth:  auth,
			Depth: 1,
			Prune: true,
		},
	); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("analyzer: unable to fetch git repository: %w", err)
	}

	return clone, nil
}
