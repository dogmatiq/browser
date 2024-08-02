package messages

import (
	"context"
	"log/slog"
)

// RepoFound is a message that indicates a repository was found.
type RepoFound struct {
	RepoSource string
	RepoName   string
}

// LogTo logs the message to the given logger.
func (m RepoFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.InfoContext(
		ctx,
		"repository found",
		slog.String("repo_source", m.RepoSource),
		slog.String("repo_name", m.RepoName),
	)
}

// RepoLost is a message that indicates a repository is lost, either because it
// has been deleted or is no longer accessible to the browser.
type RepoLost struct {
	RepoSource string
	RepoName   string
}

// LogTo logs the message to the given logger.
func (m RepoLost) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.InfoContext(
		ctx,
		"repository lost",
		slog.String("repo_source", m.RepoSource),
		slog.String("repo_name", m.RepoName),
	)
}

// GoModuleFound is a message that indicates a Go module has been found.
type GoModuleFound struct {
	ModulePath    string
	ModuleVersion string
}

// LogTo logs the message to the given logger.
func (m GoModuleFound) LogTo(ctx context.Context, logger *slog.Logger) {
	logger.InfoContext(
		ctx,
		"go module found",
		slog.String("module_path", m.ModulePath),
		slog.String("module_version", m.ModuleVersion),
	)
}
