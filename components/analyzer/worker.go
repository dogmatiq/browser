package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"

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
		case messages.GoModuleFound:
			if err := w.handleGoModuleFound(ctx, m); err != nil {
				return err
			}
		}
	}

	return nil
}

var once sync.Once

func (w *worker) handleGoModuleFound(
	ctx context.Context,
	m messages.GoModuleFound,
) error {
	w.Logger.InfoContext(
		ctx,
		"analyzing go module",
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
	)

	dir, err := w.downloadModule(ctx, m.ModulePath, m.ModuleVersion)
	if err != nil {
		return err
	}

	w.Logger.DebugContext(
		ctx,
		"downloaded go module",
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
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
		Dir: dir,
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
					slog.String("package_path", p.PkgPath),
					slog.String("error", err.Error()),
				)
			}
		}
	}

	w.Logger.InfoContext(
		ctx,
		"loaded packages from go module",
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
		slog.Int("package_count", count),
	)

	return nil
}

func (w *worker) downloadModule(
	ctx context.Context,
	path, version string,
) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		"go",
		"mod",
		"download",
		"-json",
		fmt.Sprintf("%s@%s", path, version),
	)
	cmd.Env = w.Environment

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	var output struct {
		Dir   string
		Error string
	}

	runErr := cmd.Run()
	parseErr := json.
		NewDecoder(&stdout).
		Decode(&output)

	if parseErr == nil && output.Error != "" {
		return "", fmt.Errorf(
			"unable to download module: %s",
			output.Error,
		)
	}

	if runErr != nil {
		return "", fmt.Errorf(
			"unable to download module: %w",
			runErr,
		)
	}

	if parseErr != nil {
		return "", fmt.Errorf("unable to parse 'go mod download' output: %w", parseErr)
	}

	return output.Dir, nil
}
