package messages

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/dogmatiq/browser/web/components"
	"github.com/dogmatiq/configkit"
	"github.com/gin-gonic/gin"
)

type detailsView struct {
	Impl               components.Type
	Role               string
	HasRoleMismatch    bool
	HasPointerMismatch bool

	Applications []applicationSummary
	Producers    []handlerSummary
	Consumers    []handlerSummary
}

type applicationSummary struct {
	Key  string
	Name string
	Impl components.Type
}

type handlerSummary struct {
	Key     string
	Name    string
	Type    configkit.HandlerType
	Impl    components.Type
	AppName string
	AppKey  string
}

type DetailsHandler struct {
	DB *sql.DB
}

func (h *DetailsHandler) Route() (string, string) {
	return http.MethodGet, "/messages/*name"
}

func (h *DetailsHandler) Template() string {
	return "messages/details.html"
}

func (h *DetailsHandler) ActiveMenuItem() components.MenuItem {
	return components.MessagesMenuItem
}

func (h *DetailsHandler) View(ctx *gin.Context) (string, interface{}, error) {
	var view detailsView

	name := strings.TrimPrefix(ctx.Param("name"), "/")
	var pkg string

	if n := strings.LastIndexByte(name, '.'); n != -1 {
		pkg = name[:n]
		name = name[n+1:]
	}

	if err := h.loadDetails(ctx, &view, pkg, name); err != nil {
		if err == sql.ErrNoRows {
			ctx.AbortWithStatus(http.StatusNotFound)
			return "", nil, nil
		}

		return "", nil, err
	}

	if err := h.loadApplications(ctx, &view, pkg, name); err != nil {
		return "", nil, err
	}

	if err := h.loadHandlers(ctx, &view, pkg, name); err != nil {
		return "", nil, err
	}

	return view.Impl.Name, view, nil
}

func (h *DetailsHandler) loadDetails(
	ctx context.Context,
	view *detailsView,
	pkg, name string,
) error {
	row := h.DB.QueryRowContext(
		ctx,
		`SELECT
			t.package,
			t.name,
			COALESCE(t.url, ''),
			COALESCE(t.docs, ''),
			MODE() WITHIN GROUP (ORDER BY m.role) AS role,
			COUNT(DISTINCT m.role) > 1 AS has_role_mismatch,
			COUNT(DISTINCT m.is_pointer) > 1 AS has_pointer_mismatch
		FROM dogmabrowser.handler_message AS m
		INNER JOIN dogmabrowser.type AS t
		ON t.id = m.type_id
		WHERE t.package = $1
		AND t.name = $2
		GROUP BY t.id`,
		pkg,
		name,
	)

	return row.Scan(
		&view.Impl.Package,
		&view.Impl.Name,
		&view.Impl.URL,
		&view.Impl.Docs,
		&view.Role,
		&view.HasRoleMismatch,
		&view.HasPointerMismatch,
	)
}

func (h *DetailsHandler) loadApplications(
	ctx context.Context,
	view *detailsView,
	pkg, name string,
) error {
	rows, err := h.DB.QueryContext(
		ctx,
		`SELECT DISTINCT ON (a.name, a.key)
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
		INNER JOIN dogmabrowser.handler AS h
		ON h.application_key = a.key
		INNER JOIN dogmabrowser.handler_message AS m
		ON m.handler_key = h.key
		INNER JOIN dogmabrowser.type AS mt
		ON mt.id = m.type_id
		WHERE mt.package = $1
		AND mt.name = $2
		ORDER BY a.name`,
		pkg,
		name,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var s applicationSummary

		if err := rows.Scan(
			&s.Key,
			&s.Name,
			&s.Impl.Package,
			&s.Impl.Name,
			&s.Impl.IsPointer,
			&s.Impl.URL,
			&s.Impl.Docs,
		); err != nil {
			return err
		}

		view.Applications = append(view.Applications, s)
	}

	return rows.Err()
}

func (h *DetailsHandler) loadHandlers(
	ctx context.Context,
	view *detailsView,
	pkg, name string,
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
			m.is_produced,
			m.is_consumed
		FROM dogmabrowser.handler AS h
		INNER JOIN dogmabrowser.type AS t
		ON t.id = h.type_id
		INNER JOIN dogmabrowser.application AS a
		ON a.key = h.application_key
		INNER JOIN dogmabrowser.handler_message AS m
		ON m.handler_key = h.key
		INNER JOIN dogmabrowser.type AS mt
		ON mt.id = m.type_id
		WHERE mt.package = $1
		AND mt.name = $2
		ORDER BY h.name, a.name`,
		pkg,
		name,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var s handlerSummary

		var isProduced, isConsumed bool

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
			&isProduced,
			&isConsumed,
		); err != nil {
			return err
		}

		if isProduced {
			view.Producers = append(view.Producers, s)
		}

		if isConsumed {
			view.Consumers = append(view.Consumers, s)
		}
	}

	return rows.Err()
}
