package messages

import (
	"context"
	"log/slog"
)

// GoModuleFound is a message that indicates a Go module was found at a specific
// version.
type GoModuleFound struct {
	RepoSource    string
	RepoID        string
	ModulePath    string
	ModuleVersion string
}

// LogTo logs the message to the given logger.
func (m GoModuleFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"go module found",
		slog.String("repo_source", m.RepoSource),
		slog.String("repo_id", m.RepoID),
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
	)
}
