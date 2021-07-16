package web

import (
	"fmt"
	"net/http"

	"github.com/dogmatiq/browser/githubx"
	"github.com/dogmatiq/browser/web/components"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v35/github"
)

type Handler interface {
	Route() (string, string)
	Template() string
	ActiveMenuItem() components.MenuItem
	View(ctx *gin.Context) (string, interface{}, error)
}

func NewRouter(
	version string,
	c *githubx.Connector,
	handlers ...Handler,
) http.Handler {
	router := gin.New()
	router.HTMLRender = pageTemplates

	router.Use(gin.Recovery())

	assets := http.FileServer(
		http.FS(assetsFS),
	)

	assetsHandler := func(ctx *gin.Context) {
		assets.ServeHTTP(ctx.Writer, ctx.Request)
	}

	router.GET("/assets/*path", assetsHandler)
	router.HEAD("/assets/*path", assetsHandler)

	router.GET(
		"/github/auth",
		handleOAuthCallback(version, c.OAuthConfig),
	)

	auth := requireOAuth(version, c)

	for _, h := range handlers {
		method, path := h.Route()

		router.Handle(
			method,
			path,
			auth,
			adaptHandler(version, h),
		)
	}

	router.NoRoute(
		func(ctx *gin.Context) {
			renderError(ctx, version, http.StatusNotFound)
		},
	)

	return router
}

type templateContext struct {
	Title          string
	ActiveMenuItem components.MenuItem
	User           *github.User
	Version        string
	View           interface{}
}

func adaptHandler(version string, h Handler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		title, view, err := h.View(ctx)
		if err != nil {
			fmt.Println("unable to generate view:", err) // TODO
			renderError(ctx, version, http.StatusInternalServerError)
			return
		}

		if ctx.IsAborted() {
			code := ctx.Writer.Status()
			if code == http.StatusOK {
				panic("handler aborted execution with OK status")
			}

			renderError(
				ctx,
				version,
				ctx.Writer.Status(),
			)

			return
		}

		u, _ := currentUser(ctx)

		ctx.HTML(
			http.StatusOK,
			h.Template(),
			templateContext{
				Title:          title,
				User:           u,
				ActiveMenuItem: h.ActiveMenuItem(),
				Version:        version,
				View:           view,
			},
		)
	}
}

func renderError(ctx *gin.Context, version string, code int) {
	view := struct {
		StatusText string
		StatusCode int
		Message    string
	}{
		StatusText: http.StatusText(code),
		StatusCode: code,
	}

	switch code {
	case http.StatusNotFound:
		view.Message = "The content you have requested can not be found."
	case http.StatusUnauthorized:
		view.Message = "You do not have permission to view this context."
	case http.StatusInternalServerError:
		view.Message = "An unexpected error occurred on the server."
	}

	u, _ := currentUser(ctx)

	renderer := pageTemplates.Instance(
		"error.html",
		templateContext{
			Title:   http.StatusText(code),
			User:    u,
			Version: version,
			View:    view,
		},
	)

	if ctx.Writer.Written() {
		renderer.Render(ctx.Writer) // nolint:errcheck
		return
	}

	ctx.Render(code, renderer)
}
