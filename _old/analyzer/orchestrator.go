package analyzer

import (
	"context"
	"sync"
)

// Orchestrator orchestrates the analysis and removal of repositories.
type Orchestrator struct {
	Analyzer *Analyzer
	Remover  *Remover

	once  sync.Once
	queue chan queueItem
}

type queueItem struct {
	repoID int64
	remove bool
}

// Run performs analysis and removal until ctx is cancelled or an error occurs.
func (o *Orchestrator) Run(ctx context.Context) error {
	o.init()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item := <-o.queue:
			if err := o.handle(ctx, item); err != nil {
				return err
			}
		}
	}
}

// EnqueueAnalyis enqueues a repository for analysis.
func (o *Orchestrator) EnqueueAnalyis(ctx context.Context, repoID int64) error {
	return o.enqueue(ctx, repoID, false)
}

// EnqueueRemoval enqueues a repository for removal.
func (o *Orchestrator) EnqueueRemoval(ctx context.Context, repoID int64) error {
	return o.enqueue(ctx, repoID, true)
}

func (o *Orchestrator) handle(ctx context.Context, item queueItem) error {
	if item.remove {
		return o.Remover.Remove(ctx, item.repoID)
	}

	return o.Analyzer.Analyze(ctx, item.repoID)
}

func (o *Orchestrator) enqueue(ctx context.Context, repoID int64, remove bool) error {
	o.init()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case o.queue <- queueItem{repoID, remove}:
		return nil
	}
}

func (o *Orchestrator) init() {
	o.once.Do(func() {
		o.queue = make(chan queueItem, 100)
	})
}
