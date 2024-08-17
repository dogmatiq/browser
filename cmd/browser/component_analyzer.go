package main

import (
	"log/slog"
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
			WithDefault(max(1, runtime.NumCPU()/2)).
			Required()

	analyzerCacheDir = ferrite.
				String("ANALYZER_CACHE_DIR", "the directory in which to store analysis results").
				Required()
)

func init() {
	imbue.With2(
		container,
		func(
			ctx imbue.Context,
			env imbue.ByName[environment, []string],
			logger *slog.Logger,
		) (*analyzer.Supervisor, error) {
			return &analyzer.Supervisor{
				Cache: &analyzer.DiskCache{
					Dir: analyzerCacheDir.Value(),
				},
				Environment: env.Value(),
				Workers:     analyzerWorkers.Value(),
				Logger: logger.With(
					slog.String("component", "analyzer"),
				),
			}, nil
		},
	)

	imbue.Decorate1(
		container,
		func(
			ctx imbue.Context,
			funcs []minibus.Func,
			s *analyzer.Supervisor,
		) ([]minibus.Func, error) {
			return append(funcs, s.Run), nil
		},
	)
}
