package github

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
)

// WebHookHandler is an [http.Handler] that publishes GitHub webhook events to
// the message bus.
type WebHookHandler struct {
	Secret string
	Logger *slog.Logger

	init   sync.Once
	events chan any
}

// Run pipes events received by the webhook handler to the message bus.
func (h *WebHookHandler) Run(ctx context.Context) (err error) {
	h.init.Do(func() {
		h.events = make(chan any)
	})

	minibus.Ready(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-h.events:
			if err := minibus.Send(ctx, event); err != nil {
				return err
			}
		}
	}
}

func (h *WebHookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.init.Do(func() {
		h.events = make(chan any)
	})

	ctx := r.Context()

	payload, err := github.ValidatePayload(r, []byte(h.Secret))
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusForbidden),
			http.StatusForbidden,
		)

		h.Logger.DebugContext(
			ctx,
			"unable to validate GitHub webhook payload",
			slog.String("error", err.Error()),
		)

		return
	}

	eventType := github.WebHookType(r)
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)

		h.Logger.DebugContext(
			ctx,
			"unable to parse GitHub webhook payload",
			slog.Group(
				"event",
				slog.String("type", eventType),
			),
			slog.String("error", err.Error()),
		)

		return
	}

	select {
	case <-ctx.Done():
		http.Error(
			w,
			http.StatusText(http.StatusServiceUnavailable),
			http.StatusServiceUnavailable,
		)
	case h.events <- event:
		w.WriteHeader(http.StatusNoContent)
	}
}
