//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/pkg/dbdialect"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	redisclient "github.com/redis/go-redis/v9"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

const (
	redisImageTag    = "redis:8.4-alpine"
	postgresImageTag = "postgres:18.1-alpine3.23"
)

var (
	integrationDB        *sql.DB
	integrationEntClient *dbent.Client
	integrationRedis     *redisclient.Client

	redisNamespaceSeq uint64
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	if err := timezone.Init("UTC"); err != nil {
		log.Printf("failed to init timezone: %v", err)
		os.Exit(1)
	}

	// External-backend overrides let the suite run against an already-running DB
	// (e.g. a local CockroachDB node) and/or Redis instead of testcontainers.
	//   INTEGRATION_DB_DSN       - full lib/pq DSN; skips the Postgres container.
	//   INTEGRATION_REDIS_ADDR   - host:port; skips the Redis container.
	//   INTEGRATION_REDIS_PASSWORD - password for the external Redis (optional).
	externalDSN := os.Getenv("INTEGRATION_DB_DSN")
	externalRedis := os.Getenv("INTEGRATION_REDIS_ADDR")
	externalRedisPassword := os.Getenv("INTEGRATION_REDIS_PASSWORD")

	// Docker is only required for whichever backend is NOT externally provided.
	if externalDSN == "" || externalRedis == "" {
		if !dockerIsAvailable(ctx) {
			if os.Getenv("CI") != "" {
				log.Printf("docker is not available (CI=true); failing integration tests")
				os.Exit(1)
			}
			log.Printf("docker is not available; skipping integration tests (start Docker, or set INTEGRATION_DB_DSN + INTEGRATION_REDIS_ADDR)")
			os.Exit(0)
		}
	}

	var cleanups []func()
	runCleanups := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	fail := func(format string, args ...any) {
		log.Printf(format, args...)
		runCleanups()
		os.Exit(1)
	}

	// --- Database ---
	dsn := externalDSN
	if dsn == "" {
		postgresImage := selectDockerImage(ctx, postgresImageTag)
		pgContainer, err := tcpostgres.Run(
			ctx,
			postgresImage,
			tcpostgres.WithDatabase("sub2api_test"),
			tcpostgres.WithUsername("postgres"),
			tcpostgres.WithPassword("postgres"),
			tcpostgres.BasicWaitStrategies(),
		)
		if err != nil {
			fail("failed to start postgres container: %v", err)
		}
		cleanups = append(cleanups, func() { _ = pgContainer.Terminate(ctx) })

		dsn, err = pgContainer.ConnectionString(ctx, "sslmode=disable", "TimeZone=UTC")
		if err != nil {
			fail("failed to get postgres dsn: %v", err)
		}
	} else {
		log.Printf("using external database from INTEGRATION_DB_DSN")
	}

	var err error
	integrationDB, err = openSQLWithRetry(ctx, dsn, 30*time.Second)
	if err != nil {
		fail("failed to open sql db: %v", err)
	}
	cleanups = append(cleanups, func() { _ = integrationDB.Close() })

	// Publish the resolved dialect so runtime lock helpers take the CockroachDB path
	// (the harness opens the DB directly and never calls repository.InitEnt).
	dbdialect.SetCockroach(detectDialect(ctx, integrationDB).IsCockroach())

	if err := ApplyMigrations(ctx, integrationDB); err != nil {
		fail("failed to apply db migrations: %v", err)
	}

	// 创建 ent client 用于集成测试
	drv := entsql.OpenDB(dialect.Postgres, integrationDB)
	integrationEntClient = dbent.NewClient(dbent.Driver(drv))
	cleanups = append(cleanups, func() { _ = integrationEntClient.Close() })

	// --- Redis ---
	redisAddr := externalRedis
	if redisAddr == "" {
		redisContainer, err := tcredis.Run(ctx, redisImageTag)
		if err != nil {
			fail("failed to start redis container: %v", err)
		}
		cleanups = append(cleanups, func() { _ = redisContainer.Terminate(ctx) })

		redisHost, err := redisContainer.Host(ctx)
		if err != nil {
			fail("failed to get redis host: %v", err)
		}
		redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
		if err != nil {
			fail("failed to get redis port: %v", err)
		}
		redisAddr = fmt.Sprintf("%s:%d", redisHost, redisPort.Int())
	} else {
		log.Printf("using external redis from INTEGRATION_REDIS_ADDR")
	}

	integrationRedis = redisclient.NewClient(&redisclient.Options{
		Addr:     redisAddr,
		Password: externalRedisPassword,
		DB:       0,
	})
	cleanups = append(cleanups, func() { _ = integrationRedis.Close() })
	if err := integrationRedis.Ping(ctx).Err(); err != nil {
		fail("failed to ping redis: %v", err)
	}

	code := m.Run()

	runCleanups()
	os.Exit(code)
}

