package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestResolveDatabaseDialectFromConfig(t *testing.T) {
	t.Parallel()

	dialect, err := ResolveDatabaseDialect(context.Background(), nil, &config.DatabaseConfig{Type: "cockroachdb"})
	require.NoError(t, err)
	require.Equal(t, DatabaseTypeCockroachDB, dialect.Name())
	require.False(t, dialect.SupportsAdvisoryLocks())
	require.False(t, dialect.UsesPhysicalRowID())
	require.False(t, dialect.SupportsMaterializedCTE())
	require.False(t, dialect.SupportsPartialConflictTarget())
	require.False(t, dialect.SupportsPostgresPartitionCatalog())
	require.False(t, dialect.SupportsInvalidIndexCatalog())

	dialect, err = ResolveDatabaseDialect(context.Background(), nil, &config.DatabaseConfig{Type: "postgres"})
	require.NoError(t, err)
	require.Equal(t, DatabaseTypePostgres, dialect.Name())
	require.True(t, dialect.SupportsAdvisoryLocks())
	require.True(t, dialect.UsesPhysicalRowID())
	require.True(t, dialect.SupportsMaterializedCTE())
	require.True(t, dialect.SupportsPartialConflictTarget())
	require.True(t, dialect.SupportsPostgresPartitionCatalog())
	require.True(t, dialect.SupportsInvalidIndexCatalog())
}

func TestResolveDatabaseDialectAutoDetectsCockroachDB(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT version\\(\\)").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("CockroachDB CCL v25.2.0"))

	dialect, err := ResolveDatabaseDialect(context.Background(), db, &config.DatabaseConfig{Type: "auto"})
	require.NoError(t, err)
	require.Equal(t, DatabaseTypeCockroachDB, dialect.Name())
	require.NoError(t, mock.ExpectationsWereMet())
}
