package messages

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// GoModuleFound is a message that indicates a Go module was found at a specific
// version.
type GoModuleFound struct {
	RepoID        uuid.UUID
	ModulePath    string
	ModuleVersion string
}

// LogTo logs the message to the given logger.
func (m GoModuleFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"go module found",
		slog.String("repo_id", m.RepoID.String()),
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
	)
}
