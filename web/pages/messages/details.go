package messages

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/docserve/web/components"
	"github.com/gin-gonic/gin"
)

type detailsView struct {
	Impl  components.Type
	Roles string // comma separated

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

	return view.Impl.Name + " - Message Details", view, nil
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
			t.url,
			string_agg(DISTINCT m.role, ', ' ORDER BY m.role)
		FROM docserve.handler_message AS m
		INNER JOIN docserve.type AS t
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
		&view.Roles,
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
			t.url
		FROM docserve.application AS a
		INNER JOIN docserve.type AS t
		ON t.id = a.type_id
		INNER JOIN docserve.handler AS h
		ON h.application_key = a.key
		INNER JOIN docserve.handler_message AS m
		ON m.handler_key = h.key
		INNER JOIN docserve.type AS mt
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
			t.url,
			a.key,
			a.name,
			m.is_produced,
			m.is_consumed
		FROM docserve.handler AS h
		INNER JOIN docserve.type AS t
		ON t.id = h.type_id
		INNER JOIN docserve.application AS a
		ON a.key = h.application_key
		INNER JOIN docserve.handler_message AS m
		ON m.handler_key = h.key
		INNER JOIN docserve.type AS mt
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