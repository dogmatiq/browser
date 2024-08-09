package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
)

type worker struct {
	Environment []string
	Logger      *slog.Logger
}

func (w *worker) Run(ctx context.Context) (err error) {
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

func (w *worker) download(
	ctx context.Context,
	m messages.GoModuleFound,
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
		"downloading go module",
	)

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

	logger.InfoContext(
		ctx,
		"downloaded go module",
		slog.Duration("elapsed", time.Since(start)),
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
