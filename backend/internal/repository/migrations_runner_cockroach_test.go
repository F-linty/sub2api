package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// TestAcquireMigrationLock_CockroachUsesLeaseTable 验证 CockroachDB 方言下迁移锁
// 走 schema_migration_lock 租约表，而非 session 级 advisory lock。
func TestAcquireMigrationLock_CockroachUsesLeaseTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migration_lock").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migration_lock").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE schema_migration_lock").
		WillReturnResult(sqlmock.NewResult(0, 1)) // 抢占成功：1 行受影响
	// 释放：清空锁归属。
	mock.ExpectExec("UPDATE schema_migration_lock").
		WillReturnResult(sqlmock.NewResult(0, 1))

	release, err := acquireMigrationLock(context.Background(), db, DialectCockroach)
	require.NoError(t, err)
	require.NotNil(t, release)
	release()

	require.NoError(t, mock.ExpectationsWereMet())
}

// TestAcquireMigrationLock_PostgresUsesAdvisoryLock 验证 PostgreSQL 方言下仍使用
// 原有的 advisory lock，行为保持不变。
func TestAcquireMigrationLock_PostgresUsesAdvisoryLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT pg_try_advisory_lock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
	mock.ExpectExec("SELECT pg_advisory_unlock\\(\\$1\\)").
		WithArgs(migrationsAdvisoryLockID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	release, err := acquireMigrationLock(context.Background(), db, DialectPostgres)
	require.NoError(t, err)
	require.NotNil(t, release)
	release()

	require.NoError(t, mock.ExpectationsWereMet())
}
