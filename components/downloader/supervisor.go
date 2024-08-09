package downloader

import (
	"context"
	"log/slog"
	"runtime"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/sync/errgroup"
)

// Supervisor oversees a pool of works that download Go modules.
type Supervisor struct {
	Environment []string
	Workers     int
	Logger      *slog.Logger
}

// Run starts the supervisor.
func (d *Supervisor) Run(ctx context.Context) (err error) {
	minibus.Subscribe[messages.GoModuleFound](ctx)
	minibus.Ready(ctx)

	workers := d.Workers
	if workers == 0 {
		workers = runtime.NumCPU() * 10
	}

	g, ctx := errgroup.WithContext(ctx)
	for n := range workers {
		g.Go(func() error {
			w := &worker{
				Environment: d.Environment,
				Logger: d.Logger.With(
					slog.Group(
						"worker",
						slog.Int("id", n+1),
					),
				),
			}
			return w.Run(ctx)
		})
	}

	d.Logger.InfoContext(
		ctx,
		"started download supervisor",
		slog.Int("workers", workers),
	)

	return g.Wait()
}
