package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/sync/errgroup"
)

// Downloader downloads all discovered Go modules.
type Downloader struct {
	Environment []string
	Workers     int
	Logger      *slog.Logger
}

// Run starts the downloader.
func (d *Downloader) Run(ctx context.Context) (err error) {
	minibus.Subscribe[messages.GoModuleFound](ctx)
	minibus.Ready(ctx)

	workers := d.Workers
	if workers == 0 {
		workers = runtime.NumCPU() * 10
	}

	g, ctx := errgroup.WithContext(ctx)
	for n := range workers {
		g.Go(func() error {
			w := &downloaderWorker{
				Environment: d.Environment,
				Logger: d.Logger.With(
					slog.Int("worker", n+1),
				),
			}
			return w.Run(ctx)
		})
	}

	d.Logger.InfoContext(
		ctx,
		"started downloader",
		slog.Int("workers", workers),
	)

	return g.Wait()
}

type downloaderWorker struct {
	Environment []string
	Logger      *slog.Logger
}

func (w *downloaderWorker) Run(ctx context.Context) (err error) {
	for m := range minibus.Inbox(ctx) {
		switch m := m.(type) {
		case messages.GoModuleFound:
			if err := w.download(ctx, m); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *downloaderWorker) download(
	ctx context.Context,
	m messages.GoModuleFound,
) error {
	cmd := exec.CommandContext(
		ctx,
		"go",
		"mod",
		"download",
		"-json",
		fmt.Sprintf("%s@%s", m.ModulePath, m.ModuleVersion),
	)
	cmd.Env = w.Environment

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	var output struct {
		Dir     string
		Version string
		Error   string
	}

	runErr := cmd.Run()
	parseErr := json.
		NewDecoder(&stdout).
		Decode(&output)

	if parseErr == nil && output.Error != "" {
		return fmt.Errorf(
			"unable to download module: %s",
			output.Error,
		)
	}

	if runErr != nil {
		return fmt.Errorf(
			"unable to download module: %w",
			runErr,
		)
	}

	if parseErr != nil {
		return fmt.Errorf("unable to parse 'go mod download' output: %w", parseErr)
	}

	w.Logger.InfoContext(
		ctx,
		"downloaded go module",
		slog.String("mod.path", m.ModulePath),
		slog.String("mod.version", m.ModuleVersion),
		slog.String("mod.dir", output.Dir),
	)

	return minibus.Send(
		ctx,
		messages.GoModuleDownloaded{
			RepoSource:    m.RepoSource,
			RepoID:        m.RepoID,
			ModulePath:    m.ModulePath,
			ModuleVersion: output.Version,
			ModuleDir:     output.Dir,
		},
	)
}
