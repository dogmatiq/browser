package messages

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// RepoFound is a message that indicates a repository was found.
type RepoFound struct {
	ID     uuid.UUID
	Source string
	Name   string
}

// LogTo logs the message to the given logger.
func (m RepoFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"repository found",
		slog.String("repo_id", m.ID.String()),
		slog.String("repo_source", m.Source),
		slog.String("repo_name", m.Name),
	)
}

// RepoLost is a message that indicates a repository is lost, either because it
// has been deleted or is no longer accessible to the browser.
type RepoLost struct {
	ID uuid.UUID
}

// LogTo logs the message to the given logger.
func (m RepoLost) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"repository lost",
		slog.String("repo_id", m.ID.String()),
	)
}
