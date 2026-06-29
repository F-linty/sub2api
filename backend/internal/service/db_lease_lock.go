package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

// runtimeLockTableDDL defines the keyed lease table used as a CockroachDB substitute
// for session-scoped pg_try_advisory_lock (which CRDB exposes only as a no-op stub).
const runtimeLockTableDDL = `
CREATE TABLE IF NOT EXISTS runtime_locks (
	lock_key   TEXT PRIMARY KEY,
	owner      TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL
);`

const (
	// runtimeLockTTL bounds how long a crashed holder can block others.
	runtimeLockTTL = 30 * time.Second
	// runtimeLockHeartbeat renews the lease while the work runs; must be < TTL.
	runtimeLockHeartbeat = 10 * time.Second
)

func runtimeLockOwner() string {
	host, _ := os.Hostname()
	var nonce [8]byte
	_, _ = rand.Read(nonce[:])
	return fmt.Sprintf("%s/%d/%s", host, os.Getpid(), hex.EncodeToString(nonce[:]))
}

// TryAcquireDBLeaseLock is a non-blocking, best-effort cross-instance lock for key on
// CockroachDB, backed by a TTL lease row with heartbeat renewal. It returns
// (release, true) when acquired, or (nil, false) when another live owner holds it.
//
// This is the CockroachDB counterpart to PostgreSQL's session advisory lock: callers
// branch on dbdialect.IsCockroach() and use this instead.
func TryAcquireDBLeaseLock(ctx context.Context, db *sql.DB, key string) (func(), bool) {
	if db == nil {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := db.ExecContext(ctx, runtimeLockTableDDL); err != nil {
		return nil, false
	}

	owner := runtimeLockOwner()
	ttl := fmt.Sprintf("%d seconds", int(runtimeLockTTL.Seconds()))

	// Acquire iff the key is free, expired, or already ours (reentrant heartbeat).
	res, err := db.ExecContext(ctx,
		`INSERT INTO runtime_locks (lock_key, owner, expires_at)
		 VALUES ($1, $2, now() + $3::interval)
		 ON CONFLICT (lock_key) DO UPDATE
		   SET owner = excluded.owner, expires_at = excluded.expires_at
		   WHERE runtime_locks.owner = excluded.owner OR runtime_locks.expires_at < now()`,
		key, owner, ttl)
	if err != nil {
		return nil, false
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, false
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(runtimeLockHeartbeat)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				hbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, _ = db.ExecContext(hbCtx,
					`UPDATE runtime_locks SET expires_at = now() + $2::interval
					 WHERE lock_key = $1 AND owner = $3`, key, ttl, owner)
				cancel()
			}
		}
	}()

	release := func() {
		close(stop)
		<-done
		rc, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(rc, `DELETE FROM runtime_locks WHERE lock_key = $1 AND owner = $2`, key, owner)
	}
	return release, true
}
