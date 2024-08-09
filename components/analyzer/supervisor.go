package analyzer

import (
	"context"
	"log/slog"
	"runtime"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/sync/errgroup"
)

// Supervisor oversees a pool of workers that perform static analysis of Go
// modules.
type Supervisor struct {
	Environment []string
	Workers     int
	Logger      *slog.Logger
}

// Run starts the analyzer.
func (a *Supervisor) Run(ctx context.Context) error {
	minibus.Subscribe[messages.GoModuleDownloaded](ctx)
	minibus.Ready(ctx)

	workers := a.Workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	g, ctx := errgroup.WithContext(ctx)
	for n := range workers {
		g.Go(func() error {
			w := &worker{
				Environment: a.Environment,
				Logger: a.Logger.With(
					slog.Group(
						"worker",
						slog.Int("id", n+1),
					),
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
