package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dogmatiq/configkit"
	"github.com/google/go-github/v38/github"
)

func RepositoryNeedsSync(
	ctx context.Context,
	db *sql.DB,
	r *github.Repository,
	commit string,
) (bool, error) {
	row := db.QueryRowContext(
		ctx,
		`SELECT NOT EXISTS (
			SELECT *
			FROM dogmabrowser.repository
			WHERE id = $1
			AND commit_hash = $2
			AND is_stale = FALSE
		)`,
		r.GetID(),
		commit,
	)

	var ok bool
	if err := row.Scan(&ok); err != nil {
		return false, fmt.Errorf("unable to check if repository needs sync: %w", err)
	}

	return ok, nil
}

func RemoveRepository(
	ctx context.Context,
	tx *sql.Tx,
	repoID int64,
) error {
	// Un-link the type definitions from the repository so that we can delete
	// the repository without removing basic type information.
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE dogmabrowser.type SET
			repository_id = NULL,
			url = NULL
		WHERE repository_id = $1`,
		repoID,
	); err != nil {
		return fmt.Errorf("unable to decouple types from repository: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM dogmabrowser.repository
		WHERE id = $1`,
		repoID,
	); err != nil {
		return fmt.Errorf("unable to remove repository: %w", err)
	}

	return nil
}

func SyncRepository(
	ctx context.Context,
	db *sql.DB,
	r *github.Repository,
	commit string,
	apps []configkit.Application,
	defs []TypeDef,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint:errcheck

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO dogmabrowser.repository AS r (
			id,
			full_name,
			commit_hash
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (id) DO UPDATE SET
			full_name = excluded.full_name,
			commit_hash = excluded.commit_hash,
			is_stale = FALSE`,
		r.GetID(),
		r.GetFullName(),
		commit,
	); err != nil {
		return fmt.Errorf("unable to sync repository: %w", err)
	}

	if err := syncApplications(ctx, tx, r, apps); err != nil {
		return err
	}

	if err := syncTypeDefs(ctx, tx, r, defs, commit); err != nil {
		return err
	}

	return tx.Commit()
}
