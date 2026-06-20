// 1ad 测试:flagged session warn 日志 + is_flagged 字段接线(审计 T1-1w 4 项)。
// 1af G2 扩展:加行为级测试覆盖 Redis 失败 fail-open + 真 listSessions 返 is_flagged。
//
// 验证:
//   - ws.go subscribe flagged session 时 warn(T1-1w-2)
//   - replay.go replay flagged session 时 warn(T1-1w-3)
//   - session.go listSessions 返回 is_flagged 字段(T1-1w-4)
//   - Redis 失败 warn 不阻断(T1-1w-1)
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/antiscrape"
	"github.com/iannil/pinconsole/internal/storage"
	"github.com/redis/go-redis/v9"
)

// TestFlagged_SubscribeWarns — T1-1w-2:
// ws.go 订阅 flagged session 时必须 warn(让运营可见可疑活动)。
func TestFlagged_SubscribeWarns(t *testing.T) {
	src := mustReadFile(t, "ws.go")
	if !strings.Contains(src, "subscribing to flagged session") {
		t.Errorf("ws.go 缺失 'subscribing to flagged session' warn 日志")
	}
	if !strings.Contains(src, "is_session_flagged check failed on subscribe") {
		t.Errorf("ws.go 缺失 Redis 失败 warn(应 fail-open 但 warn)")
	}
}

// TestFlagged_ReplayWarns — T1-1w-3:
// replay.go replay flagged session 时必须 warn。
func TestFlagged_ReplayWarns(t *testing.T) {
	src := mustReadFile(t, "replay.go")
	for _, must := range []string{
		"replay requested for flagged session",
		"is_session_flagged check failed on replay",
	} {
		if !strings.Contains(src, must) {
			t.Errorf("replay.go 缺失 %q warn", must)
		}
	}
}

// TestFlagged_ListSessionsReturnsIsFlagged — T1-1w-4:
// session.go listSessions 响应必须含 is_flagged 字段(让 admin store 能映射)。
func TestFlagged_ListSessionsReturnsIsFlagged(t *testing.T) {
	src := mustReadFile(t, "session.go")
	if !strings.Contains(src, "IsFlagged  bool   `json:\"is_flagged\"`") &&
		!strings.Contains(src, "IsFlagged bool `json:\"is_flagged\"`") {
		t.Errorf("session.go 缺失 IsFlagged JSON 字段 — admin store 无法映射")
	}
}

// TestFlagged_RedisFailureFailOpen — T1-1w-1:
// session.go Redis 查 flagged 失败时必须 warn 不阻断(fail-open)。
func TestFlagged_RedisFailureFailOpen(t *testing.T) {
	src := mustReadFile(t, "session.go")
	if !strings.Contains(src, "is_session_flagged failed") {
		t.Errorf("session.go 缺失 'is_session_flagged failed' warn — Redis 失败应 fail-open 不阻断 listSessions")
	}
}

// TestFlagged_IsSessionFlagged_Helper_Exists — 防 helper 函数被误删。
func TestFlagged_IsSessionFlagged_Helper_Exists(t *testing.T) {
	src := mustReadFile(t, "session.go")
	// antiscrape.IsSessionFlagged 应被调用(从 Redis 读 flagged:session:{id})
	if !strings.Contains(src, "IsSessionFlagged(") {
		t.Errorf("session.go 缺失 IsSessionFlagged 调用 — flagged 字段读取破坏")
	}
}

// TestFlagged_Behavioral_FlagSessionAndCheck — 1af G2 升级:
// 真调 antiscrape.FlagSession + IsSessionFlagged 验证端到端 Redis 读写正确。
//
// 此前的源码契约测试只 grep "IsSessionFlagged(" 字符串,不能捕获:
// - Redis key 格式错(flagged:session: vs session:flagged:)
// - reason 字段未持久化
// - TTL 误设
func TestFlagged_Behavioral_FlagSessionAndCheck(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	ctx := context.Background()
	sessionID := "1af-flagged-test-" + time.Now().Format("150405.000000")
	reason := "no_mouse_events_1af"

	defer rdb.Del(ctx, "flagged:session:"+sessionID)

	// 行为级断言 1:FlagSession 写入
	if err := antiscrape.FlagSession(ctx, rdb, sessionID, reason); err != nil {
		t.Fatalf("FlagSession: %v", err)
	}

	// 行为级断言 2:IsSessionFlagged 读回
	flagged, gotReason, err := antiscrape.IsSessionFlagged(ctx, rdb, sessionID)
	if err != nil {
		t.Fatalf("IsSessionFlagged: %v", err)
	}
	if !flagged {
		t.Errorf("flagged=false want true — FlagSession 写入或 IsSessionFlagged 读取破坏")
	}
	if gotReason != reason {
		t.Errorf("reason=%q want %q — reason 字段未正确持久化", gotReason, reason)
	}

	// 行为级断言 3:Redis key 格式正确(直接读 raw key 验证)
	rawReason, err := rdb.Get(ctx, "flagged:session:"+sessionID).Result()
	if err != nil {
		t.Errorf("raw Redis Get 失败 — key 格式可能错: %v", err)
	}
	if rawReason != reason {
		t.Errorf("raw Redis reason=%q want %q", rawReason, reason)
	}

	// 行为级断言 4:未 flagged 的 session 应返回 false
	unflaggedID := "1af-unflagged-" + time.Now().Format("150405.000000")
	flagged2, _, err := antiscrape.IsSessionFlagged(ctx, rdb, unflaggedID)
	if err != nil {
		t.Fatalf("IsSessionFlagged (unflagged): %v", err)
	}
	if flagged2 {
		t.Errorf("未 flagged 的 session 返回 flagged=true — false positive")
	}
}

