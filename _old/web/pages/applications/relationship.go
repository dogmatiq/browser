package applications

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/dogmatiq/browser/web/components"
	"github.com/dogmatiq/configkit/message"
	"github.com/gin-gonic/gin"
)

type relationshipView struct {
	Left  relatedApp
	Right relatedApp

	UpstreamMessages   []relatedMessage
	DownstreamMessages []relatedMessage
}

type relatedApp struct {
	Key  string
	Name string
	Impl components.Type
}

type relatedMessage struct {
	Impl               components.Type
	Role               message.Role
	HasRoleMismatch    bool
	HasPointerMismatch bool
	ProducerCount      int
	ConsumerCount      int
}

type RelationshipHandler struct {
	DB *sql.DB
}

func (h *RelationshipHandler) Route() (string, string) {
	return http.MethodGet, "/relationships/:rel"
}

func (h *RelationshipHandler) Template() string {
	return "applications/relationship.html"
}

func (h *RelationshipHandler) ActiveMenuItem() components.MenuItem {
	return components.ApplicationsMenuItem
}

func (h *RelationshipHandler) View(ctx *gin.Context) (string, interface{}, error) {
	var view relationshipView

	rel := ctx.Param("rel")
	n := strings.Index(rel, "...")
	if n == -1 {
		ctx.AbortWithStatus(http.StatusNotFound)
		return "", nil, nil
	}

	leftAppKey := rel[:n]
	rightAppKey := rel[n+3:]

	if err := h.loadApplication(ctx, &view.Left, leftAppKey); err != nil {
		if err == sql.ErrNoRows {
			ctx.AbortWithStatus(http.StatusNotFound)
			return "", nil, nil
		}

		return "", nil, err
	}

	if err := h.loadApplication(ctx, &view.Right, rightAppKey); err != nil {
		if err == sql.ErrNoRows {
			ctx.AbortWithStatus(http.StatusNotFound)
			return "", nil, nil
		}

		return "", nil, err
	}

	if err := h.loadMessages(
		ctx,
		&view.DownstreamMessages,
		leftAppKey,
		rightAppKey,
	); err != nil {
		return "", nil, err
	}

	if err := h.loadMessages(
		ctx,
		&view.UpstreamMessages,
		rightAppKey,
		leftAppKey,
	); err != nil {
		return "", nil, err
	}

	return view.Left.Name + "..." + view.Right.Name, view, nil
}

func (h *RelationshipHandler) loadApplication(
	ctx context.Context,
	a *relatedApp,
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
		&a.Key,
		&a.Name,
		&a.Impl.Package,
		&a.Impl.Name,
		&a.Impl.IsPointer,
		&a.Impl.URL,
		&a.Impl.Docs,
	)
}

func (h *RelationshipHandler) loadMessages(
	ctx context.Context,
	messages *[]relatedMessage,
	producerAppKey string,
	consumerAppKey string,
) error {
	rows, err := h.DB.QueryContext(
		ctx,
		`SELECT
			t.package,
			t.name,
			MODE() WITHIN GROUP (ORDER BY pm.is_pointer),
			COALESCE(t.url, ''),
			COALESCE(t.docs, ''),
			MODE() WITHIN GROUP (ORDER BY pm.role),
			BOOL_OR(pm.role != cm.role) AS has_role_mismatch,
			BOOL_OR(pm.is_pointer != cm.is_pointer) AS has_pointer_mismatch,
			COUNT(DISTINCT pm.handler_key) AS producer_count,
			COUNT(DISTINCT cm.handler_key) AS consumer_count
		FROM dogmabrowser.handler_message AS pm
		INNER JOIN dogmabrowser.type AS t
		ON t.id = pm.type_id
		INNER JOIN dogmabrowser.handler_message AS cm
		ON cm.type_id = t.id
		INNER JOIN dogmabrowser.handler AS ph
		ON ph.key = pm.handler_key
		INNER JOIN dogmabrowser.handler AS ch
		ON ch.key = cm.handler_key
		WHERE ph.application_key = $1
		AND pm.is_produced
		AND pm.role != 'timeout'
		AND ch.application_key = $2
		AND cm.is_consumed
		AND cm.role != 'timeout'
		GROUP BY t.id
		ORDER BY t.name, t.package`,
		producerAppKey,
		consumerAppKey,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var r relatedMessage

		if err := rows.Scan(
			&r.Impl.Package,
			&r.Impl.Name,
			&r.Impl.IsPointer,
			&r.Impl.URL,
			&r.Impl.Docs,
			&r.Role,
			&r.HasRoleMismatch,
			&r.HasPointerMismatch,
			&r.ProducerCount,
			&r.ConsumerCount,
		); err != nil {
			return err
		}

		*messages = append(*messages, r)
	}

	return rows.Err()
}
