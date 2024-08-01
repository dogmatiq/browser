package analyzer

import (
	"context"
	"database/sql"
	"fmt"
	"go/ast"
	"go/token"
	"net/http"
	"os"
	"strings"

	"github.com/dogmatiq/browser/githubx"
	"github.com/dogmatiq/browser/persistence"
	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/configkit/static"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/google/go-github/v38/github"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// Analyzer performs static analysis on repositories and stores the results in
// the database.
type Analyzer struct {
	DB        *sql.DB
	Connector *githubx.Connector
	Logger    logging.Logger
}

// Analyze analyzes the repo with the given ID.
func (a *Analyzer) Analyze(ctx context.Context, repoID int64) error {
	c, ok, err := a.Connector.RepositoryClient(ctx, repoID)
	if err != nil {
		return fmt.Errorf("unable to obtain github client for repository #%d: %w", repoID, err)
	}
	if !ok {
		logging.Log(
			a.Logger,
			"[#%d] skipping analysis of inaccessible repository",
			repoID,
		)

		return nil
	}

	// First look up the repository by ID to find its name.
	r, res, err := c.Repositories.GetByID(ctx, repoID)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			logging.Log(
				a.Logger,
				"[#%d] skipping analysis of non-existent repository",
				repoID,
			)

			return nil
		}

		return fmt.Errorf(
			"unable to fetch repository details for #%d: %w",
			repoID,
			err,
		)
	}

	// Then look it up again by name because the GetByID() operation doesn't
	// return the complete repository information, such as whether the
	// repository is archived or a template repository.
	r, res, err = c.Repositories.Get(ctx, r.GetOwner().GetLogin(), r.GetName())
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			logging.Log(
				a.Logger,
				"[#%d %s] skipping analysis of non-existent repository",
				repoID,
				r.GetFullName(),
			)

			return nil
		}

		return fmt.Errorf(
			"unable to fetch repository details for %s: %w",
			r.GetFullName(),
			err,
		)
	}

	if err := a.analyze(ctx, c, r); err != nil {
		return fmt.Errorf("unable to analyze %s: %w", r.GetFullName(), err)
	}

	return nil
}

func (a *Analyzer) analyze(
	ctx context.Context,
	c *github.Client,
	r *github.Repository,
) error {
	if r.GetIsTemplate() {
		logging.Log(
			a.Logger,
			"[#%d %s] skipping analysis of template repository",
			r.GetID(),
			r.GetFullName(),
		)

		return nil
	}

	if r.GetArchived() {
		logging.Log(
			a.Logger,
			"[#%d %s] skipping analysis of archived repository",
			r.GetID(),
			r.GetFullName(),
		)

		return nil
	}

	if r.GetFork() {
		logging.Log(
			a.Logger,
			"[#%d %s] skipping analysis of forked repository",
			r.GetID(),
			r.GetFullName(),
		)

		return nil
	}

	branch, _, err := c.Repositories.GetBranch(
		ctx,
		r.GetOwner().GetLogin(),
		r.GetName(),
		r.GetDefaultBranch(),
		false,
	)
	if err != nil {
		return err
	}

	commit := branch.GetCommit().GetSHA()

	needsSync, err := persistence.RepositoryNeedsSync(
		ctx,
		a.DB,
		r,
		commit,
	)
	if err != nil {
		return err
	}

	if !needsSync {
		logging.Log(
			a.Logger,
			"[#%d %s] skipping analysis of %s branch (%s), commit has already been analysed",
			r.GetID(),
			r.GetFullName(),
			r.GetDefaultBranch(),
			commit,
		)

		return nil
	}

	ok, err := a.isGoModule(
		ctx,
		c,
		r,
		commit,
	)
	if err != nil {
		return err
	}

	var (
		apps []configkit.Application
		defs []persistence.TypeDef
	)

	if ok {
		pkgs, dir, err := a.loadPackages(
			ctx,
			c,
			r,
			commit,
		)
		if err != nil {
			return err
		}

		apps, defs = a.analyzePackages(r, pkgs, dir)
	}

	return persistence.SyncRepository(
		ctx,
		a.DB,
		r,
		commit,
		apps,
		defs,
	)
}

// isGoModule returns true if the given repository has a valid go.mod file in
// its root directory.
func (a *Analyzer) isGoModule(
	ctx context.Context,
	c *github.Client,
	r *github.Repository,
	commit string,
) (bool, error) {
	content, _, res, err := c.Repositories.GetContents(
		ctx,
		r.GetOwner().GetLogin(),
		r.GetName(),
		"go.mod",
		&github.RepositoryContentGetOptions{
			Ref: commit,
		},
	)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			logging.Log(
				a.Logger,
				"[#%d %s] skipping analysis of %s branch (%s), go.mod file not present",
				r.GetID(),
				r.GetFullName(),
				r.GetDefaultBranch(),
				commit,
			)

			return false, nil
		}

		return false, err
	}

	data, err := content.GetContent()
	if err != nil {
		return false, err
	}

	mod, err := modfile.ParseLax(
		"go.mod",
		[]byte(data),
		nil,
	)
	if err != nil {
		logging.Log(
			a.Logger,
			"[#%d %s] skipping analysis of %s branch (%s), go.mod file is invalid: %s",
			r.GetID(),
			r.GetFullName(),
			r.GetDefaultBranch(),
			commit,
			err,
		)

		return false, nil
	}

	if mod.Module.Mod.Path == "github.com/dogmatiq/dogma" {
		logging.Log(
			a.Logger,
			"[#%d %s] skipping analysis of %s branch (%s), found dogma module: %s",
			r.GetID(),
			r.GetFullName(),
			r.GetDefaultBranch(),
			commit,
			mod.Module.Mod.Path,
		)

		return false, nil
	}

	logging.Log(
		a.Logger,
		"[#%d %s] analyzing %s branch (%s), found module %s",
		r.GetID(),
		r.GetFullName(),
		r.GetDefaultBranch(),
		commit,
		mod.Module.Mod.Path,
	)

	return true, nil
}

