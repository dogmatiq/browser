package askpass

import (
	"context"
	"log/slog"
	"net/url"
)

// Request is a message that requests credentials for the repository located at
// the given URL. The URL might not contain the complete path to the repository.
type Request struct {
	CorrelationID uint64
	RepoURL       *url.URL
}

// LogTo logs the message to the given logger.
func (m Request) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"askpass request",
		slog.Uint64("correlation_id", m.CorrelationID),
		slog.String("repo_url", m.RepoURL.String()),
	)
}

// Response is a response to a [Request] message.
type Response struct {
	CorrelationID uint64
	Username      string
	Password      string
}

// LogTo logs the message to the given logger.
func (m Response) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"askpass response",
		slog.Uint64("correlation_id", m.CorrelationID),
		slog.String("repo_username", m.Username),
	)
}
