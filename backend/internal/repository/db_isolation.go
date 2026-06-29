package repository

import (
	"context"
	"database/sql/driver"
	"fmt"
)

const setReadCommittedStmt = "SET default_transaction_isolation = 'read committed'"

// cockroachReadCommittedConnector wraps a base driver.Connector and runs
// `SET default_transaction_isolation = 'read committed'` on every new connection.
//
// This makes the whole pool use READ COMMITTED on CockroachDB, matching PostgreSQL's
// default isolation (which this application was written for, with no client-side 40001
// serialization-retry logic). CockroachDB defaults to SERIALIZABLE and aborts contended
// transactions with 40001, which would surface as request failures on hot write paths
// (usage_logs, quota, scheduler outbox).
//
// A per-connection SET is required because ALTER DATABASE/ROLE SET does not propagate
// this GUC to new sessions on CockroachDB.
type cockroachReadCommittedConnector struct {
	base driver.Connector
}

func (c cockroachReadCommittedConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	execer, ok := conn.(driver.ExecerContext)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("driver conn does not implement ExecerContext; cannot set read committed isolation")
	}
	if _, err := execer.ExecContext(ctx, setReadCommittedStmt, nil); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("set read committed isolation on new connection: %w", err)
	}
	return conn, nil
}

func (c cockroachReadCommittedConnector) Driver() driver.Driver { return c.base.Driver() }