// execTruncate runs a TRUNCATE statement, transparently dropping the RESTART IDENTITY
// clause on CockroachDB (which has no sequences to restart and rejects the syntax).
// PostgreSQL keeps the clause unchanged.
func execTruncate(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, stmt string) (sql.Result, error) {
	if dbdialect.IsCockroach() {
		// Strip the clause regardless of surrounding whitespace (callers use both
		// "TRUNCATE x RESTART IDENTITY" and multi-line "...\nRESTART IDENTITY").
		stmt = strings.ReplaceAll(stmt, "RESTART IDENTITY", "")
	}
	return exec.ExecContext(ctx, stmt)
}

// skipOnCockroach skips a test that asserts PostgreSQL-specific migration-replay or
// schema-snapshot behavior. On CockroachDB the equivalent logic runs through the
// cockroach/ migration overlays (fresh-install semantics), so replaying the original
// PostgreSQL migrations and asserting their side effects does not apply.
func skipOnCockroach(t *testing.T, reason string) {
	t.Helper()
	if dbdialect.IsCockroach() {
		t.Skipf("skipped on CockroachDB: %s", reason)
	}
}

func dockerIsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

func selectDockerImage(ctx context.Context, preferred string) string {
	if dockerImageExists(ctx, preferred) {
		return preferred
	}

	return preferred
}

