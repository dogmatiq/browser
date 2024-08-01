package main

import (
	"database/sql"

	"github.com/dogmatiq/browser/analyzer"
	"github.com/dogmatiq/browser/githubx"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/imbue"
)

func init() {
	imbue.With3(
		container,
		func(
			ctx imbue.Context,
			db *sql.DB,
			c *githubx.Connector,
			l logging.Logger,
		) (*analyzer.Analyzer, error) {
			return &analyzer.Analyzer{
				DB:        db,
				Connector: c,
				Logger:    l,
			}, nil
		},
	)

	imbue.With2(
		container,
		func(
			ctx imbue.Context,
			db *sql.DB,
			l logging.Logger,
		) (*analyzer.Remover, error) {
			return &analyzer.Remover{
				DB:     db,
				Logger: l,
			}, nil
		},
	)

	imbue.With2(
		container,
		func(
			ctx imbue.Context,
			a *analyzer.Analyzer,
			r *analyzer.Remover,
		) (*analyzer.Orchestrator, error) {
			return &analyzer.Orchestrator{
				Analyzer: a,
				Remover:  r,
			}, nil
		},
	)
}
