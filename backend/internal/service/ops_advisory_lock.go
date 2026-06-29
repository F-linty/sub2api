package service

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/dbdialect"
)

func hashAdvisoryLockID(key string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return int64(h.Sum64())
}

func tryAcquireDBAdvisoryLock(ctx context.Context, db *sql.DB, lockID int64) (func(), bool) {
	if db == nil {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// CockroachDB exposes pg_try_advisory_lock only as a no-op stub (no real mutual
	// exclusion), so use the keyed TTL lease table instead for genuine cross-instance gating.
	if dbdialect.IsCockroach() {
		return TryAcquireDBLeaseLock(ctx, db, fmt.Sprintf("advisory:%d", lockID))
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, false
	}

	acquired := false
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false
	}
	if !acquired {
		_ = conn.Close()
		return nil, false
	}

	release := func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", lockID)
		_ = conn.Close()
	}
	return release, true
}
