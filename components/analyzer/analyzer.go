package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/packages"
)

// Analyzer performs static analysis on all Go modules that are downloaded.
type Analyzer struct {
	Environment []string
	Workers     int
	Logger      *slog.Logger
}

// Run starts the analyzer.
func (a *Analyzer) Run(ctx context.Context) error {
	minibus.Subscribe[messages.GoModuleDownloaded](ctx)
	minibus.Ready(ctx)

	workers := a.Workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	g, ctx := errgroup.WithContext(ctx)
	for n := range workers {
		g.Go(func() error {
			w := &analyzerWorker{
				Environment: a.Environment,
				Logger: a.Logger.With(
					slog.Int("worker", n+1),
				),
			}
			return w.Run(ctx)
		})
	}

	a.Logger.InfoContext(
		ctx,
		"started analyzer",
		slog.Int("workers", workers),
	)

	return g.Wait()
}

type analyzerWorker struct {
	Environment []string
	Logger      *slog.Logger
}

func (w *analyzerWorker) Run(ctx context.Context) (err error) {
	for m := range minibus.Inbox(ctx) {
		switch m := m.(type) {
		case messages.GoModuleDownloaded:
			if err := w.analyze(ctx, m); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *analyzerWorker) analyze(
	ctx context.Context,
	m messages.GoModuleDownloaded,
) error {
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
		Dir: m.ModuleDir,
		Env: w.Environment,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("unable to load packages: %w", err)
	}

	count := 0
	for _, p := range pkgs {
		if p.Errors == nil {
			count++
		} else {
			for _, err := range p.Errors {
				w.Logger.ErrorContext(
					ctx,
					"error loading package",
					slog.String("mod.path", m.ModulePath),
					slog.String("mod.version", m.ModuleVersion),
					slog.String("pkg.path", p.PkgPath),
					slog.String("error", err.Error()),
				)
			}
		}
	}

	w.Logger.InfoContext(
		ctx,
		"analyzed go module",
		slog.String("mod.path", m.ModulePath),
		slog.String("mod.version", m.ModuleVersion),
		slog.Int("pkg.count", count),
	)

	return nil
}
