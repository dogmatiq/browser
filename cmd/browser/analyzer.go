package main

import (
	"log/slog"
	"os"
	"runtime"

	"github.com/dogmatiq/browser/components/analyzer"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

var (
	analyzerWorkers = ferrite.
			Signed[int]("ANALYZER_WORKERS", "the maximum number of Go modules to analyze concurrently").
			WithMinimum(1).
			WithDefault(runtime.NumCPU()).
			Required()

	downloadWorkers = ferrite.
			Signed[int]("DOWNLOADER_WORKERS", "the maximum number of Go modules to download concurrently").
			WithMinimum(1).
			WithDefault(runtime.NumCPU() * 4).
			Required()
)

func init() {
	bin, err := os.Executable()
	if err != nil {
		panic(err)
	}

	env := append(
		os.Environ(),
		"GIT_CONFIG_SYSTEM=",
		"GIT_CONFIG_GLOBAL=",
		"GIT_ASKPASS="+bin,
	)

	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			logger *slog.Logger,
		) (*analyzer.Analyzer, error) {
			return &analyzer.Analyzer{
				Environment: env,
				Workers:     analyzerWorkers.Value(),
				Logger:      logger,
			}, nil
		},
	)

	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			logger *slog.Logger,
		) (*analyzer.Downloader, error) {
			return &analyzer.Downloader{
				Environment: env,
				Workers:     downloadWorkers.Value(),
				Logger:      logger,
			}, nil
		},
	)

	imbue.Decorate2(
		container,
		func(
			ctx imbue.Context,
			options []minibus.Option,
			a *analyzer.Analyzer,
			d *analyzer.Downloader,
		) ([]minibus.Option, error) {
			return append(
				options,
				minibus.WithFunc(a.Run),
				minibus.WithFunc(d.Run),
			), nil
		},
	)
}
