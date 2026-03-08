// Package cluster provides leader election for multi-instance Rampart deployments.
//
// It uses PostgreSQL advisory locks to ensure only one instance runs
// background workers (cleanup, webhook retries) at a time. If the leader
// crashes, the lock is automatically released and another instance acquires it.
package cluster

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// advisoryLockID is the application-level lock key for leader election.
// This is an arbitrary constant — all Rampart instances must use the same value.
const advisoryLockID int64 = 0x52414D50415254 // "RAMPART" in hex

// Leader manages leader election using PostgreSQL advisory locks.
// Only the leader instance runs background workers to prevent duplicate work.
type Leader struct {
	pool     *pgxpool.Pool
	logger   *slog.Logger
	isLeader atomic.Bool
	done     chan struct{}
}

// NewLeader creates a new leader elector.
func NewLeader(pool *pgxpool.Pool, logger *slog.Logger) *Leader {
	return &Leader{
		pool:   pool,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// IsLeader returns true if this instance currently holds the leader lock.
func (l *Leader) IsLeader() bool {
	return l.isLeader.Load()
}

// Run starts the leader election loop. It periodically tries to acquire the
// advisory lock. When acquired, background workers should run. When lost,
// they should stop. This function blocks until ctx is cancelled.
func (l *Leader) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	defer close(l.done)

	for {
		select {
		case <-ctx.Done():
			l.release(context.Background())
			return
		case <-ticker.C:
			l.tryAcquire(ctx)
		}
	}
}

// tryAcquire attempts to acquire the advisory lock.
// pg_try_advisory_lock is non-blocking — returns true if lock acquired, false otherwise.
func (l *Leader) tryAcquire(ctx context.Context) {
	var acquired bool
	err := l.pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockID).Scan(&acquired)
	if err != nil {
		l.logger.Warn("leader election: failed to try advisory lock", "error", err)
		l.isLeader.Store(false)
		return
	}

	wasLeader := l.isLeader.Load()
	l.isLeader.Store(acquired)

	if acquired && !wasLeader {
		l.logger.Info("leader election: this instance is now the leader")
	} else if !acquired && wasLeader {
		l.logger.Info("leader election: this instance lost leadership")
	}
}

// release explicitly releases the advisory lock on shutdown.
func (l *Leader) release(ctx context.Context) {
	if !l.isLeader.Load() {
		return
	}
	_, err := l.pool.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
	if err != nil {
		l.logger.Warn("leader election: failed to release advisory lock", "error", err)
	} else {
		l.logger.Info("leader election: released leadership")
	}
	l.isLeader.Store(false)
}

// Wait blocks until the leader election loop exits.
func (l *Leader) Wait() {
	<-l.done
}
