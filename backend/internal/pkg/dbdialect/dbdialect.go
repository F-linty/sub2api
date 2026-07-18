package dbdialect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	TypeAuto        = "auto"
	TypePostgres    = "postgres"
	TypeCockroachDB = "cockroachdb"
)

// Dialect centralizes SQL differences between PostgreSQL and CockroachDB.
// Ent still uses the PostgreSQL wire dialect for both databases; this adapter
// is for raw SQL and database-specific capabilities.
type Dialect interface {
	Name() string
	SupportsAdvisoryLocks() bool
	UsesPhysicalRowID() bool
	SupportsMaterializedCTE() bool
	SupportsPartialConflictTarget() bool
	SupportsPostgresPartitionCatalog() bool
	SupportsInvalidIndexCatalog() bool
}

type Postgres struct{}

func (Postgres) Name() string                { return TypePostgres }
func (Postgres) SupportsAdvisoryLocks() bool { return true }
func (Postgres) UsesPhysicalRowID() bool     { return true }
func (Postgres) SupportsMaterializedCTE() bool {
	return true
}
func (Postgres) SupportsPartialConflictTarget() bool {
	return true
}
func (Postgres) SupportsPostgresPartitionCatalog() bool {
	return true
}
func (Postgres) SupportsInvalidIndexCatalog() bool {
	return true
}

type CockroachDB struct{}

func (CockroachDB) Name() string                { return TypeCockroachDB }
func (CockroachDB) SupportsAdvisoryLocks() bool { return false }
func (CockroachDB) UsesPhysicalRowID() bool     { return false }
func (CockroachDB) SupportsMaterializedCTE() bool {
	return false
}
func (CockroachDB) SupportsPartialConflictTarget() bool {
	return false
}
func (CockroachDB) SupportsPostgresPartitionCatalog() bool {
	return false
}
func (CockroachDB) SupportsInvalidIndexCatalog() bool {
	return false
}

type holder struct {
	d Dialect
}

var current atomic.Value

func init() {
	current.Store(holder{d: Postgres{}})
}

func Current() Dialect {
	if h, ok := current.Load().(holder); ok && h.d != nil {
		return h.d
	}
	return Postgres{}
}

func Set(d Dialect) {
	if d == nil {
		d = Postgres{}
	}
	current.Store(holder{d: d})
}

func Resolve(ctx context.Context, db *sql.DB, cfg *config.DatabaseConfig) (Dialect, error) {
	requested := TypeAuto
	if cfg != nil && strings.TrimSpace(cfg.Type) != "" {
		requested = strings.ToLower(strings.TrimSpace(cfg.Type))
	}

	switch requested {
	case TypePostgres, "postgresql", "pg":
		return Postgres{}, nil
	case TypeCockroachDB, "cockroach", "crdb":
		return CockroachDB{}, nil
	case TypeAuto:
		return detect(ctx, db)
	default:
		return nil, fmt.Errorf("unsupported database.type %q (expected auto, postgres, or cockroachdb)", requested)
	}
}

func detect(ctx context.Context, db *sql.DB) (Dialect, error) {
	if db == nil {
		return Postgres{}, nil
	}

	var version string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		return nil, fmt.Errorf("detect database dialect: %w", err)
	}
	if strings.Contains(strings.ToLower(version), "cockroach") {
		return CockroachDB{}, nil
	}
	return Postgres{}, nil
}
