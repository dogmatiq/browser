package messagelogger

import (
	"context"
	"log/slog"
	"reflect"
	"strings"

	"github.com/dogmatiq/minibus"
)

// LogAware is an interface for a message that can log itself to a logger.
type LogAware interface {
	LogTo(context.Context, *slog.Logger)
}

// Run listens to all messages and logs them to the provided logger.
func Run(ctx context.Context, l *slog.Logger) (err error) {
	minibus.Subscribe[any](ctx)
	minibus.Ready(ctx)

	for m := range minibus.Inbox(ctx) {
		typ := reflect.TypeOf(m)

		impl := typ
		for impl.Kind() == reflect.Ptr {
			impl = impl.Elem()
		}

		// Assume that anything defined in this package _SHOULD_ inmplement
		// LogAware, even if it doesn't, and allow it panic.
		if strings.HasPrefix(impl.PkgPath(), "github.com/dogmatiq/browser/") {
			m.(LogAware).LogTo(ctx, l)
		} else {
			l.DebugContext(
				ctx,
				"third-party message",
				slog.String("message_type", typ.String()),
			)
		}
	}

	return ctx.Err()
}
