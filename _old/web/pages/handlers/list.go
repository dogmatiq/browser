package handlers

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/dogmatiq/browser/web/components"
	"github.com/dogmatiq/configkit"
	"github.com/gin-gonic/gin"
)

type listView struct {
	TotalRepoCount int
	TotalAppCount  int
	Handlers       []handlerSummary
}

type handlerSummary struct {
	Key                  string
	Name                 string
	Type                 configkit.HandlerType
	Impl                 components.Type
	AppKey               string
	AppName              string
	ConsumedMessageCount int
	ProducedMessageCount int
}

type ListHandler struct {
	DB *sql.DB
}

func (h *ListHandler) Route() (string, string) {
	return http.MethodGet, "/handlers"
}

func (h *ListHandler) Template() string {
	return "handlers/list.html"
}

func (h *ListHandler) ActiveMenuItem() components.MenuItem {
	return components.HandlersMenuItem
}

func (h *ListHandler) View(ctx *gin.Context) (string, interface{}, error) {
	var view listView

	if err := h.loadStats(ctx, &view); err != nil {
		return "", nil, err
	}

	if err := h.loadHandlers(ctx, &view); err != nil {
		return "", nil, err
	}

	return "Handlers", view, nil
}

func (h *ListHandler) loadStats(
	ctx context.Context,
	view *listView,
) error {
	row := h.DB.QueryRowContext(
		ctx,
		`SELECT
			(SELECT COUNT(*) FROM dogmabrowser.repository),
			(SELECT COUNT(*) FROM dogmabrowser.application)`,
	)
	return row.Scan(
		&view.TotalRepoCount,
		&view.TotalAppCount,
	)
}

func (h *ListHandler) loadHandlers(
	ctx context.Context,
	view *listView,
) error {
	rows, err := h.DB.QueryContext(
		ctx,
		`SELECT
			h.key,
			h.name,
			h.handler_type,
			t.package,
			t.name,
			h.is_pointer,
			COALESCE(t.url, ''),
			COALESCE(t.docs, ''),
			a.key,
			a.name,
			(
				SELECT COUNT(DISTINCT m.type_id)
				FROM dogmabrowser.handler_message AS m
				WHERE m.handler_key = h.key
				AND m.is_consumed
			) AS consumed_count,
			(
				SELECT COUNT(DISTINCT m.type_id)
				FROM dogmabrowser.handler_message AS m
				WHERE m.handler_key = h.key
				AND m.is_produced
			) AS produced_count
		FROM dogmabrowser.handler AS h
		INNER JOIN dogmabrowser.type AS t
		ON t.id = h.type_id
		INNER JOIN dogmabrowser.application AS a
		ON a.key = h.application_key
		ORDER BY h.name, a.name`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var s handlerSummary

		if err := rows.Scan(
			&s.Key,
			&s.Name,
			&s.Type,
			&s.Impl.Package,
			&s.Impl.Name,
			&s.Impl.IsPointer,
			&s.Impl.URL,
			&s.Impl.Docs,
			&s.AppKey,
			&s.AppName,
			&s.ConsumedMessageCount,
			&s.ProducedMessageCount,
		); err != nil {
			return err
		}

		view.Handlers = append(view.Handlers, s)
	}

	return rows.Err()
}
