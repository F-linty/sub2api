package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/dbdialect"
)

const (
	DatabaseTypeAuto        = dbdialect.TypeAuto
	DatabaseTypePostgres    = dbdialect.TypePostgres
	DatabaseTypeCockroachDB = dbdialect.TypeCockroachDB
)

type DatabaseDialect = dbdialect.Dialect

func CurrentDatabaseDialect() DatabaseDialect {
	return dbdialect.Current()
}

func SetCurrentDatabaseDialect(d DatabaseDialect) {
	dbdialect.Set(d)
}

func ResolveDatabaseDialect(ctx context.Context, db *sql.DB, cfg *config.DatabaseConfig) (DatabaseDialect, error) {
	return dbdialect.Resolve(ctx, db, cfg)
}
