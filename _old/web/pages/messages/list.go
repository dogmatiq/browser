package messages

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/dogmatiq/browser/web/components"
	"github.com/gin-gonic/gin"
)

type listView struct {
	TotalRepoCount    int
	TotalAppCount     int
	TotalHandlerCount int
	Messages          []messageSummary
}

type messageSummary struct {
	Impl               components.Type
	Role               string
	HasRoleMismatch    bool
	HasPointerMismatch bool
	AppCount           int
	ProducerCount      int
	ConsumerCount      int
}

type ListHandler struct {
	DB *sql.DB
}

func (h *ListHandler) Route() (string, string) {
	return http.MethodGet, "/messages"
}

func (h *ListHandler) Template() string {
	return "messages/list.html"
}

func (h *ListHandler) ActiveMenuItem() components.MenuItem {
	return components.MessagesMenuItem
}

func (h *ListHandler) View(ctx *gin.Context) (string, interface{}, error) {
	var view listView

	if err := h.loadStats(ctx, &view); err != nil {
		return "", nil, err
	}

	if err := h.loadMessages(ctx, &view); err != nil {
		return "", nil, err
	}

	return "Messages", view, nil
}

func (h *ListHandler) loadStats(
	ctx context.Context,
	view *listView,
) error {
	row := h.DB.QueryRowContext(
		ctx,
		`SELECT
			(SELECT COUNT(*) FROM dogmabrowser.repository),
			(SELECT COUNT(*) FROM dogmabrowser.application),
			(SELECT COUNT(*) FROM dogmabrowser.handler)`,
	)
	return row.Scan(
		&view.TotalRepoCount,
		&view.TotalAppCount,
		&view.TotalHandlerCount,
	)
}

func (h *ListHandler) loadMessages(
	ctx context.Context,
	view *listView,
) error {
	rows, err := h.DB.QueryContext(
		ctx,
		`SELECT
			t.package,
			t.name,
			COALESCE(t.url, ''),
			COALESCE(t.docs, ''),
			MODE() WITHIN GROUP (ORDER BY m.role) AS role,
			COUNT(DISTINCT m.role) > 1 AS has_role_mismatch,
			COUNT(DISTINCT m.is_pointer) > 1 AS has_pointer_mismatch,
			COUNT(DISTINCT a.key),
			COUNT(DISTINCT CASE WHEN m.is_produced THEN h.key END),
			COUNT(DISTINCT CASE WHEN m.is_consumed THEN h.key END)
		FROM dogmabrowser.handler_message AS m
		INNER JOIN dogmabrowser.type AS t
		ON t.id = m.type_id
		INNER JOIN dogmabrowser.handler AS h
		ON h.key = m.handler_key
		INNER JOIN dogmabrowser.application AS a
		ON a.key = h.application_key
		GROUP BY t.id
		ORDER BY t.name, t.package`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var s messageSummary

		if err := rows.Scan(
			&s.Impl.Package,
			&s.Impl.Name,
			&s.Impl.URL,
			&s.Impl.Docs,
			&s.Role,
			&s.HasRoleMismatch,
			&s.HasPointerMismatch,
			&s.AppCount,
			&s.ProducerCount,
			&s.ConsumerCount,
		); err != nil {
			return err
		}

		view.Messages = append(view.Messages, s)
	}

	return rows.Err()
}