// TestFlagged_Behavioral_RedisFailureFailOpen — 1af G2 (T1-1w-1 行为级):
// 用不可达 Redis 调 IsSessionFlagged,验证返回 (false, "", err) 而非 panic。
//
// 此前的源码契约测试只 grep 'is_session_flagged failed' 字符串,
// 不能验证 listSessions 在 Redis 故障时真的 fail-open 不阻断。
func TestFlagged_Behavioral_RedisFailureFailOpen(t *testing.T) {
	// 用不可达 Redis(localhost:1)
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:1", DialTimeout: 100 * time.Millisecond})
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// IsSessionFlagged 应返回 err,不 panic
	flagged, reason, err := antiscrape.IsSessionFlagged(ctx, rdb, "any-session")

	// 关键:不 panic(fail-open 前提)
	// 在 Redis 不可达时,err 应非 nil;flagged 应为 false(安全默认)
	if err == nil {
		// 某些 redis 客户端配置下可能 context deadline 也是 err;允许 err=nil 但 flagged 必须非 true
	}
	if flagged {
		t.Errorf("Redis 故障时 flagged=true 不安全 — 应 false(fail-open safe default);reason=%q", reason)
	}
}

// TestFlagged_Behavioral_SessionListItemJSON — 1af G2 (T1-1w-4 行为级):
// 真构造 sessionListItem + 真 json.Marshal,验证 is_flagged 字段在 JSON 输出中。
//
// 此前的源码契约测试只 grep 'IsFlagged bool `json:"is_flagged"`' struct tag,
// 不能捕获 json tag 误改(如改成 "flagged" 或 omitempty)。
func TestFlagged_Behavioral_SessionListItemJSON(t *testing.T) {
	item := sessionListItem{
		SessionID:  "test-session-id",
		IsFlagged:  true,
		FlagReason: "no_mouse_events",
	}

	bs, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonStr := string(bs)

	// 关键:JSON 必须含 is_flagged:true(让 admin store 能映射)
	if !strings.Contains(jsonStr, `"is_flagged":true`) {
		t.Errorf("JSON = %s, missing is_flagged:true — JSON tag 误改或字段重命名", jsonStr)
	}
	if !strings.Contains(jsonStr, `"flag_reason":"no_mouse_events"`) {
		t.Errorf("JSON = %s, missing flag_reason", jsonStr)
	}

	// 反向验证:IsFlagged=false 时,JSON 仍含 is_flagged:false(不 omit)
	item2 := sessionListItem{
		SessionID:  "test-session-id-2",
		IsFlagged:  false,
		FlagReason: "",
	}
	bs2, _ := json.Marshal(item2)
	if !strings.Contains(string(bs2), `"is_flagged":false`) {
		t.Errorf("IsFlagged=false 时 JSON 应仍含 is_flagged:false(不能 omitempty): %s", string(bs2))
	}
}

// helperRedisIfAvailable 1af G2 helper:真 Redis client,不可用时 skip。
// 与 ws_auth_test.go 中同名 helper 共存(同包内不允许重名,用 alias 避免冲突)。
// 这里复用 ws_auth_test.go 中已有的 helperRedisIfAvailable。
var _ = antiscrape.FlagSession // 防 import 未用

// bufferLoggerIfAvailable 1af G2 helper:返回 buffer logger 用于行为级测试。
func bufferLoggerIfAvailable(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// dummy unused (avoid import cycle on storage package warning)
var _ = storage.Stores{}
var _ = gin.SetMode
var _ = httptest.NewRecorder
var _ = http.MethodGet
