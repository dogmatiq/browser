package analyzer

import (
	"context"
	"log/slog"
	"runtime"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/sync/errgroup"
)

type Analyzer struct {
	Workers int
	Logger  *slog.Logger
}

func (a *Analyzer) Run(ctx context.Context) error {
	minibus.Subscribe[messages.GoModuleFound](ctx)
	minibus.Ready(ctx)

	workers := a.Workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	g, ctx := errgroup.WithContext(ctx)
	for n := range workers {
		g.Go(func() error {
			return a.runWorker(ctx, n+1)
		})
	}

	a.Logger.InfoContext(
		ctx,
		"started analyzer",
		slog.Int("worker_count", workers),
	)

	return g.Wait()
}

func (a *Analyzer) runWorker(ctx context.Context, id int) error {
	for m := range minibus.Inbox(ctx) {
		switch m := m.(type) {
		case messages.GoModuleFound:
			a.Logger.InfoContext(
				ctx,
				"analyzing go module",
				slog.Int("worker_id", id),
				slog.String("module_path", m.ModulePath),
				slog.String("module_version", m.ModuleVersion),
			)
		}
	}

	return ctx.Err()
}
