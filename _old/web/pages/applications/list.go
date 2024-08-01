package applications

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/dogmatiq/browser/web/components"
	"github.com/gin-gonic/gin"
)

// listView is the template context for list.html.
type listView struct {
	TotalRepoCount int
	Applications   []appSummary
}

// appSummary contains a summary of information about an application, for
// display within a listView.
type appSummary struct {
	Key               string
	Name              string
	Impl              components.Type
	RelationshipCount int
	HandlerCount      int
	MessageCount      int
}

// ListHandler is an implementation of web.Handler that displays a list of
// discovered Dogma applications.
type ListHandler struct {
	DB *sql.DB
}

func (h *ListHandler) Route() (string, string) {
	return http.MethodGet, "/"
}

func (h *ListHandler) Template() string {
	return "applications/list.html"
}

func (h *ListHandler) ActiveMenuItem() components.MenuItem {
	return components.ApplicationsMenuItem
}

func (h *ListHandler) View(ctx *gin.Context) (string, interface{}, error) {
	var view listView

	if err := h.loadStats(ctx, &view); err != nil {
		return "", nil, err
	}

	if err := h.loadApplications(ctx, &view); err != nil {
		return "", nil, err
	}

	return "Applications", view, nil
}

func (h *ListHandler) loadStats(ctx context.Context, view *listView) error {
	return h.DB.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		FROM dogmabrowser.repository`,
	).Scan(&view.TotalRepoCount)
}

func (h *ListHandler) loadApplications(ctx context.Context, view *listView) error {
	rows, err := h.DB.QueryContext(
		ctx,
		`SELECT
			a.key,
			a.name,
			t.package,
			t.name,
			a.is_pointer,
			COALESCE(t.url, ''),
			COALESCE(t.docs, ''),
			(
				SELECT COUNT(DISTINCT h.application_key)
				FROM dogmabrowser.handler AS h
				INNER JOIN dogmabrowser.handler_message AS m
				ON m.handler_key = h.key
				AND m.role != 'timeout'
				INNER JOIN dogmabrowser.handler_message AS xm
				ON xm.type_id = m.type_id
				AND xm.handler_key != m.handler_key
				AND xm.role != 'timeout'
				INNER JOIN dogmabrowser.handler AS xh
				ON xh.key = xm.handler_key
				AND xh.application_key != h.application_key
				WHERE xh.application_key = a.key
			) AS relationship_count,
			(
				SELECT COUNT(h.key)
				FROM dogmabrowser.handler AS h
				WHERE h.application_key = a.key
			) AS handler_count,
			(
				SELECT COUNT(DISTINCT m.type_id)
				FROM dogmabrowser.handler AS h
				INNER JOIN dogmabrowser.handler_message AS m
				ON m.handler_key = h.key
				WHERE h.application_key = a.key
			) AS message_count
		FROM dogmabrowser.application AS a
		INNER JOIN dogmabrowser.type AS t
		ON t.id = a.type_id
		ORDER BY a.name, a.key`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var s appSummary

		if err := rows.Scan(
			&s.Key,
			&s.Name,
			&s.Impl.Package,
			&s.Impl.Name,
			&s.Impl.IsPointer,
			&s.Impl.URL,
			&s.Impl.Docs,
			&s.RelationshipCount,
			&s.HandlerCount,
			&s.MessageCount,
		); err != nil {
			return err
		}

		view.Applications = append(view.Applications, s)
	}

	return rows.Err()
}
