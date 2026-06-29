// Command migrate applies the embedded SQL migrations to a database and exits.
//
// It boots none of the application (no Redis, no secrets) — it only opens a DB
// connection and runs repository.ApplyMigrations, which auto-detects PostgreSQL vs
// CockroachDB via SELECT version() and picks the matching lock + overlay path.
//
// Usage:
//
//	go run ./cmd/migrate                          # uses the default DSN below
//	go run ./cmd/migrate -dsn "host=... dbname=..." # explicit lib/pq DSN
//
// The default DSN targets a local insecure CockroachDB node.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/repository"
	_ "github.com/lib/pq"
)

func main() {
	dsn := flag.String("dsn", "host=localhost port=26257 user=root dbname=sub2api sslmode=disable", "lib/pq connection string")
	flag.Parse()

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		fatal("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		fatal("ping db (is the DSN/database correct and reachable?): %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	start := time.Now()
	fmt.Println("applying migrations...")
	if err := repository.ApplyMigrations(ctx, db); err != nil {
		fatal("apply migrations: %v", err)
	}

	var n int
	_ = db.QueryRowContext(ctx, "SELECT count(*) FROM schema_migrations").Scan(&n)
	fmt.Printf("OK: migrations applied in %s; schema_migrations rows=%d\n", time.Since(start).Round(time.Millisecond), n)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "migrate: "+format+"\n", args...)
	os.Exit(1)
}
