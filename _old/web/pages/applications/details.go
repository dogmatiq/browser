package applications

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/dogmatiq/browser/web/components"
	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/configkit/message"
	"github.com/gin-gonic/gin"
)

// detailsView is the template context for details.html.
type detailsView struct {
	Key  string
	Name string
	Impl components.Type

	Relationships []relationship
	Handlers      []handlerSummary
	Messages      []messageSummary
}

// relationship contains a summary of information about an application that is
// related to the application being displayed.
type relationship struct {
	Key                    string
	Name                   string
	Impl                   components.Type
	HasPointerMismatch     bool
	HasRoleMismatch        bool
	UpstreamMessageCount   int
	DownstreamMessageCount int
	TotalMessageCount      int
}

// handlerSummary contains a summary of information about a handler within an
// application, for display within a detailsView.
type handlerSummary struct {
	Key                  string
	Name                 string
	Type                 configkit.HandlerType
	Impl                 components.Type
	ConsumedMessageCount int
	ProducedMessageCount int
}

// handlerSummary contains a summary of information about a message used by an
// application, for display within a detailsView.
type messageSummary struct {
	Impl          components.Type
	Role          message.Role
	ProducerCount int
	ConsumerCount int
}

// DetailsHandler is an implementation of web.Handler that displays detailed
// information about a single Dogma application.
type DetailsHandler struct {
	DB *sql.DB
}

func (h *DetailsHandler) Route() (string, string) {
	return http.MethodGet, "/applications/:key"
}

func (h *DetailsHandler) Template() string {
	return "applications/details.html"
}

func (h *DetailsHandler) ActiveMenuItem() components.MenuItem {
	return components.ApplicationsMenuItem
}

func (h *DetailsHandler) View(ctx *gin.Context) (string, interface{}, error) {
	var view detailsView

	appKey := ctx.Param("key")

	if err := h.loadDetails(ctx, &view, appKey); err != nil {
		if err == sql.ErrNoRows {
			ctx.AbortWithStatus(http.StatusNotFound)
			return "", nil, nil
		}

		return "", nil, err
	}

	if err := h.loadRelationships(ctx, &view, appKey); err != nil {
		return "", nil, err
	}

	if err := h.loadHandlers(ctx, &view, appKey); err != nil {
		return "", nil, err
	}

	if err := h.loadMessages(ctx, &view, appKey); err != nil {
		return "", nil, err
	}

	return view.Name, view, nil
}

func (h *DetailsHandler) loadDetails(
	ctx context.Context,
	view *detailsView,
	appKey string,
) error {
	row := h.DB.QueryRowContext(
		ctx,
		`SELECT
			a.key,
			a.name,
			t.package,
			t.name,
			a.is_pointer,
			COALESCE(t.url, ''),
			COALESCE(t.docs, '')
		FROM dogmabrowser.application AS a
		INNER JOIN dogmabrowser.type AS t
		ON t.id = a.type_id
		WHERE a.key = $1`,
		appKey,
	)

	return row.Scan(
		&view.Key,
		&view.Name,
		&view.Impl.Package,
		&view.Impl.Name,
		&view.Impl.IsPointer,
		&view.Impl.URL,
		&view.Impl.Docs,
	)
}

func (h *DetailsHandler) loadRelationships(
	ctx context.Context,
	view *detailsView,
	appKey string,
) error {
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
			BOOL_OR(m.is_pointer != xm.is_pointer) AS has_pointer_mismatch,
			BOOL_OR(m.role != xm.role) AS has_role_mismatch,
			COUNT(DISTINCT m.type_id) FILTER (WHERE m.is_produced AND xm.is_consumed) AS upstream_count,
			COUNT(DISTINCT m.type_id) FILTER (WHERE m.is_consumed AND xm.is_produced) AS downstream_count,
			COUNT(DISTINCT m.type_id)
		FROM dogmabrowser.application AS a
		INNER JOIN dogmabrowser.type AS t
		ON t.id = a.type_id
		INNER JOIN dogmabrowser.handler AS h
		ON h.application_key = a.key
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
		WHERE xh.application_key = $1
		AND (
			(m.is_consumed AND xm.is_produced)
			OR (m.is_produced AND xm.is_consumed)
		)
		GROUP BY a.key, t.id
		ORDER BY a.name, a.key`,
		appKey,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var r relationship

		if err := rows.Scan(
			&r.Key,
			&r.Name,
			&r.Impl.Package,
			&r.Impl.Name,
			&r.Impl.IsPointer,
			&r.Impl.URL,
			&r.Impl.Docs,
			&r.HasPointerMismatch,
			&r.HasRoleMismatch,
			&r.UpstreamMessageCount,
			&r.DownstreamMessageCount,
			&r.TotalMessageCount,
		); err != nil {
			return err
		}

		view.Relationships = append(view.Relationships, r)
	}

	return rows.Err()
}

func (h *DetailsHandler) loadHandlers(
	ctx context.Context,
	view *detailsView,
	appKey string,
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
		WHERE h.application_key = $1
		ORDER BY h.name`,
		appKey,
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
			&s.ConsumedMessageCount,
			&s.ProducedMessageCount,
		); err != nil {
			return err
		}

		view.Handlers = append(view.Handlers, s)
	}

	return rows.Err()
}

func (h *DetailsHandler) loadMessages(
	ctx context.Context,
	view *detailsView,
	appKey string,
) error {
	rows, err := h.DB.QueryContext(
		ctx,
		`SELECT DISTINCT ON (t.name, t.package)
			t.package,
			t.name,
			m.is_pointer,
			COALESCE(t.url, ''),
			COALESCE(t.docs, ''),
			m.role,
			(
				SELECT COUNT(DISTINCT pm.handler_key)
				FROM dogmabrowser.handler_message AS pm
				WHERE pm.type_id = m.type_id
				AND pm.is_produced = TRUE
			) AS produced_count,
			(
				SELECT COUNT(DISTINCT pm.handler_key)
				FROM dogmabrowser.handler_message AS pm
				WHERE pm.type_id = m.type_id
				AND pm.is_consumed = TRUE
			) AS consumed_count
		FROM dogmabrowser.handler_message AS m
		INNER JOIN dogmabrowser.handler AS h
		ON h.key = m.handler_key
		INNER JOIN dogmabrowser.type AS t
		ON t.id = m.type_id
		WHERE h.application_key = $1
		ORDER BY t.name, t.package`,
		appKey,
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
			&s.Impl.IsPointer,
			&s.Impl.URL,
			&s.Impl.Docs,
			&s.Role,
			&s.ProducerCount,
			&s.ConsumerCount,
		); err != nil {
			return err
		}

		view.Messages = append(view.Messages, s)
	}

	return rows.Err()
}
