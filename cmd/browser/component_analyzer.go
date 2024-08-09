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
		WithDefault(runtime.NumCPU()).
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
			options []minibus.Option,
			s *analyzer.Supervisor,
		) ([]minibus.Option, error) {
			return append(
				options,
				minibus.WithFunc(s.Run),
			), nil
		},
	)
}
