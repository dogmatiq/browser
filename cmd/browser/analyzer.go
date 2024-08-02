package main

import (
	"log/slog"

	"github.com/dogmatiq/browser/components/analyzer"
	"github.com/dogmatiq/imbue"
)

func init() {
	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			logger *slog.Logger,
		) (*analyzer.Analyzer, error) {
			return &analyzer.Analyzer{
				Logger: logger,
			}, nil
		},
	)
}
