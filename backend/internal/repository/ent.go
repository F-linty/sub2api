// Package repository 提供应用程序的基础设施层组件。
// 包括数据库连接初始化、ORM 客户端管理、Redis 连接、数据库迁移等核心功能。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/dbdialect"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/migrations"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/lib/pq" // PostgreSQL 驱动（注册驱动 + pq.NewConnector）
)

// InitEnt 初始化 Ent ORM 客户端并返回客户端实例和底层的 *sql.DB。
//
// 该函数执行以下操作：
//  1. 初始化全局时区设置，确保时间处理一致性
//  2. 建立 PostgreSQL 数据库连接
//  3. 自动执行数据库迁移，确保 schema 与代码同步
//  4. 创建并返回 Ent 客户端实例
//
// 重要提示：调用者必须负责关闭返回的 ent.Client（关闭时会自动关闭底层的 driver/db）。
//
// 参数：
//   - cfg: 应用程序配置，包含数据库连接信息和时区设置
//
// 返回：
//   - *ent.Client: Ent ORM 客户端，用于执行数据库操作
//   - *sql.DB: 底层的 SQL 数据库连接，可用于直接执行原生 SQL
//   - error: 初始化过程中的错误
func InitEnt(cfg *config.Config) (*ent.Client, *sql.DB, error) {
	// 优先初始化时区设置，确保所有时间操作使用统一的时区。
	// 这对于跨时区部署和日志时间戳的一致性至关重要。
	if err := timezone.Init(cfg.Timezone); err != nil {
		return nil, nil, err
	}

	// 构建包含时区信息的数据库连接字符串 (DSN)。
	// 时区信息会传递给 PostgreSQL，确保数据库层面的时间处理正确。
	dsn := cfg.Database.DSNWithTimezone(cfg.Timezone)

	// 使用 Ent 的 SQL 驱动打开数据库连接（PostgreSQL/CockroachDB 共用 dialect.Postgres）。
	//
	// CockroachDB 且启用 read-committed 时，包一层 connector，在每个新连接上执行
	// SET default_transaction_isolation = 'read committed'，使整个连接池与 PostgreSQL
	// 默认隔离级别一致（本项目无 40001 序列化重试；CRDB 默认 SERIALIZABLE 会在高并发
	// 写路径上以 40001 中止事务）。ALTER DATABASE/ROLE SET 在 CRDB 上不会传播该 GUC，
	// 故必须按连接 SET。
	var drv *entsql.Driver
	if cfg.Database.IsCockroach() && cfg.Database.CockroachReadCommitted {
		base, connErr := pq.NewConnector(dsn)
		if connErr != nil {
			return nil, nil, connErr
		}
		sqlDB := sql.OpenDB(cockroachReadCommittedConnector{base: base})
		drv = entsql.OpenDB(dialect.Postgres, sqlDB)
		slog.Info("CockroachDB connections will use read committed isolation")
	} else {
		opened, openErr := entsql.Open(dialect.Postgres, dsn)
		if openErr != nil {
			return nil, nil, openErr
		}
		drv = opened
	}
	applyDBPoolSettings(drv.DB(), cfg)

	// 解析数据库方言（postgres / cockroach）。配置为权威来源，
	// 运行时 SELECT version() 仅用于纠正误配置并记录告警。
	detectCtx, detectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	dbDialect := resolveDialect(detectCtx, drv.DB(), cfg)
	detectCancel()
	if cfg.Database.IsCockroach() != dbDialect.IsCockroach() {
		slog.Warn("database dialect mismatch: configuration overridden by runtime detection",
			"configured", cfg.Database.DriverName(),
			"detected", string(dbDialect),
		)
	}
	// 发布到进程级方言标志，供 repository/service 各层的低级锁助手分支使用。
	dbdialect.SetCockroach(dbDialect.IsCockroach())
	slog.Info("database dialect resolved", "dialect", string(dbDialect))

	// 确保数据库 schema 已准备就绪。
	// SQL 迁移文件是 schema 的权威来源（source of truth）。
	// 这种方式比 Ent 的自动迁移更可控，支持复杂的迁移场景。
	migrationCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := applyMigrationsFS(migrationCtx, drv.DB(), migrations.FS, dbDialect); err != nil {
		_ = drv.Close() // 迁移失败时关闭驱动，避免资源泄露
		return nil, nil, err
	}

	// 创建 Ent 客户端，绑定到已配置的数据库驱动。
	client := ent.NewClient(ent.Driver(drv))

	// 启动阶段：从配置或数据库中确保系统密钥可用。
	if err := ensureBootstrapSecrets(migrationCtx, client, cfg); err != nil {
		_ = client.Close()
		return nil, nil, err
	}

	// 在密钥补齐后执行完整配置校验，避免空 jwt.secret 导致服务运行时失败。
	if err := cfg.Validate(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("validate config after secret bootstrap: %w", err)
	}

	// SIMPLE 模式：启动时补齐各平台默认分组。
	// - anthropic/openai/gemini: 确保存在 <platform>-default
	// - antigravity: 仅要求存在 >=2 个未软删除分组（用于 claude/gemini 混合调度场景）
	if cfg.RunMode == config.RunModeSimple {
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer seedCancel()
		if err := ensureSimpleModeDefaultGroups(seedCtx, client); err != nil {
			_ = client.Close()
			return nil, nil, err
		}
		if err := ensureSimpleModeAdminConcurrency(seedCtx, client); err != nil {
			_ = client.Close()
			return nil, nil, err
		}
	}

	return client, drv.DB(), nil
}
