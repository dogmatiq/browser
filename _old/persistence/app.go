package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dogmatiq/configkit"
	"github.com/google/go-github/v38/github"
)

func syncApplications(
	ctx context.Context,
	tx *sql.Tx,
	r *github.Repository,
	apps []configkit.Application,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE dogmabrowser.application SET
			needs_removal = TRUE
		WHERE repository_id = $1`,
		r.GetID(),
	); err != nil {
		return fmt.Errorf("unable to mark applications for removal: %w", err)
	}

	for _, a := range apps {
		if err := syncApplication(ctx, tx, r, a); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM dogmabrowser.application
		WHERE repository_id = $1
		AND needs_removal`,
		r.GetID(),
	); err != nil {
		return fmt.Errorf("unable remove applications: %w", err)
	}

	return nil
}

func syncApplication(
	ctx context.Context,
	tx *sql.Tx,
	r *github.Repository,
	a configkit.Application,
) error {
	typeID, isPointer, err := syncTypeRef(ctx, tx, a.TypeName())
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO dogmabrowser.application (
			key,
			name,
			type_id,
			is_pointer,
			repository_id
		) VALUES (
			$1, $2, $3, $4, $5
		) ON CONFLICT (key) DO UPDATE SET
			name = excluded.name,
			type_id = excluded.type_id,
			is_pointer = excluded.is_pointer,
			repository_id = excluded.repository_id,
			needs_removal = FALSE`,
		a.Identity().Key,
		a.Identity().Name,
		typeID,
		isPointer,
		r.GetID(),
	); err != nil {
		return fmt.Errorf("unable to sync application: %w", err)
	}

	return syncHandlers(ctx, tx, a)
}

func syncHandlers(
	ctx context.Context,
	tx *sql.Tx,
	a configkit.Application,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE dogmabrowser.handler SET
			needs_removal = TRUE
		WHERE application_key = $1`,
		a.Identity().Key,
	); err != nil {
		return fmt.Errorf("unable to mark handlers for removal: %w", err)
	}

	for _, h := range a.Handlers() {
		if err := syncHandler(
			ctx,
			tx,
			a.Identity().Key,
			h,
		); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM dogmabrowser.handler
		WHERE application_key = $1
		AND needs_removal`,
		a.Identity().Key,
	); err != nil {
		return fmt.Errorf("unable to remove handlers: %w", err)
	}

	return nil
}

func syncHandler(
	ctx context.Context,
	tx *sql.Tx,
	appKey string,
	h configkit.Handler,
) error {
	typeID, isPointer, err := syncTypeRef(ctx, tx, h.TypeName())
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO dogmabrowser.handler (
			key,
			name,
			application_key,
			handler_type,
			type_id,
			is_pointer
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) ON CONFLICT (key) DO UPDATE SET
			name = excluded.name,
			application_key = excluded.application_key,
			handler_type = excluded.handler_type,
			type_id = excluded.type_id,
			is_pointer = excluded.is_pointer,
			needs_removal = FALSE`,
		h.Identity().Key,
		h.Identity().Name,
		appKey,
		h.HandlerType(),
		typeID,
		isPointer,
	); err != nil {
		return fmt.Errorf("unable to sync handler: %w", err)
	}

	return syncMessages(ctx, tx, h)
}

func syncMessages(
	ctx context.Context,
	tx *sql.Tx,
	h configkit.Handler,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE dogmabrowser.handler_message SET
			needs_removal = TRUE
		WHERE handler_key = $1`,
		h.Identity().Key,
	); err != nil {
		return fmt.Errorf("unable to mark messages for removal: %w", err)
	}

	for n, r := range h.MessageNames().Produced {
		typeID, isPointer, err := syncTypeRef(ctx, tx, n.String())
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO dogmabrowser.handler_message (
				handler_key,
				type_id,
				is_pointer,
				role,
				is_produced
			) VALUES (
				$1, $2, $3, $4, TRUE
			) ON CONFLICT (handler_key, type_id, is_pointer) DO UPDATE SET
				role = excluded.role,
				is_produced = excluded.is_produced,
				needs_removal = FALSE`,
			h.Identity().Key,
			typeID,
			isPointer,
			r,
		); err != nil {
			return fmt.Errorf("unable to sync message: %w", err)
		}
	}

	for n, r := range h.MessageNames().Consumed {
		typeID, isPointer, err := syncTypeRef(ctx, tx, n.String())
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO dogmabrowser.handler_message (
				handler_key,
				type_id,
				is_pointer,
				role,
				is_consumed
			) VALUES (
				$1, $2, $3, $4, TRUE
			) ON CONFLICT (handler_key, type_id, is_pointer) DO UPDATE SET
				role = excluded.role,
				is_consumed = excluded.is_consumed,
				needs_removal = FALSE`,
			h.Identity().Key,
			typeID,
			isPointer,
			r,
		); err != nil {
			return fmt.Errorf("unable to sync message: %w", err)
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM dogmabrowser.handler_message
		WHERE handler_key = $1
		AND needs_removal`,
		h.Identity().Key,
	); err != nil {
		return fmt.Errorf("unable to remove messages: %w", err)
	}

	return nil
}
