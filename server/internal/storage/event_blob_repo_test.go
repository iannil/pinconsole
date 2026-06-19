// 1ai-b 测试:event_blobs 表 repo PG 集成测试。
//
// 补 1d 录像归档依赖的 CreateEventBlob / ListBySession / ListOlderThan。
// DeleteEventBlobByID 已被 1ad 覆盖(100%),1ai-b 加深 + 补前 3 个函数。
package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// makeEventBlob 构造一个 EventBlob(seed 数据用)。
func makeEventBlob(sessionID uuid.UUID, idx int32) EventBlob {
	now := time.Now()
	return EventBlob{
		SessionID:      sessionID,
		TenantID:       DefaultTenantID,
		BlobIndex:      idx,
		StartedAt:      now,
		EndedAt:        now.Add(time.Second),
		EventCount:     int32(idx * 10),
		MinIOObjectKey: "test/1aib/blob-" + uuid.New().String()[:8],
		SizeBytes:      int64(idx * 1024),
		ChecksumSHA256: "sha256-1aib-" + uuid.New().String()[:8],
	}
}

// TestCreateEventBlob_AndList — Create + ListBySession 字段一致。
func TestCreateEventBlob_AndList(t *testing.T) {
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

	in := makeEventBlob(session.ID, 0)
	out, err := pg.CreateEventBlob(ctx, in)
	if err != nil {
		t.Fatalf("CreateEventBlob: %v", err)
	}
	if out == nil || out.ID == uuid.Nil {
		t.Fatal("CreateEventBlob returned nil/zero-ID")
	}
	if out.MinIOObjectKey != in.MinIOObjectKey {
		t.Errorf("MinIOObjectKey = %q, want %q", out.MinIOObjectKey, in.MinIOObjectKey)
	}
	if out.SizeBytes != in.SizeBytes {
		t.Errorf("SizeBytes = %d, want %d", out.SizeBytes, in.SizeBytes)
	}

	list, err := pg.ListEventBlobsBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListEventBlobsBySession: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
	if list[0].ID != out.ID {
		t.Errorf("list[0].ID = %s, want %s", list[0].ID, out.ID)
	}
}

// TestListEventBlobsBySession_OrderedByBlobIndex — 多 blob 按 blob_index 正序。
func TestListEventBlobsBySession_OrderedByBlobIndex(t *testing.T) {
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

	// 乱序创建(2, 0, 1),期望 List 按 0, 1, 2 排序
	for _, idx := range []int32{2, 0, 1} {
		if _, err := pg.CreateEventBlob(ctx, makeEventBlob(session.ID, idx)); err != nil {
			t.Fatalf("CreateEventBlob(%d): %v", idx, err)
		}
	}

	list, err := pg.ListEventBlobsBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("list len = %d, want 3", len(list))
	}
	for i, want := range []int32{0, 1, 2} {
		if list[i].BlobIndex != want {
			t.Errorf("list[%d].BlobIndex = %d, want %d (应按 blob_index 正序)", i, list[i].BlobIndex, want)
		}
	}
}

// TestListEventBlobsOlderThan_FiltersAndLimits — threshold 过滤 + LIMIT 生效。
//
// ListEventBlobsOlderThan WHERE created_at < threshold,返"早于 threshold"的 blob(用于 GC)。
func TestListEventBlobsOlderThan_FiltersAndLimits(t *testing.T) {
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

	// 顺序创建 4 个 blob,在第 1 个之后取 threshold
	// blobs 0/1 早于 threshold(应被列),2/3 晚于(不应被列)
	var oldIDs []uuid.UUID
	for i := 0; i < 2; i++ {
		b, err := pg.CreateEventBlob(ctx, makeEventBlob(session.ID, int32(i)))
		if err != nil {
			t.Fatalf("CreateEventBlob old %d: %v", i, err)
		}
		oldIDs = append(oldIDs, b.ID)
	}
	time.Sleep(10 * time.Millisecond)
	threshold := time.Now()
	time.Sleep(10 * time.Millisecond)
	var newIDs []uuid.UUID
	for i := 2; i < 4; i++ {
		b, err := pg.CreateEventBlob(ctx, makeEventBlob(session.ID, int32(i)))
		if err != nil {
			t.Fatalf("CreateEventBlob new %d: %v", i, err)
		}
		newIDs = append(newIDs, b.ID)
	}

	// 列 threshold 之前的(应 2 个,都是 oldIDs)
	list, err := pg.ListEventBlobsOlderThan(ctx, threshold, 100)
	if err != nil {
		t.Fatalf("ListEventBlobsOlderThan: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2 (2 个 old blob)", len(list))
	}
	listedIDs := map[uuid.UUID]bool{}
	for _, b := range list {
		listedIDs[b.ID] = true
	}
	for _, oid := range oldIDs {
		if !listedIDs[oid] {
			t.Errorf("old blob %s 应在 threshold 前列表", oid)
		}
	}
	for _, nid := range newIDs {
		if listedIDs[nid] {
			t.Errorf("new blob %s 不应在 threshold 前列表(过滤反向)", nid)
		}
	}

	// LIMIT 生效:limit=1 应只返 1 条
	listLimited, err := pg.ListEventBlobsOlderThan(ctx, threshold, 1)
	if err != nil {
		t.Fatalf("ListEventBlobsOlderThan(limit=1): %v", err)
	}
	if len(listLimited) != 1 {
		t.Errorf("limit=1 应只返 1 条, got %d", len(listLimited))
	}
}

// TestDeleteEventBlobByID_ExistingAndMissing — 已存在 + 不存在 ID 都不报错。
//
// DeleteEventBlobByID 已被 1ad 覆盖(100%),本测试加深幂等性。
func TestDeleteEventBlobByID_ExistingAndMissing(t *testing.T) {
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

	blob, err := pg.CreateEventBlob(ctx, makeEventBlob(session.ID, 0))
	if err != nil {
		t.Fatalf("CreateEventBlob: %v", err)
	}

	// 存在的 ID 删除应成功
	if err := pg.DeleteEventBlobByID(ctx, blob.ID); err != nil {
		t.Errorf("DeleteEventBlobByID(existing): %v", err)
	}

	// 验证已删
	list, _ := pg.ListEventBlobsBySession(ctx, session.ID)
	for _, b := range list {
		if b.ID == blob.ID {
			t.Error("blob should be deleted")
		}
	}

	// 不存在的 ID 删除应无 error(PG DELETE 无匹配行不报错)
	if err := pg.DeleteEventBlobByID(ctx, uuid.New()); err != nil {
		t.Errorf("DeleteEventBlobByID(missing): err = %v, want nil (PG DELETE 无匹配行不报错)", err)
	}
}
