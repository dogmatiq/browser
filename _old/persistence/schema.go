package persistence

import (
	"context"
	"database/sql"
	_ "embed"
)

//go:embed schema.sql
var schemaDDL string

func CreateSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, schemaDDL)
	return err
}

func DropSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `DROP SCHEMA IF EXISTS dogmabrowser CASCADE`)
	return err
}
