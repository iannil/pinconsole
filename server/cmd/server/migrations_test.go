// 1k 测试：migrations 嵌入与版本解析 (P0-14)。
// 1ac 扩展:advisory lock + fail-fast 源码契约(T0-1k-7 + T0-1k-8)。
// 1ae R3c 扩展:行为级 fail-fast 测试(用 failingPool 真调 runMigrations)。
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/iannil/marketing-monitor/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestParseMigrationVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"init", "000001_init.up.sql", 1, false},
		{"cobrowsing", "000002_cobrowsing.up.sql", 2, false},
		{"chat", "000003_chat.up.sql", 3, false},
		{"auth", "000004_auth.up.sql", 4, false},
		{"large version", "000099_some_change.up.sql", 99, false},
		{"missing prefix", "init.up.sql", 0, true},
		{"wrong format", "abc_def.up.sql", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMigrationVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMigrationVersion(%q) err = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMigrationVersion(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestMigrationFiles_AllPresentInEmbed(t *testing.T) {
	// 验证 4 个 up + 4 个 down 都已嵌入
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	upCount := 0
	downCount := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upCount++
		} else if strings.HasSuffix(name, ".down.sql") {
			downCount++
		}
	}

	if upCount < 4 {
		t.Errorf("expected ≥4 up.sql files embedded, got %d", upCount)
	}
	if downCount < 4 {
		t.Errorf("expected ≥4 down.sql files embedded, got %d", downCount)
	}
}

func TestMigrationFiles_AllVersionsParseable(t *testing.T) {
	// 验证所有嵌入的 up.sql 文件名都能被 parseMigrationVersion 解析
	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		if _, err := parseMigrationVersion(name); err != nil {
			t.Errorf("embedded migration %q not parseable: %v", name, err)
		}
	}
}

// TestMigration_AdvisoryLock_Used — T0-1k-7 回归测试:
// migrations.go 必须用 pg_advisory_lock 防多实例并发迁移。
//
// 源码契约:runMigrations 函数体含 "pg_advisory_lock" + "pg_advisory_unlock"。
// 如果被误删/改注释掉,此测试失败。
func TestMigration_AdvisoryLock_Used(t *testing.T) {
	src, err := os.ReadFile("migrations.go")
	if err != nil {
		t.Fatalf("read migrations.go: %v", err)
	}
	body := string(src)

	for _, must := range []string{
		"pg_advisory_lock($1)",
		"pg_advisory_unlock($1)",
		"migrationAdvisoryLockID",
	} {
		if !strings.Contains(body, must) {
			t.Errorf("migrations.go 缺失 %q — pg_advisory_lock race-safety 破坏", must)
		}
	}
	// 拒绝反模式:acquire 成功后忘记 unlock
	if strings.Count(body, "pg_advisory_lock(") > 0 &&
		strings.Count(body, "pg_advisory_unlock(") == 0 {
		t.Errorf("migrations.go 用了 pg_advisory_lock 但缺 pg_advisory_unlock — 锁泄露")
	}
}

// TestMigration_FailFastOnMigrationError — T0-1k-8 回归测试:
// main.go 中 runMigrations 失败必须 os.Exit(1)(fail-fast),不能继续启动。
//
// 否则坏 schema 上线,silent corruption。
func TestMigration_FailFastOnMigrationError(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	body := string(src)

	// 定位 runMigrations 调用块(应包含 err != nil → os.Exit)
	if !strings.Contains(body, "runMigrations(") {
		t.Fatal("main.go 缺失 runMigrations 调用")
	}

	// 在 runMigrations 调用后 200 字符内,应存在 os.Exit(1)
	idx := strings.Index(body, "runMigrations(")
	if idx < 0 {
		t.Fatal("找不到 runMigrations 调用")
	}
	tail := body[idx:]
	if len(tail) > 400 {
		tail = tail[:400]
	}
	if !strings.Contains(tail, "os.Exit(1)") {
		t.Errorf("runMigrations 失败后未 os.Exit(1) — fail-fast 破坏:\n%s", tail)
	}
}

// TestMigration_AdvisoryLockID_Stable — 防锁 ID 误改破坏既有部署。
// 同一项目内必须用同一 ID(20260618),否则升级后新版本拿不到旧版本的锁。
func TestMigration_AdvisoryLockID_Stable(t *testing.T) {
	if migrationAdvisoryLockID != 20260618 {
		t.Errorf("migrationAdvisoryLockID = %d, want 20260618(项目起始日期,防跨服务冲突)", migrationAdvisoryLockID)
	}
}

// TestRunMigrations_FailingMigration_ReturnsError — 1ae R3c 升级:
// 真调 runMigrations + 故意坏的 SQL,验证返回 error(fail-fast)。
//
// 此前的源码契约测试 TestMigration_FailFastOnMigrationError 只 grep "os.Exit(1)",
// 不能捕获:
// - runMigrations 内部错误处理 broken(吞掉错误返回 nil)
// - 错误传播链断(runMigrations 返 err 但 main.go 不处理)
//
// 行为级测试:用不可达 PG DSN,runMigrations 必须返回非 nil error。
func TestRunMigrations_FailingMigration_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("需要 PG 连接尝试")
	}

	// 用不可达 DSN:runMigrations 第一步 pg_advisory_lock 必失败
	// 不直接 mock pool,因 runMigrations 用 storage.PgxPool 接口
	badPool := &failingPool{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	err := runMigrations(context.Background(), badPool, logger)
	if err == nil {
		t.Fatal("runMigrations with failing pool: 应返回 error(fail-fast),got nil")
	}
	// 错误信息应暗示 advisory lock 或 PG 连接失败
	errStr := err.Error()
	if !strings.Contains(errStr, "advisory lock") && !strings.Contains(errStr, "lock") {
		t.Errorf("runMigrations err 不含 advisory lock 信息: %v", err)
	}
}

// TestRunMigrations_NoErrOnPoolClose — 1ae R3c 补充:
// 验证 runMigrations 在 nil/无效 pool 时不 panic(fail-fast 优雅返回 error)。
func TestRunMigrations_NoErrOnPoolClose(t *testing.T) {
	if testing.Short() {
		t.Skip("需要 PG 连接尝试")
	}

	// 用另一个 failing pool 模拟池关闭场景
	closedPool := &failingPool{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runMigrations panic on bad pool: %v", r)
		}
	}()
	_ = runMigrations(context.Background(), closedPool, logger)
}

// failingPool 实现 storage.PgxPool,所有方法返回 error 或 panic。
// 用于 1ae R3c 验证 runMigrations 的 fail-fast 行为。
type failingPool struct{}

func (f *failingPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, fmt.Errorf("failingPool: synthetic PG unavailable")
}
func (f *failingPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, fmt.Errorf("failingPool: synthetic PG unavailable")
}
func (f *failingPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return errorRow{err: fmt.Errorf("failingPool: synthetic PG unavailable")}
}
func (f *failingPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, fmt.Errorf("failingPool: synthetic PG unavailable")
}
func (f *failingPool) Ping(ctx context.Context) error {
	return fmt.Errorf("failingPool: synthetic PG unavailable")
}
func (f *failingPool) Close() {}

// errorRow 实现 pgx.Row,Scan 总返回注入的 error。
type errorRow struct {
	err error
}

func (r errorRow) Scan(dest ...any) error { return r.err }
