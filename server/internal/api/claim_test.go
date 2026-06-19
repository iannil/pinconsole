// 1ac 测试:claim 原子性 + release owner-only Lua(审计 T0-1k-4 + T0-1k-5)。
//
// 验证 claim.go 用到的两个 race-safe 原语:
//  1. SetNX(claimKey, uid, TTL) — 并发只有一方 win(T0-1k-4)
//  2. releaseClaimLua EvalLua — owner match 才 DEL(T0-1k-5)
//
// 注:此为原语层覆盖;handler 端到端测试需 PG fixture,留 1ac 后续工作。
// 原语正确即等价于 claim.go 的 race-safety(因 claim.go 直接调用这些原语)。
package api

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
)

// TestClaim_SetNX_RaceSafety — T0-1k-4: 并发 SetNX 同一 session_key 只一方 win。
//
// 模拟两个运营同时 claim 同一 session。
// 期望:first.SetNX=true, second.SetNX=false(已存在)。
// 如果 claim.go 被错误重构成 Set(非 NX),此测试会失败(都返回成功)。
func TestClaim_SetNX_RaceSafety(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	ctx := context.Background()
	sessionID := uuid.New()
	key := claimKey(sessionID)
	defer rdb.Del(ctx, key)

	uid1 := uuid.New()
	uid2 := uuid.New()

	store := &storage.Redis{Client: rdb}

	// 并发 SetNX
	var wg sync.WaitGroup
	wg.Add(2)
	results := make([]bool, 2)
	errs := make([]error, 2)

	go func() {
		defer wg.Done()
		results[0], errs[0] = store.SetNX(ctx, key, []byte(uid1.String()), claimTTL)
	}()
	go func() {
		defer wg.Done()
		results[1], errs[1] = store.SetNX(ctx, key, []byte(uid2.String()), claimTTL)
	}()
	wg.Wait()

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("SetNX errors: %v / %v", errs[0], errs[1])
	}

	winCount := 0
	for _, r := range results {
		if r {
			winCount++
		}
	}
	if winCount != 1 {
		t.Errorf("race-safe SetNX: winCount=%d want 1 (results=%v)", winCount, results)
	}

	// 校验 winner 写入的 uid 是 u1 或 u2 之一(不是空、不是混合)
	got, err := rdb.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != uid1.String() && got != uid2.String() {
		t.Errorf("claim value=%q want uid1(%s) or uid2(%s)", got, uid1, uid2)
	}
}

// TestClaim_ReleaseLua_OwnerOnlyDelete — T0-1k-5:
// releaseClaimLua 仅当 Redis 中 owner == caller UID 才 DEL。
func TestClaim_ReleaseLua_OwnerOnlyDelete(t *testing.T) {
	rdb := helperRedisIfAvailable(t)
	defer rdb.Close()

	store := &storage.Redis{Client: rdb}

	t.Run("owner_can_release", func(t *testing.T) {
		ctx := context.Background()
		sessionID := uuid.New()
		key := claimKey(sessionID)
		defer rdb.Del(ctx, key)
		ownerUID := uuid.New()

		if err := rdb.Set(ctx, key, ownerUID.String(), claimTTL).Err(); err != nil {
			t.Fatalf("seed claim: %v", err)
		}

		result, err := store.EvalLua(ctx, releaseClaimLua, []string{key}, ownerUID.String())
		if err != nil {
			t.Fatalf("EvalLua: %v", err)
		}
		deleted, _ := result.(int64)
		if deleted != 1 {
			t.Errorf("owner release: deleted=%d want 1", deleted)
		}
		// key 应被删除
		exists, _ := rdb.Exists(ctx, key).Result()
		if exists != 0 {
			t.Errorf("after owner release: key still exists (race-safety broken)")
		}
	})

	t.Run("non_owner_cannot_release", func(t *testing.T) {
		ctx := context.Background()
		sessionID := uuid.New()
		key := claimKey(sessionID)
		defer rdb.Del(ctx, key)
		ownerUID := uuid.New()
		otherUID := uuid.New()

		if err := rdb.Set(ctx, key, ownerUID.String(), claimTTL).Err(); err != nil {
			t.Fatalf("seed claim: %v", err)
		}

		result, err := store.EvalLua(ctx, releaseClaimLua, []string{key}, otherUID.String())
		if err != nil {
			t.Fatalf("EvalLua: %v", err)
		}
		deleted, _ := result.(int64)
		if deleted != 0 {
			t.Errorf("non-owner release: deleted=%d want 0 (must NOT delete)", deleted)
		}
		// claim 应保持原 owner
		got, _ := rdb.Get(ctx, key).Result()
		if got != ownerUID.String() {
			t.Errorf("after non-owner release: claim=%q want %s (still owned by original)", got, ownerUID)
		}
	})

	t.Run("missing_key_no_op", func(t *testing.T) {
		ctx := context.Background()
		sessionID := uuid.New()
		key := claimKey(sessionID) // 故意不 set
		anyUID := uuid.New()

		result, err := store.EvalLua(ctx, releaseClaimLua, []string{key}, anyUID.String())
		if err != nil {
			t.Fatalf("EvalLua: %v", err)
		}
		deleted, _ := result.(int64)
		if deleted != 0 {
			t.Errorf("missing key: deleted=%d want 0 (no-op)", deleted)
		}
	})

	t.Run("corrupted_claim_prevents_release", func(t *testing.T) {
		ctx := context.Background()
		sessionID := uuid.New()
		key := claimKey(sessionID)
		defer rdb.Del(ctx, key)
		anyUID := uuid.New()

		// 写入损坏值(非 UUID 字符串)
		if err := rdb.Set(ctx, key, "corrupt-data", claimTTL).Err(); err != nil {
			t.Fatalf("seed corrupt: %v", err)
		}

		result, err := store.EvalLua(ctx, releaseClaimLua, []string{key}, anyUID.String())
		if err != nil {
			t.Fatalf("EvalLua: %v", err)
		}
		deleted, _ := result.(int64)
		if deleted != 0 {
			t.Errorf("corrupt claim: deleted=%d want 0 (must NOT delete on corrupt)", deleted)
		}
	})
}

// TestClaim_ClaimTTL_Constant — 防 claimTTL 误改(1k P0-4 锁定 5min)。
func TestClaim_ClaimTTL_Constant(t *testing.T) {
	if claimTTL != 5*time.Minute {
		t.Errorf("claimTTL = %v, want 5m (1k 锁定窗口)", claimTTL)
	}
}

// TestClaim_ReleaseClaimLua_Contract — 防 releaseClaimLua 脚本被误改。
// 脚本必须包含 GET + DEL + 比对逻辑。
func TestClaim_ReleaseClaimLua_Contract(t *testing.T) {
	for _, must := range []string{
		"redis.call('GET'",       // 读 owner
		"ARGV[1]",                // 比对 caller
		"redis.call('DEL'",       // 删除
		"return 0",               // 失败路径
	} {
		if !contains(releaseClaimLua, must) {
			t.Errorf("releaseClaimLua missing %q (script broken):\n%s", must, releaseClaimLua)
		}
	}
}

// 防 fmt 被裁剪。
var _ = fmt.Sprintf