// loadPackages parses the Go source in the repository and returns the packages
// it contains.
func (a *Analyzer) loadPackages(
	ctx context.Context,
	c *github.Client,
	r *github.Repository,
	commit string,
) ([]*packages.Package, string, error) {
	inst, _, err := a.Connector.AppClient.Apps.FindRepositoryInstallationByID(ctx, r.GetID())
	if err != nil {
		return nil, "", err
	}

	token, _, err := a.Connector.AppClient.Apps.CreateInstallationToken(
		ctx,
		inst.GetID(),
		&github.InstallationTokenOptions{
			Permissions: &github.InstallationPermissions{
				Contents: github.String("read"),
			},
		},
	)
	if err != nil {
		return nil, "", err
	}

	dir, err := a.downloadRepository(ctx, c, r, commit)
	if err != nil {
		return nil, dir, err
	}

	// Remove the repository contents immediately after we load the packages
	// from it so that it doesn't spend any longer on disk than it needs to.
	defer os.RemoveAll(dir)

	cfg := &packages.Config{
		Context: ctx,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedDeps,
		Dir: dir,
		Env: append(
			os.Environ(),
			// This environment variable is read by the `askpass` binary, which
			// is part of this project.
			"_DOGMA_BROWSER_GITHUB_TOKEN="+token.GetToken(),
		),
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, "", err
	}

	return pkgs, dir, nil
}

// downloadRepository downloads the repository contents at the given commit and
// returns the name of a temporary directory containing the contents.
func (a *Analyzer) downloadRepository(
	ctx context.Context,
	c *github.Client,
	r *github.Repository,
	commit string,
) (string, error) {
	url, _, err := c.Repositories.GetArchiveLink(ctx,
		r.GetOwner().GetLogin(),
		r.GetName(),
		github.Tarball,
		&github.RepositoryContentGetOptions{
			Ref: commit,
		},
		true,
	)
	if err != nil {
		return "", err
	}

	var hc *http.Client
	if a.Connector.Transport != nil {
		hc = &http.Client{
			Transport: a.Connector.Transport,
		}
	}

	return githubx.GetArchive(
		ctx,
		hc,
		url.String(),
	)
}

func (a *Analyzer) analyzePackages(
	r *github.Repository,
	pkgs []*packages.Package,
	dir string,
) ([]configkit.Application, []persistence.TypeDef) {
	var (
		apps []configkit.Application
		defs []persistence.TypeDef
	)

	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			for _, err := range pkg.Errors {
				logging.Log(
					a.Logger,
					"[#%d %s] unable to analyze %s: %s",
					r.GetID(),
					r.GetFullName(),
					pkg.PkgPath,
					err,
				)
			}

			continue
		}

		logging.Log(
			a.Logger,
			"[#%d %s] analyzing %s",
			r.GetID(),
			r.GetFullName(),
			pkg.PkgPath,
		)

		a, d := a.analyzePackage(r, pkg, dir)
		apps = append(apps, a...)
		defs = append(defs, d...)
	}

	return apps, defs
}

func (a *Analyzer) analyzePackage(
	r *github.Repository,
	pkg *packages.Package,
	dir string,
) ([]configkit.Application, []persistence.TypeDef) {
	defer func() {
		if p := recover(); p != nil {
			logging.Log(
				a.Logger,
				"[#%d %s] recovered from panic: %s",
				r.GetID(),
				r.GetFullName(),
				p,
			)
		}
	}()

	apps := static.FromPackages([]*packages.Package{pkg})

	for _, app := range apps {
		logging.Log(
			a.Logger,
			"[#%d %s] discovered application: %s",
			r.GetID(),
			r.GetFullName(),
			app.Identity(),
		)
	}

	var defs []persistence.TypeDef

	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			d, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}

			if d.Tok != token.TYPE {
				continue
			}

			for _, s := range d.Specs {
				s := s.(*ast.TypeSpec)

				logging.Log(
					a.Logger,
					"[#%d %s] discovered type: %s",
					r.GetID(),
					r.GetFullName(),
					s.Name,
				)

				pos := pkg.Fset.Position(s.Pos())

				defs = append(defs, persistence.TypeDef{
					Package: pkg.PkgPath,
					Name:    s.Name.String(),
					File:    strings.TrimPrefix(pos.Filename, dir),
					Line:    pos.Line,
					Docs:    d.Doc.Text(),
				})
			}
		}
	}

	return apps, defs
}
