package analyzer

import (
	"context"
	"database/sql"

	"github.com/dogmatiq/browser/persistence"
	"github.com/dogmatiq/dodeca/logging"
)

// Remover removes information about repositories from the database.
type Remover struct {
	DB     *sql.DB
	Logger logging.Logger
}

// Remove removes all analysis results from the given repository.
func (rm *Remover) Remove(ctx context.Context, repoID int64) error {
	logging.Log(rm.Logger, "[#%d] removing repository", repoID)

	tx, err := rm.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint:errcheck

	if err := persistence.RemoveRepository(ctx, tx, repoID); err != nil {
		return err
	}

	return tx.Commit()
}
