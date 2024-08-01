package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/dogmatiq/browser/persistence"
	"github.com/dogmatiq/imbue"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func init() {
	imbue.With0(
		container,
		func(
			ictx imbue.Context,
		) (*sql.DB, error) {
			ctx, cancel := context.WithTimeout(ictx, 3*time.Second)
			defer cancel()

			db, err := sql.Open("pgx", postgresDSN.Value())
			if err != nil {
				return nil, err
			}

			db.SetConnMaxIdleTime(30 * time.Second)
			db.SetConnMaxLifetime(5 * time.Minute)
			db.SetMaxIdleConns(3)
			db.SetMaxOpenConns(50)

			if err := persistence.CreateSchema(ctx, db); err != nil {
				return nil, err
			}

			return db, nil
		},
	)
}
