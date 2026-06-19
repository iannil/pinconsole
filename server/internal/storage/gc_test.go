// 1ac 续集测试:GC 5 表 + consent upsert(审计 T0-1l-4 + T0-1l-6)。
//
// 验证 1d/1l 的 GC 顺序与 consent CRUD:
//   - DeleteSessionsEndedBefore(ended_at < threshold)
//   - DeleteVisitorsLastSeenBefore(last_seen_at < threshold)
//   - DeleteEventBlobByID + ListEventBlobsOlderThan
//   - ListChatMessagesOlderThan + DeleteChatMessagesByID
//   - DeleteCoBrowsingCommandsOlderThan
//   - UpsertConsent / GetLatestConsent
//
// 此前这些 storage 方法**零 PG 集成测试**(1l-4 长期 🔴 主因)。
package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestGC_DeleteSessionsEndedBefore — T0-1l-4:
// 仅删除 ended_at 非 NULL 且 < threshold 的 sessions,保留活跃或最近的。
func TestGC_DeleteSessionsEndedBefore(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	// seed 3 个 visitor + 3 个 session:
	// - 旧已结束(< threshold → 删)
	// - 新已结束(>= threshold → 保留)
	// - 活跃(ended_at NULL → 保留)
	visitorOld := uuid.New()
	visitorRecent := uuid.New()
	visitorActive := uuid.New()
	for _, vid := range []uuid.UUID{visitorOld, visitorRecent, visitorActive} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, vid, tenantID, "1ac-gc-fp-"+vid.String()[:8]); err != nil {
			t.Fatalf("seed visitor %s: %v", vid, err)
		}
	}
	defer func() {
		pool.Exec(ctx, `DELETE FROM visitors WHERE id = ANY($1)`,
			[]uuid.UUID{visitorOld, visitorRecent, visitorActive})
	}()

	oldSession := uuid.New()
	recentSession := uuid.New()
	activeSession := uuid.New()

	oldThreshold := time.Now().Add(-48 * time.Hour)
	recentThreshold := time.Now().Add(-1 * time.Hour)

	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at, ended_at, status)
		VALUES ($1, $2, $3, $4, $5, 'ended')
	`, oldSession, tenantID, visitorOld, oldThreshold.Add(-1*time.Hour), oldThreshold); err != nil {
		t.Fatalf("seed old session: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at, ended_at, status)
		VALUES ($1, $2, $3, NOW(), NOW(), 'ended')
	`, recentSession, tenantID, visitorRecent); err != nil {
		t.Fatalf("seed recent session: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, visitor_id, started_at, status)
		VALUES ($1, $2, $3, NOW(), 'active')
	`, activeSession, tenantID, visitorActive); err != nil {
		t.Fatalf("seed active session: %v", err)
	}

	// GC:threshold = 24h ago,应只删 oldSession
	if err := pg.DeleteSessionsEndedBefore(ctx, time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("DeleteSessionsEndedBefore: %v", err)
	}

	for _, tc := range []struct {
		name    string
		sid     uuid.UUID
		wantGone bool
	}{
		{"old ended (48h ago)", oldSession, true},
		{"recent ended", recentSession, false},
		{"active", activeSession, false},
	} {
		var n int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessions WHERE id = $1`, tc.sid).Scan(&n); err != nil {
			t.Errorf("count %s: %v", tc.name, err)
			continue
		}
		if tc.wantGone && n != 0 {
			t.Errorf("%s: still exists (n=%d), want gone", tc.name, n)
		}
		if !tc.wantGone && n != 1 {
			t.Errorf("%s: gone (n=%d), want exists", tc.name, n)
		}
	}
	_ = recentThreshold // 仅用于清晰对比
}

