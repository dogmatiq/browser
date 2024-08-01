package web

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type searchItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Docs string `json:"docs,omitempty"`
	URI  string `json:"uri"`
}

func searchItems(version string, db *sql.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var items []searchItem

		i, err := applicationSearchItems(ctx, db)
		if err != nil {
			fmt.Println(err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			return
		}
		items = append(items, i...)

		i, err = handlerSearchItems(ctx, db)
		if err != nil {
			fmt.Println(err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			return
		}
		items = append(items, i...)

		i, err = messageSearchItems(ctx, db)
		if err != nil {
			fmt.Println(err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			return
		}
		items = append(items, i...)

		ctx.PureJSON(http.StatusOK, items)
	}
}

func applicationSearchItems(ctx context.Context, db *sql.DB) ([]searchItem, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT
			a.key,
			a.name,
			COALESCE(t.docs, '')
		FROM dogmabrowser.application AS a
		INNER JOIN dogmabrowser.type AS t
		ON t.id = a.type_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to query applications: %w", err)
	}
	defer rows.Close()

	var items []searchItem

	for rows.Next() {
		var (
			key  string
			item searchItem
		)

		if err := rows.Scan(
			&key,
			&item.Name,
			&item.Docs,
		); err != nil {
			return nil, fmt.Errorf("unable scan application result: %w", err)
		}

		item.Type = "application"
		item.URI = "/applications/" + key

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("unable to iterate all application rows: %w", err)
	}

	return items, nil
}

func handlerSearchItems(ctx context.Context, db *sql.DB) ([]searchItem, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT
			h.key,
			h.name,
			h.handler_type,
			COALESCE(t.docs, '')
		FROM dogmabrowser.handler AS h
		INNER JOIN dogmabrowser.type AS t
		ON t.id = h.type_id
		INNER JOIN dogmabrowser.application AS a
		ON a.key = h.application_key`,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to query handlers: %w", err)
	}
	defer rows.Close()

	var items []searchItem

	for rows.Next() {
		var (
			key  string
			item searchItem
		)

		if err := rows.Scan(
			&key,
			&item.Name,
			&item.Type,
			&item.Docs,
		); err != nil {
			return nil, fmt.Errorf("unable scan handler result: %w", err)
		}

		item.URI = "/handlers/" + key

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("unable to iterate all handler rows: %w", err)
	}

	return items, nil
}

func messageSearchItems(ctx context.Context, db *sql.DB) ([]searchItem, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT
			t.package,
			t.name,
			MODE() WITHIN GROUP (ORDER BY m.role),
			COALESCE(t.docs, '')
		FROM dogmabrowser.handler_message AS m
		INNER JOIN dogmabrowser.type AS t
		ON t.id = m.type_id
		GROUP BY t.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to query handlers: %w", err)
	}
	defer rows.Close()

	var items []searchItem

	for rows.Next() {
		var (
			pkg  string
			item searchItem
		)

		if err := rows.Scan(
			&pkg,
			&item.Name,
			&item.Type,
			&item.Docs,
		); err != nil {
			return nil, fmt.Errorf("unable scan handler result: %w", err)
		}

		item.URI = "/messages/" + pkg + "." + item.Name

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("unable to iterate all handler rows: %w", err)
	}

	return items, nil
}