func dockerImageExists(ctx context.Context, image string) bool {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	cmd.Env = os.Environ()
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func openSQLWithRetry(ctx context.Context, dsn string, timeout time.Duration) (*sql.DB, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}

		if err := pingWithTimeout(ctx, db, 2*time.Second); err != nil {
			lastErr = err
			_ = db.Close()
			time.Sleep(250 * time.Millisecond)
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("db not ready after %s: %w", timeout, lastErr)
}

func pingWithTimeout(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return db.PingContext(pingCtx)
}

func testTx(t *testing.T) *sql.Tx {
	t.Helper()

	tx, err := integrationDB.BeginTx(context.Background(), nil)
	require.NoError(t, err, "begin tx")
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return tx
}

// testEntClient 返回全局的 ent client，用于测试需要内部管理事务的代码（如 Create/Update 方法）。
// 注意：此 client 的操作会真正写入数据库，测试结束后不会自动回滚。
func testEntClient(t *testing.T) *dbent.Client {
	t.Helper()
	return integrationEntClient
}

// testEntTx 返回一个 ent 事务，用于需要事务隔离的测试。
// 测试结束后会自动回滚，不会影响数据库状态。
func testEntTx(t *testing.T) *dbent.Tx {
	t.Helper()

	tx, err := integrationEntClient.Tx(context.Background())
	require.NoError(t, err, "begin ent tx")
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return tx
}

// testEntSQLTx 已弃用：不要在新测试中使用此函数。
// 基于 *sql.Tx 创建的 ent client 在调用 client.Tx() 时会 panic。
// 对于需要测试内部使用事务的代码，请使用 testEntClient。
// 对于需要事务隔离的测试，请使用 testEntTx。
//
// Deprecated: Use testEntClient or testEntTx instead.
func testEntSQLTx(t *testing.T) (*dbent.Client, *sql.Tx) {
	t.Helper()

	// 直接失败，避免旧测试误用导致的事务嵌套 panic。
	t.Fatalf("testEntSQLTx 已弃用：请使用 testEntClient 或 testEntTx")
	return nil, nil
}

func testRedis(t *testing.T) *redisclient.Client {
	t.Helper()

	prefix := fmt.Sprintf(
		"it:%s:%d:%d:",
		sanitizeRedisNamespace(t.Name()),
		time.Now().UnixNano(),
		atomic.AddUint64(&redisNamespaceSeq, 1),
	)

	opts := *integrationRedis.Options()
	rdb := redisclient.NewClient(&opts)
	rdb.AddHook(prefixHook{prefix: prefix})

	t.Cleanup(func() {
		ctx := context.Background()

		var cursor uint64
		for {
			keys, nextCursor, err := integrationRedis.Scan(ctx, cursor, prefix+"*", 500).Result()
			require.NoError(t, err, "scan redis keys for cleanup")
			if len(keys) > 0 {
				require.NoError(t, integrationRedis.Unlink(ctx, keys...).Err(), "unlink redis keys for cleanup")
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}

		_ = rdb.Close()
	})

	return rdb
}

func assertTTLWithin(t *testing.T, ttl time.Duration, min, max time.Duration) {
	t.Helper()
	require.GreaterOrEqual(t, ttl, min, "ttl should be >= min")
	require.LessOrEqual(t, ttl, max, "ttl should be <= max")
}

func sanitizeRedisNamespace(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

type prefixHook struct {
	prefix string
}

func (h prefixHook) DialHook(next redisclient.DialHook) redisclient.DialHook { return next }

func (h prefixHook) ProcessHook(next redisclient.ProcessHook) redisclient.ProcessHook {
	return func(ctx context.Context, cmd redisclient.Cmder) error {
		h.prefixCmd(cmd)
		return next(ctx, cmd)
	}
}

func (h prefixHook) ProcessPipelineHook(next redisclient.ProcessPipelineHook) redisclient.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redisclient.Cmder) error {
		for _, cmd := range cmds {
			h.prefixCmd(cmd)
		}
		return next(ctx, cmds)
	}
}

func (h prefixHook) prefixCmd(cmd redisclient.Cmder) {
	args := cmd.Args()
	if len(args) < 2 {
		return
	}

	prefixOne := func(i int) {
		if i < 0 || i >= len(args) {
			return
		}

		switch v := args[i].(type) {
		case string:
			if v != "" && !strings.HasPrefix(v, h.prefix) {
				args[i] = h.prefix + v
			}
		case []byte:
			s := string(v)
			if s != "" && !strings.HasPrefix(s, h.prefix) {
				args[i] = []byte(h.prefix + s)
			}
		}
	}

	switch strings.ToLower(cmd.Name()) {
	case "get", "set", "setnx", "setex", "psetex", "incr", "decr", "incrby", "expire", "pexpire", "ttl", "pttl",
		"hgetall", "hget", "hset", "hdel", "hincrbyfloat", "exists",
		"zadd", "zcard", "zrange", "zrangebyscore", "zrem", "zremrangebyscore", "zrevrange", "zrevrangebyscore", "zscore":
		prefixOne(1)
	case "mget":
		for i := 1; i < len(args); i++ {
			prefixOne(i)
		}
	case "del", "unlink":
		for i := 1; i < len(args); i++ {
			prefixOne(i)
		}
	case "eval", "evalsha", "eval_ro", "evalsha_ro":
		if len(args) < 3 {
			return
		}
		numKeys, err := strconv.Atoi(fmt.Sprint(args[2]))
		if err != nil || numKeys <= 0 {
			return
		}
		for i := 0; i < numKeys && 3+i < len(args); i++ {
			prefixOne(3 + i)
		}
	case "scan":
		for i := 2; i+1 < len(args); i++ {
			if strings.EqualFold(fmt.Sprint(args[i]), "match") {
				prefixOne(i + 1)
				break
			}
		}
	}
}

// IntegrationRedisSuite provides a base suite for Redis integration tests.
// Embedding suites should call SetupTest to initialize ctx and rdb.
type IntegrationRedisSuite struct {
	suite.Suite
	ctx context.Context
	rdb *redisclient.Client
}

// SetupTest initializes ctx and rdb for each test method.
func (s *IntegrationRedisSuite) SetupTest() {
	s.ctx = context.Background()
	s.rdb = testRedis(s.T())
}

// RequireNoError is a convenience method wrapping require.NoError with s.T().
func (s *IntegrationRedisSuite) RequireNoError(err error, msgAndArgs ...any) {
	s.T().Helper()
	require.NoError(s.T(), err, msgAndArgs...)
}

// AssertTTLWithin asserts that ttl is within [min, max].
func (s *IntegrationRedisSuite) AssertTTLWithin(ttl, min, max time.Duration) {
	s.T().Helper()
	assertTTLWithin(s.T(), ttl, min, max)
}

// IntegrationDBSuite provides a base suite for DB integration tests.
// Embedding suites should call SetupTest to initialize ctx and client.
type IntegrationDBSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	tx     *dbent.Tx
}

// SetupTest initializes ctx and client for each test method.
func (s *IntegrationDBSuite) SetupTest() {
	s.ctx = context.Background()
	// 统一使用 ent.Tx，确保每个测试都有独立事务并自动回滚。
	tx := testEntTx(s.T())
	s.tx = tx
	s.client = tx.Client()
}

// RequireNoError is a convenience method wrapping require.NoError with s.T().
func (s *IntegrationDBSuite) RequireNoError(err error, msgAndArgs ...any) {
	s.T().Helper()
	require.NoError(s.T(), err, msgAndArgs...)
}
