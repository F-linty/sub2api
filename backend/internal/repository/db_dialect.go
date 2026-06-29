package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// Dialect 标识底层数据库引擎方言。
// PostgreSQL 与 CockroachDB 共用 lib/pq 驱动及 ent 的 dialect.Postgres SQL 生成，
// 仅在迁移锁、部分 DDL（CONCURRENTLY/BRIN/分区/advisory lock）上需要分支处理。
type Dialect string

const (
	DialectPostgres  Dialect = "postgres"
	DialectCockroach Dialect = "cockroach"
)

// IsCockroach 报告该方言是否为 CockroachDB。
func (d Dialect) IsCockroach() bool { return d == DialectCockroach }

// detectDialect 通过 SELECT version() 探测真实引擎。
// CockroachDB 的 version() 字符串以 "CockroachDB" 开头，借此与原生 PostgreSQL 区分。
// 探测失败（如权限/网络）时返回空 Dialect，由调用方回退到配置声明值。
func detectDialect(ctx context.Context, db *sql.DB) Dialect {
	if db == nil {
		return ""
	}
	var version string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		return ""
	}
	if strings.Contains(strings.ToLower(version), "cockroach") {
		return DialectCockroach
	}
	return DialectPostgres
}

// resolveDialect 结合配置声明与运行时探测，确定最终使用的方言。
//
// 配置是权威来源（决定走哪条迁移/锁路径），探测仅用于纠正明显的误配置：
// 若二者不一致，以探测到的真实引擎为准并记录告警，避免对 CockroachDB 误用
// PostgreSQL 专属语法（CONCURRENTLY/BRIN/advisory lock）而在迁移期硬失败。
func resolveDialect(ctx context.Context, db *sql.DB, cfg *config.Config) Dialect {
	declared := DialectPostgres
	if cfg != nil && cfg.Database.IsCockroach() {
		declared = DialectCockroach
	}

	detected := detectDialect(ctx, db)
	if detected == "" || detected == declared {
		return declared
	}
	return detected
}
