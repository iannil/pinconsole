// 1ad 续集测试:chat_messages repo CRUD(审计 T1-1g-1)。
//
// 验证 1g 的 chat 持久化层:Create / List / Delete(GC 用)。
// 此前这些 storage 方法零 PG 集成测试(1g 长期 🔴 主因之一)。
package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestChat_CreateAndList — T1-1g-1:
// CreateChatMessage 写入后,ListChatMessagesBySession 应能读出。
func TestChat_CreateAndList(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	// seed visitor + session(FK 要求)
	vid := uuid.New()
	sid := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, vid, tenantID, "1ac-chat-fp-"+vid.String()[:8]); err != nil {
		t.Fatalf("seed visitor: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at)
		VALUES ($1, $2, $3, NOW())
	`, sid, tenantID, vid); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, vid)

	// create 3 messages
	msgs := []struct {
		sender  string
		content string
	}{
		{"operator", "hello"},
		{"visitor", "hi"},
		{"operator", "how can I help?"},
	}
	var firstID int64
	for i, m := range msgs {
		got, err := pg.CreateChatMessage(ctx, tenantID, sid, m.sender, m.content)
		if err != nil {
			t.Fatalf("create #%d: %v", i, err)
		}
		if got.Content != m.content {
			t.Errorf("create #%d: content=%q want %q", i, got.Content, m.content)
		}
		if got.Sender != m.sender {
			t.Errorf("create #%d: sender=%q want %q", i, got.Sender, m.sender)
		}
		if i == 0 {
			firstID = got.ID
		}
	}

	// list 全部(sinceID=0)
	list, err := pg.ListChatMessagesBySession(ctx, sid, 0, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("list len=%d want 3", len(list))
	}

	// list sinceID=firstID(应只返回后 2 条)
	list2, err := pg.ListChatMessagesBySession(ctx, sid, firstID, 100)
	if err != nil {
		t.Fatalf("list sinceID: %v", err)
	}
	if len(list2) != 2 {
		t.Errorf("list sinceID=%d len=%d want 2", firstID, len(list2))
	}
}

// TestChat_SenderIsOperatorOrVisitor — T1-1g-1 边界:
// sender 字段接受 operator / visitor 两种(由 chat handler 强制,详见 chat.go)。
func TestChat_SenderIsOperatorOrVisitor(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	vid := uuid.New()
	sid := uuid.New()
	pool.Exec(ctx, `INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		vid, tenantID, "1ac-sender-fp-"+vid.String()[:8])
	pool.Exec(ctx, `INSERT INTO sessions (id, tenant_id, visitor_id, started_at) VALUES ($1, $2, $3, NOW())`,
		sid, tenantID, vid)
	defer pool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, vid)

	for _, sender := range []string{"operator", "visitor"} {
		if _, err := pg.CreateChatMessage(ctx, tenantID, sid, sender, "test-"+sender); err != nil {
			t.Errorf("create sender=%q: %v", sender, err)
		}
	}
}

// TestChat_GC_ListOlderThanAndDelete — T1-1g-1 + T1-1l-4 chat GC:
// 验证 ListChatMessagesOlderThan + DeleteChatMessagesByID(GC 路径)。
func TestChat_GC_ListOlderThanAndDelete(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	vid := uuid.New()
	sid := uuid.New()
	pool.Exec(ctx, `INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		vid, tenantID, "1ac-chatgc-fp-"+vid.String()[:8])
	pool.Exec(ctx, `INSERT INTO sessions (id, tenant_id, visitor_id, started_at) VALUES ($1, $2, $3, NOW())`,
		sid, tenantID, vid)
	defer pool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, vid)

	// create 2 messages
	m1, err := pg.CreateChatMessage(ctx, tenantID, sid, "operator", "old")
	if err != nil {
		t.Fatalf("create m1: %v", err)
	}
	m2, err := pg.CreateChatMessage(ctx, tenantID, sid, "operator", "recent")
	if err != nil {
		t.Fatalf("create m2: %v", err)
	}

	// 手动改 m1.created_at 为 2 天前(GC threshold 24h)
	if _, err := pool.Exec(ctx, `UPDATE chat_messages SET created_at = NOW() - INTERVAL '2 days' WHERE id = $1`, m1.ID); err != nil {
		t.Fatalf("age m1: %v", err)
	}

	// GC:threshold = 24h ago,应只列出 m1
	ids, err := pg.ListChatMessagesOlderThan(ctx, timeNowAdd24hAgo(), 100)
	if err != nil {
		t.Fatalf("ListOlderThan: %v", err)
	}
	found := false
	for _, id := range ids {
		if id == m1.ID {
			found = true
		}
		if id == m2.ID {
			t.Errorf("m2 should NOT be listed (< threshold)")
		}
	}
	if !found {
		t.Errorf("m1 should be listed (>= threshold)")
	}

	// Delete m1
	if err := pg.DeleteChatMessagesByID(ctx, []int64{m1.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// 验证 m1 gone,m2 仍在
	var n1, n2 int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM chat_messages WHERE id = $1`, m1.ID).Scan(&n1)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM chat_messages WHERE id = $1`, m2.ID).Scan(&n2)
	if n1 != 0 {
		t.Errorf("m1 should be deleted (n=%d)", n1)
	}
	if n2 != 1 {
		t.Errorf("m2 should still exist (n=%d)", n2)
	}
}

// TestChat_DeleteEmptyIDs_NoOp — 边界:DeleteChatMessagesByID(nil) 应 no-op 不报错。
func TestChat_DeleteEmptyIDs_NoOp(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()
	pg := &Postgres{Pool: pool}
	ctx := context.Background()

	if err := pg.DeleteChatMessagesByID(ctx, nil); err != nil {
		t.Errorf("DeleteChatMessagesByID(ctx, nil): %v, want nil", err)
	}
	if err := pg.DeleteChatMessagesByID(ctx, []int64{}); err != nil {
		t.Errorf("DeleteChatMessagesByID(ctx, []): %v, want nil", err)
	}
}

// timeNowAdd24hAgo 返回 24 小前的时间(GC threshold 模拟)。
func timeNowAdd24hAgo() time.Time {
	return time.Now().Add(-24 * time.Hour)
}
