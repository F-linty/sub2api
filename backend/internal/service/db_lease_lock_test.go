package service

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestTryAcquireDBLeaseLock_AcquiredAndReleased(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS runtime_locks").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO runtime_locks").
		WillReturnResult(sqlmock.NewResult(0, 1)) // acquired: 1 row affected
	mock.ExpectExec("DELETE FROM runtime_locks").
		WillReturnResult(sqlmock.NewResult(0, 1))

	release, ok := TryAcquireDBLeaseLock(context.Background(), db, "job-x")
	require.True(t, ok)
	require.NotNil(t, release)
	release()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTryAcquireDBLeaseLock_HeldByOther(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS runtime_locks").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO runtime_locks").
		WillReturnResult(sqlmock.NewResult(0, 0)) // held elsewhere: 0 rows affected

	release, ok := TryAcquireDBLeaseLock(context.Background(), db, "job-x")
	require.False(t, ok)
	require.Nil(t, release)

	require.NoError(t, mock.ExpectationsWereMet())
}
