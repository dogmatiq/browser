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
			w := &worker{
				Logger: a.Logger.With(
					slog.Int("worker_id", n+1),
				),
			}
			return w.Run(ctx)
		})
	}

	a.Logger.InfoContext(
		ctx,
		"started analyzer",
		slog.Int("worker_count", workers),
	)

	return g.Wait()
}
