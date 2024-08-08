package messages

import (
	"context"
	"log/slog"
)

// RepoFound is a message that indicates a repository was found.
type RepoFound struct {
	Source string
	ID     string
	Name   string
}

// LogTo logs the message to the given logger.
func (m RepoFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"repository found",
		slog.String("repo_source", m.Source),
		slog.String("repo_id", m.ID),
		slog.String("repo_name", m.Name),
	)
}

// RepoLost is a message that indicates a repository is lost, either because it
// has been deleted or is no longer accessible to the browser.
type RepoLost struct {
	Source string
	ID     string
}

// LogTo logs the message to the given logger.
func (m RepoLost) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"repository lost",
		slog.String("repo_source", m.Source),
		slog.String("repo_id", m.ID),
	)
}
