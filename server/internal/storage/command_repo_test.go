// 1ai-b 测试:co_browsing_commands 表 repo PG 集成测试。
//
// 补 1e 双向通道依赖的 CreateCoBrowsingCommand / ListBySession / DeleteOlderThan。
// 沿用 1ai 既定 helperPGPool + seedVisitor 模式。
package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// assertJSONEqual 比较两段 JSON 的语义(非字节级)。
// PG JSONB 列存时会规范化空格/键序,直接 bytes.Equal 会假阳性失败。
func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var gv, wv any
	if err := json.Unmarshal(got, &gv); err != nil {
		t.Fatalf("got is not valid JSON: %v\nraw: %s", err, got)
	}
	if err := json.Unmarshal(want, &wv); err != nil {
		t.Fatalf("want is not valid JSON: %v\nraw: %s", err, want)
	}
	if !jsonEqual(gv, wv) {
		t.Errorf("JSON mismatch:\n got:  %s\n want: %s", got, want)
	}
}

// jsonEqual 递归比较解码后的 JSON(any / map / slice / scalar)。
func jsonEqual(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !jsonEqual(v, bv[k]) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// TestCreateCoBrowsingCommand_AndList — Create + List 字段一致(含 JSON payload round-trip)。
func TestCreateCoBrowsingCommand_AndList(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	session, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	payload := []byte(`{"x":123,"y":456,"text":"hello 世界"}`)
	cmd, err := pg.CreateCoBrowsingCommand(ctx, CoBrowsingCommand{
		TenantID:    DefaultTenantID,
		SessionID:   session.ID,
		OperatorID:  "operator-1aib",
		CommandType: "fill_input",
		Payload:     payload,
	})
	if err != nil {
		t.Fatalf("CreateCoBrowsingCommand: %v", err)
	}
	if cmd == nil || cmd.ID == uuid.Nil {
		t.Fatal("CreateCoBrowsingCommand returned nil/zero-ID")
	}
	if cmd.CommandType != "fill_input" {
		t.Errorf("CommandType = %q, want fill_input", cmd.CommandType)
	}
	// PG JSONB 列会规范化空格,用语义比较(非字节级)
	assertJSONEqual(t, cmd.Payload, payload)

	list, err := pg.ListCoBrowsingCommandsBySession(ctx, session.ID, 100)
	if err != nil {
		t.Fatalf("ListCoBrowsingCommandsBySession: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	if list[0].ID != cmd.ID {
		t.Errorf("list[0].ID = %s, want %s", list[0].ID, cmd.ID)
	}
	assertJSONEqual(t, list[0].Payload, payload)
}

// TestCreateCoBrowsingCommand_NullNodeID — TargetNodeID=nil 必须存为 NULL(非 0)。
func TestCreateCoBrowsingCommand_NullNodeID(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	session, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	cmd, err := pg.CreateCoBrowsingCommand(ctx, CoBrowsingCommand{
		TenantID:    DefaultTenantID,
		SessionID:   session.ID,
		OperatorID:  "op",
		CommandType: "release_control",
		// TargetNodeID 故意 nil(release_control 无 target)
		Payload: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateCoBrowsingCommand with nil TargetNodeID: %v", err)
	}
	if cmd.TargetNodeID != nil {
		t.Errorf("TargetNodeID = %v, want nil", cmd.TargetNodeID)
	}

	// 同样验证 non-nil 路径
	nodeID := int32(42)
	cmd2, err := pg.CreateCoBrowsingCommand(ctx, CoBrowsingCommand{
		TenantID:     DefaultTenantID,
		SessionID:    session.ID,
		OperatorID:   "op",
		CommandType:  "cursor_highlight",
		TargetNodeID: &nodeID,
		Payload:      []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateCoBrowsingCommand with TargetNodeID=42: %v", err)
	}
	if cmd2.TargetNodeID == nil || *cmd2.TargetNodeID != 42 {
		t.Errorf("TargetNodeID = %v, want 42", cmd2.TargetNodeID)
	}
}

// TestListCoBrowsingCommandsBySession_OrderedByCreated — 多条按 created_at 正序。
func TestListCoBrowsingCommandsBySession_OrderedByCreated(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	session, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// 顺序创建 3 条不同 command_type
	types := []string{"click", "scroll", "fill_input"}
	var createdIDs []uuid.UUID
	for _, ct := range types {
		cmd, err := pg.CreateCoBrowsingCommand(ctx, CoBrowsingCommand{
			TenantID:    DefaultTenantID,
			SessionID:   session.ID,
			OperatorID:  "op",
			CommandType: ct,
			Payload:     []byte(`{}`),
		})
		if err != nil {
			t.Fatalf("CreateCoBrowsingCommand(%s): %v", ct, err)
		}
		createdIDs = append(createdIDs, cmd.ID)
		time.Sleep(5 * time.Millisecond) // 确保 created_at 有差异
	}

	list, err := pg.ListCoBrowsingCommandsBySession(ctx, session.ID, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("list len = %d, want 3", len(list))
	}
	for i, want := range createdIDs {
		if list[i].ID != want {
			t.Errorf("list[%d].ID = %s, want %s (顺序应正序)", i, list[i].ID, want)
		}
	}
}

// TestDeleteCoBrowsingCommandsOlderThan — 阈值前的删,后的留。
func TestDeleteCoBrowsingCommandsOlderThan(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	ctx := context.Background()
	pg := &Postgres{Pool: pool}

	visitorID, fp := seedVisitor(t, pool)
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM visitors WHERE fingerprint = $1`, fp) }()

	session, err := pg.CreateSession(ctx, DefaultTenantID, visitorID, "", "")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// 创建 2 条,记录中间时刻,然后再创建 1 条
	old1, _ := pg.CreateCoBrowsingCommand(ctx, CoBrowsingCommand{
		TenantID: DefaultTenantID, SessionID: session.ID, OperatorID: "op",
		CommandType: "click", Payload: []byte(`{}`),
	})
	time.Sleep(100 * time.Millisecond) // 1ai-h:增大 sleep 避免 PG 时钟漂移导致边界 flaky
	threshold := time.Now()
	time.Sleep(100 * time.Millisecond)
	new1, _ := pg.CreateCoBrowsingCommand(ctx, CoBrowsingCommand{
		TenantID: DefaultTenantID, SessionID: session.ID, OperatorID: "op",
		CommandType: "scroll", Payload: []byte(`{}`),
	})

	// 删除 threshold 之前
	if err := pg.DeleteCoBrowsingCommandsOlderThan(ctx, threshold); err != nil {
		t.Fatalf("DeleteCoBrowsingCommandsOlderThan: %v", err)
	}

	list, err := pg.ListCoBrowsingCommandsBySession(ctx, session.ID, 100)
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	// old1 应被删,new1 应保留
	for _, c := range list {
		if c.ID == old1.ID {
			t.Error("old1 should be deleted (created_at < threshold)")
		}
	}
	foundNew := false
	for _, c := range list {
		if c.ID == new1.ID {
			foundNew = true
		}
	}
	if !foundNew {
		t.Error("new1 should remain (created_at >= threshold)")
	}
}
