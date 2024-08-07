package askpass

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
)

// Handler is an [http.Handler] that publishes "askpass" requests to the message bus.
type Handler struct {
	init    sync.Once
	outbox  chan<- any
	ready   chan struct{}
	pending sync.Map // correlation ID -> response channel
}

// Run pipes events received by the webhook handler to the message bus.
func (h *Handler) Run(ctx context.Context) (err error) {
	h.init.Do(func() {
		h.ready = make(chan struct{})
	})

	minibus.Subscribe[messages.RepoCredentialsResponse](ctx)
	minibus.Ready(ctx)

	h.outbox = minibus.Outbox(ctx)
	close(h.ready)

	for m := range minibus.Inbox(ctx) {
		res := m.(messages.RepoCredentialsResponse)
		if reply, ok := h.pending.Load(res.CorrelationID); ok {
			reply.(chan messages.RepoCredentialsResponse) <- res
		}
	}

	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if host, _, _ := net.SplitHostPort(r.RemoteAddr); host != "127.0.0.1" {
		http.Error(
			w,
			http.StatusText(http.StatusForbidden),
			http.StatusForbidden,
		)
		return
	}

	h.init.Do(func() {
		h.ready = make(chan struct{})
	})

	select {
	case <-ctx.Done():
		writeContextError(ctx, w)
		return
	case <-h.ready:
	}

	var req messages.RepoCredentialsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	reply := make(chan messages.RepoCredentialsResponse, 1)
	h.pending.Store(req.CorrelationID, reply)
	defer h.pending.Delete(req.CorrelationID)

	select {
	case <-ctx.Done():
		writeContextError(ctx, w)
		return
	case h.outbox <- req:
	}

	select {
	case <-ctx.Done():
		writeContextError(ctx, w)
		return

	case res := <-reply:
		data, err := json.Marshal(res)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

func writeContextError(ctx context.Context, w http.ResponseWriter) {
	code := http.StatusInternalServerError
	if ctx.Err() == context.DeadlineExceeded {
		code = http.StatusRequestTimeout
	}

	http.Error(
		w,
		ctx.Err().Error(),
		code,
	)
}
