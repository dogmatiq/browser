package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/google/go-github/v38/github"
)

type TypeDef struct {
	Package string
	Name    string
	File    string
	Line    int
	Docs    string
}

func syncTypeRef(
	ctx context.Context,
	tx *sql.Tx,
	name string,
) (typeID int, isPointer bool, err error) {
	if name[0] == '*' {
		isPointer = true
		name = name[1:]
	}

	var pkg string
	if n := strings.LastIndexByte(name, '.'); n != -1 {
		pkg = name[:n]
		name = name[n+1:]
	}

	row := tx.QueryRowContext(
		ctx,
		`INSERT INTO dogmabrowser.type (
			package,
			name
		) VALUES (
			$1, $2
		) ON CONFLICT (package, name) DO UPDATE SET
			package = excluded.package
		RETURNING id`, // DO UPDATE is a no-op that allows use of RETURNING when row already exists
		pkg,
		name,
	)

	if err := row.Scan(&typeID); err != nil {
		return 0, false, fmt.Errorf("unable to sync type reference: %w", err)
	}

	return typeID, isPointer, nil
}

func syncTypeDefs(
	ctx context.Context,
	tx *sql.Tx,
	r *github.Repository,
	defs []TypeDef,
	commit string,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE dogmabrowser.type SET
			needs_removal = TRUE
		WHERE repository_id = $1`,
		r.GetID(),
	); err != nil {
		return fmt.Errorf("unable to mark types for removal: %w", err)
	}

	for _, t := range defs {
		if err := syncTypeDef(
			ctx,
			tx,
			r,
			t,
			commit,
		); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM dogmabrowser.type AS t
		WHERE repository_id = $1
		AND needs_removal
		AND NOT EXISTS (SELECT * FROM dogmabrowser.application WHERE type_id = t.id)
		AND NOT EXISTS (SELECT * FROM dogmabrowser.handler WHERE type_id = t.id)
		AND NOT EXISTS (SELECT * FROM dogmabrowser.handler_message WHERE type_id = t.id)`,
		r.GetID(),
	); err != nil {
		return fmt.Errorf("unable to remove types: %w", err)
	}

	return nil
}

func syncTypeDef(
	ctx context.Context,
	tx *sql.Tx,
	r *github.Repository,
	t TypeDef,
	commit string,
) error {
	us := r.GetHTMLURL()

	u, err := url.Parse(us)
	if err != nil {
		return err
	}

	u.Path = path.Join(
		u.Path,
		"blob",
		commit,
		t.File,
	)

	u.Fragment = fmt.Sprintf("L%d", t.Line)

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO dogmabrowser.type (
			package,
			name,
			repository_id,
			url,
			docs
		) VALUES (
			$1, $2, $3, $4, $5
		) ON CONFLICT (package, name) DO UPDATE SET
			repository_id = excluded.repository_id,
			url = excluded.url,
			docs = excluded.docs,
			needs_removal = FALSE`,
		t.Package,
		t.Name,
		r.GetID(),
		u.String(),
		t.Docs,
	); err != nil {
		return fmt.Errorf("unable to sync type definition: %w", err)
	}

	return nil
}
