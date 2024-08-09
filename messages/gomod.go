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
		slog.Group(
			"repo",
			slog.String("source", m.RepoSource),
			slog.String("id", m.RepoID),
		),
		slog.Group(
			"module",
			slog.String("path", m.ModulePath),
			slog.String("version", m.ModuleVersion),
		),
	)
}

// GoModuleDownloaded is a message that indicates a Go module was downloaded
// into the module cache.
type GoModuleDownloaded struct {
	RepoSource    string
	RepoID        string
	ModulePath    string
	ModuleVersion string
	ModuleDir     string
}

// LogTo logs the message to the given logger.
func (m GoModuleDownloaded) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.DebugContext(
		ctx,
		"go module downloaded",
		slog.Group(
			"repo",
			slog.String("source", m.RepoSource),
			slog.String("id", m.RepoID),
		),
		slog.Group(
			"module",
			slog.String("path", m.ModulePath),
			slog.String("version", m.ModuleVersion),
		),
	)
}
