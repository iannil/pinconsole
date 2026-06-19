// 1ag 测试:replay.go HTTP 入口 + 纯函数行为级测试。
//
// 补 1d 未覆盖的 replay handler 路径:
//   - parseSince 默认/合法/非法单元(纯函数,零依赖)
//   - getSessionReplay 非 UUID → 400(不触达 stores,uuid.Parse 在 stores 调用前)
//   - listEndedSessions 非法 since → 400(parseSince 在 stores 调用前)
//
// 模式同 auth_http_test.go:r.ServeHTTP + httptest.Recorder。
// happy path 需 PG + MinIO seed,留 follow-up(见 spec §范围边界)。
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestParseSince_DefaultsTo24h — 空字符串默认 24h(replay.go:287-289)。
func TestParseSince_DefaultsTo24h(t *testing.T) {
	got, err := parseSince("")
	if err != nil {
		t.Fatalf("parseSince(\"\") err = %v", err)
	}
	if got != 24*time.Hour {
		t.Errorf("parseSince(\"\") = %v, want 24h", got)
	}
}

// TestParseSince_HoursAndDays — 支持 "12h" / "7d" 格式。
func TestParseSince_HoursAndDays(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"1h", 1 * time.Hour},
		{"12h", 12 * time.Hour},
		{"1d", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseSince(tc.in)
			if err != nil {
				t.Fatalf("parseSince(%q) err = %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("parseSince(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseSince_InvalidUnit_ReturnsError — 不支持的单位(如 w/m/s)必返 error。
func TestParseSince_InvalidUnit_ReturnsError(t *testing.T) {
	cases := []string{"7w", "30m", "60s", "1y", "1x"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := parseSince(in)
			if err == nil {
				t.Errorf("parseSince(%q) err = nil, want non-nil (unsupported unit)", in)
			}
		})
	}
}

// TestParseSince_TooShort_ReturnsError — 长度 < 2 必返 error(replay.go:292-294)。
// "" 是合法(默认 24h),从负向用例排除。
func TestParseSince_TooShort_ReturnsError(t *testing.T) {
	invalid := []string{"x", "1"}
	for _, in := range invalid {
		t.Run(in, func(t *testing.T) {
			_, err := parseSince(in)
			if err == nil {
				t.Errorf("parseSince(%q) err = nil, want non-nil (too short)", in)
			}
		})
	}
}

// TestParseSince_NonNumericPrefix_ReturnsError — 数字部分非整数必返 error。
func TestParseSince_NonNumericPrefix_ReturnsError(t *testing.T) {
	cases := []string{"abd", "7.5h"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := parseSince(in)
			if err == nil {
				t.Errorf("parseSince(%q) err = nil, want non-nil (non-numeric prefix)", in)
			}
		})
	}
}

// TestParseSince_RejectsNonPositive — 1aj:负数与零必返 error。
//
// Atoi("-1") 不报错,parseSince 此前会返回 -24h 负 duration;
// 零("0h")语义模糊(用户应传 "" 走默认)。
// 两者都拒绝,符合"输入校验在边界"原则。
func TestParseSince_RejectsNonPositive(t *testing.T) {
	cases := []string{"-1d", "-12h", "-100d", "0h", "0d"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := parseSince(in)
			if err == nil {
				t.Errorf("parseSince(%q) = %v, want error (non-positive duration)", in, got)
			}
		})
	}
}

// newReplayTestEngine 构造仅挂 replay 路由的 gin engine,不依赖 stores。
// 用于测试 handler 在触达 stores 前的拒绝路径(参数解析、binding)。
func newReplayTestEngine(h *ReplayHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)
	return r
}

// TestGetSessionReplay_InvalidUUID_Returns400 — 非 UUID 必返 400 invalid_session_id。
//
// uuid.Parse 在 stores 调用前(replay.go:138-143),所以无需 mock。
func TestGetSessionReplay_InvalidUUID_Returns400(t *testing.T) {
	h := &ReplayHandler{logger: testLogger()}
	r := newReplayTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/not-a-uuid/replay", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_session_id") {
		t.Errorf("body = %q, want contains 'invalid_session_id'", w.Body.String())
	}
}

// TestGetSessionReplay_ValidUUIDPassesParsing — 合法 UUID 通过解析阶段,
// 但因 stores 未注入会在下一步 nil deref。此处仅断言不返 400 解析错。
//
// 用 recover wrapper 保护,允许 panic 但断言 status != 400。
func TestGetSessionReplay_ValidUUIDPassesParsing(t *testing.T) {
	h := &ReplayHandler{logger: testLogger()}
	r := newReplayTestEngine(h)
	// 用 defer recover 防 nil stores panic 中断测试
	defer func() { _ = recover() }()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/550e8400-e29b-41d4-a716-446655440000/replay", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 关键:UUID 合法 → 不应返 400 invalid_session_id
	// 实际可能 500(panic)或 200(若 gin recover 接住并 500);只要不是 400 即可
	if w.Code == http.StatusBadRequest {
		t.Errorf("status = 400 for valid UUID — uuid.Parse 误拒合法 UUID")
	}
}

// TestListEndedSessions_InvalidSince_Returns400 — 非 since 格式必返 400 invalid_since。
//
// parseSince 在 stores 调用前(replay.go:66-70)。
func TestListEndedSessions_InvalidSince_Returns400(t *testing.T) {
	h := &ReplayHandler{logger: testLogger()}
	r := newReplayTestEngine(h)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/ended?since=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_since") {
		t.Errorf("body = %q, want contains 'invalid_since'", w.Body.String())
	}
}

// TestListEndedSessions_ValidSincePassesParsing — 合法 since 不返 400。
//
// 用 defer recover 防 nil stores panic;只断言 status != 400。
func TestListEndedSessions_ValidSincePassesParsing(t *testing.T) {
	h := &ReplayHandler{logger: testLogger()}
	r := newReplayTestEngine(h)
	defer func() { _ = recover() }()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/ended?since=7d", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest {
		t.Errorf("status = 400 for since=7d — parseSince 误拒合法值")
	}
}