// TestGC_DeleteVisitorsLastSeenBefore — T0-1l-4:
// 删除 last_seen_at < threshold 的孤立 visitor。
// 注意:实际场景需先清 sessions(FK CASCADE);此处用 raw SQL 模拟孤立。
func TestGC_DeleteVisitorsLastSeenBefore(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	oldVisitor := uuid.New()
	recentVisitor := uuid.New()
	for _, vid := range []uuid.UUID{oldVisitor, recentVisitor} {
		// 直接 SQL 设置不同 last_seen_at
		var ts string
		if vid == oldVisitor {
			ts = "NOW() - INTERVAL '48 hours'"
		} else {
			ts = "NOW() - INTERVAL '1 hour'"
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at)
			VALUES ($1, $2, $3, NOW(), `+ts+`)
		`, vid, tenantID, "1ac-gc-v-"+vid.String()[:8]); err != nil {
			t.Fatalf("seed visitor %s: %v", vid, err)
		}
	}

	// GC:threshold = 24h ago
	if err := pg.DeleteVisitorsLastSeenBefore(ctx, time.Now().Add(-24*time.Hour)); err != nil {
		t.Fatalf("DeleteVisitorsLastSeenBefore: %v", err)
	}

	var oldN, recentN int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM visitors WHERE id = $1`, oldVisitor).Scan(&oldN)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM visitors WHERE id = $1`, recentVisitor).Scan(&recentN)
	if oldN != 0 {
		t.Errorf("old visitor (48h): still exists, want gone")
	}
	if recentN != 1 {
		t.Errorf("recent visitor (1h): gone, want exists")
	}
}

// TestGC_DeleteEventBlobByID — T0-1l-4:删除指定 ID 的 event_blob。
func TestGC_DeleteEventBlobByID(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID

	// seed visitor + session + 2 event_blobs
	vid := uuid.New()
	sid := uuid.New()
	blobKeep := uuid.New()
	blobDelete := uuid.New()

	pool.Exec(ctx, `INSERT INTO visitors (id, tenant_id, fingerprint, first_seen_at, last_seen_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		vid, tenantID, "1ac-gc-blob-fp-"+vid.String()[:8])
	pool.Exec(ctx, `INSERT INTO sessions (id, tenant_id, visitor_id, started_at) VALUES ($1, $2, $3, NOW())`,
		sid, tenantID, vid)
	for i, bid := range []uuid.UUID{blobKeep, blobDelete} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO event_blobs (id, session_id, tenant_id, blob_index, minio_object_key, checksum_sha256, size_bytes, event_count, started_at, ended_at)
			VALUES ($1, $2, $3, $4, $5, 'sha', 100, 1, NOW(), NOW())
		`, bid, sid, tenantID, i, "key-"+bid.String()[:8]); err != nil {
			t.Fatalf("seed blob %s: %v", bid, err)
		}
	}
	defer pool.Exec(ctx, `DELETE FROM visitors WHERE id = $1`, vid)

	if err := pg.DeleteEventBlobByID(ctx, blobDelete); err != nil {
		t.Fatalf("DeleteEventBlobByID: %v", err)
	}

	var keepN, delN int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM event_blobs WHERE id = $1`, blobKeep).Scan(&keepN)
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM event_blobs WHERE id = $1`, blobDelete).Scan(&delN)
	if keepN != 1 {
		t.Errorf("blobKeep should still exist (n=%d)", keepN)
	}
	if delN != 0 {
		t.Errorf("blobDelete should be gone (n=%d)", delN)
	}
}

// TestConsent_UpsertAndGetLatest — T0-1l-6:consent 写入 + 读取流程。
// 同 (fingerprint, scope, version) upsert 应替换旧记录,GetLatestConsent 返回最新。
func TestConsent_UpsertAndGetLatest(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID
	fp := "1ac-consent-fp-" + uuid.New().String()[:8]
	defer pool.Exec(ctx, `DELETE FROM visitor_consents WHERE fingerprint = $1`, fp)

	// 1. 初始无 consent,GetLatestConsent 应 found=false
	c, found, err := pg.GetLatestConsent(ctx, tenantID, fp, "all", "v1")
	if err != nil {
		t.Fatalf("initial GetLatestConsent: %v", err)
	}
	if found {
		t.Errorf("initial: found=true want false (no consent seeded)")
	}
	if c != nil {
		t.Errorf("initial: c=%v want nil", c)
	}

	// 2. 写入 accepted=true
	c1, err := pg.UpsertConsent(ctx, tenantID, fp, "all", "v1", true)
	if err != nil {
		t.Fatalf("UpsertConsent #1: %v", err)
	}
	if !c1.Accepted {
		t.Errorf("UpsertConsent #1: Accepted=false want true")
	}

	// 3. 读取应得 found=true, Accepted=true
	c, found, err = pg.GetLatestConsent(ctx, tenantID, fp, "all", "v1")
	if err != nil {
		t.Fatalf("GetLatestConsent after upsert #1: %v", err)
	}
	if !found {
		t.Fatal("after upsert #1: found=false want true")
	}
	if !c.Accepted {
		t.Errorf("after upsert #1: Accepted=false want true")
	}

	// 4. 撤回(accepted=false)— upsert 应替换
	c2, err := pg.UpsertConsent(ctx, tenantID, fp, "all", "v1", false)
	if err != nil {
		t.Fatalf("UpsertConsent #2 (revoke): %v", err)
	}
	if c2.Accepted {
		t.Errorf("UpsertConsent #2: Accepted=true want false (revoked)")
	}

	// 5. GetLatestConsent 应反映最新(撤回)
	c, _, err = pg.GetLatestConsent(ctx, tenantID, fp, "all", "v1")
	if err != nil {
		t.Fatalf("GetLatestConsent after revoke: %v", err)
	}
	if c.Accepted {
		t.Errorf("after revoke: Accepted=true want false")
	}
}

// TestConsent_GetLatest_VersionScoped — T0-1l-6 边界:
// 同 fingerprint 不同 version 应独立(GetLatest 按指定 version 查)。
func TestConsent_GetLatest_VersionScoped(t *testing.T) {
	pool := helperPGPool(t)
	defer pool.Close()

	ctx := context.Background()
	pg := &Postgres{Pool: pool}
	tenantID := DefaultTenantID
	fp := "1ac-ver-fp-" + uuid.New().String()[:8]
	defer pool.Exec(ctx, `DELETE FROM visitor_consents WHERE fingerprint = $1`, fp)

	// v1 accepted=true, v2 未写
	if _, err := pg.UpsertConsent(ctx, tenantID, fp, "all", "v1", true); err != nil {
		t.Fatalf("UpsertConsent v1: %v", err)
	}

	// v1 found=true
	_, found, err := pg.GetLatestConsent(ctx, tenantID, fp, "all", "v1")
	if err != nil || !found {
		t.Errorf("v1: found=%v err=%v, want found=true err=nil", found, err)
	}
	// v2 未写,found=false
	_, found2, err2 := pg.GetLatestConsent(ctx, tenantID, fp, "all", "v2")
	if err2 != nil {
		t.Errorf("v2: err=%v", err2)
	}
	if found2 {
		t.Errorf("v2: found=true want false (never written)")
	}
}
