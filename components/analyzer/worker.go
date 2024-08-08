package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/tools/go/packages"
)

type worker struct {
	Logger *slog.Logger
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
		"downloading module",
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
	)

	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to determine executable path: %w", err)
	}

	env := append(
		os.Environ(),
		"GIT_CONFIG_SYSTEM=",
		"GIT_CONFIG_GLOBAL=",
		"GIT_ASKPASS="+bin,
	)

	cmd := exec.CommandContext(
		ctx,
		"go",
		"mod",
		"download",
		"-json",
		fmt.Sprintf("%s@%s", m.ModulePath, m.ModuleVersion),
	)
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		var output struct {
			Error string
		}

		json.
			NewDecoder(&stdout).
			Decode(&output)

		if output.Error != "" {
			return fmt.Errorf("unable to download module: %s", output.Error)
		}

		return fmt.Errorf(
			"analyzer: unable to download module: %w: %s",
			err,
			strings.ReplaceAll(
				stderr.String(),
				"\n",
				" ",
			),
		)
	}

	var output struct {
		Dir string
	}

	if err := json.
		NewDecoder(&stdout).
		Decode(&output); err != nil {
		return fmt.Errorf("unable to parse 'go mod download' output: %w", err)
	}

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
		Dir: output.Dir,
		Env: env,
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
