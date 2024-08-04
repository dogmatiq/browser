package main

import (
	"log/slog"

	"github.com/dogmatiq/browser/components/analyzer"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

var (
	workerCount = ferrite.
		Signed[int]("WORKER_COUNT", "number of concurrent analysis workers").
		WithMinimum(1).
		WithDefault(1).
		Required()
)

func init() {
	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			logger *slog.Logger,
		) (*analyzer.Analyzer, error) {
			return &analyzer.Analyzer{
				Workers: workerCount.Value(),
				Logger:  logger,
			}, nil
		},
	)

	imbue.Decorate1(
		container,
		func(
			ctx imbue.Context,
			options []minibus.Option,
			a *analyzer.Analyzer,
		) ([]minibus.Option, error) {
			return append(
				options,
				minibus.WithFunc(a.Run),
			), nil
		},
	)
}
