package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/tools/go/packages"
)

type worker struct {
	Environment []string
	Logger      *slog.Logger
}

func (w *worker) Run(ctx context.Context) (err error) {
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

func (w *worker) analyze(
	ctx context.Context,
	m messages.GoModuleDownloaded,
) error {
	start := time.Now()

	logger := w.Logger.With(
		slog.Group(
			"module",
			slog.String("path", m.ModulePath),
			slog.String("version", m.ModuleVersion),
		),
	)

	logger.InfoContext(
		ctx,
		"analyzing go module",
	)

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
				logger.ErrorContext(
					ctx,
					"error loading package",
					slog.String("package_path", p.PkgPath),
					slog.String("error", err.Error()),
				)
			}
		}
	}

	logger.InfoContext(
		ctx,
		"analyzed go module",
		slog.Int("packages", count),
		slog.Duration("duration", time.Since(start)),
	)

	return nil
}
