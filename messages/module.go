package messages

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// RepoFound is a message that indicates a repository was found.
type RepoFound struct {
	RepoID     uuid.UUID
	RepoSource string
	RepoName   string
}

// LogTo logs the message to the given logger.
func (m RepoFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.InfoContext(
		ctx,
		"repository found",
		slog.String("repo_id", m.RepoID.String()),
		slog.String("repo_source", m.RepoSource),
		slog.String("repo_name", m.RepoName),
	)
}

// RepoLost is a message that indicates a repository is lost, either because it
// has been deleted or is no longer accessible to the browser.
type RepoLost struct {
	RepoID uuid.UUID
}

// LogTo logs the message to the given logger.
func (m RepoLost) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.InfoContext(
		ctx,
		"repository lost",
		slog.String("repo_id", m.RepoID.String()),
	)
}

// GoModuleFound is a message that indicates a Go module has been found.
type GoModuleFound struct {
	RepoID        uuid.UUID
	ModulePath    string
	ModuleVersion string
}

// LogTo logs the message to the given logger.
func (m GoModuleFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.InfoContext(
		ctx,
		"go module found",
		slog.String("repo_id", m.RepoID.String()),
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
	)
}
