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

// RepoCredentialsRequest is a message that requests credentials for a
// repository located within the given URL. The URL might not contain the
// complete path to the repository.
type RepoCredentialsRequest struct {
	CorrelationID uuid.UUID
	RepoURL       string
}

// LogTo logs the message to the given logger.
func (m RepoCredentialsRequest) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"requesting credentials for repository",
		slog.String("correlation_id", m.CorrelationID.String()),
		slog.String("repo_url", m.RepoURL),
	)
}

// RepoCredentialsResponse is a response to a [RepoCredentialsRequest] message.
type RepoCredentialsResponse struct {
	CorrelationID uuid.UUID
	Username      string
	Password      string
}

// LogTo logs the message to the given logger.
func (m RepoCredentialsResponse) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"responding with credentials for repository",
		slog.String("correlation_id", m.CorrelationID.String()),
		slog.String("repo_username", m.Username),
	)
}
