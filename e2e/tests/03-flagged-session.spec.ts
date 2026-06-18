// 切片 1w:flagged session regression + smoke
//
// 1w P1-29:server 在 /api/sessions 响应中加入 is_flagged + flag_reason 字段,
// 数据源是 Redis key=flagged:session:<id>(TTL=10min)。
// FlagSession 由 antiscrape BehaviorTracker 触发(50+ 事件且零鼠标移动等可疑模式)。
//
// regression:验证 /api/sessions 返回的 session 对象含 is_flagged 字段
//
// smoke:用 page.evaluate 直接构造一个 flagged session — 创建 visitor session,
// 然后直连 Redis 设 flagged:session:<id>,验证 /api/sessions 返回该 session 时
// is_flagged=true。这种"半真半假"测试避开了 BehaviorTracker 触发条件复杂的问题,
// 又验证了 server 真的会读 Redis 标记并放入响应。

import { test, expect } from '../fixtures/admin-auth';
import { closeDBFixtures } from '../fixtures/db';
import { createClient } from 'redis';

async function setSessionFlag(sessionId: string, reason: string): Promise<void> {
  const r = createClient({ url: 'redis://localhost:6379' });
  await r.connect();
  await r.set(`flagged:session:${sessionId}`, reason, { EX: 600 });
  await r.quit();
}

async function clearSessionFlag(sessionId: string): Promise<void> {
  const r = createClient({ url: 'redis://localhost:6379' });
  await r.connect();
  await r.del(`flagged:session:${sessionId}`);
  await r.quit();
}

test.describe('1w: flagged session', () => {
  test.afterAll(async () => {
    await closeDBFixtures();
  });

  test('regression:/api/sessions 返回的 session 含 is_flagged 字段', async ({ adminRequest, browser }) => {
    // 启一个普通 visitor session
    const ctx = await browser.newContext();
    const visitor = await ctx.newPage();
    await visitor.goto('/');
    await visitor.waitForTimeout(2500);

    try {
      const resp = await adminRequest.get('/api/sessions');
      expect(resp.ok()).toBeTruthy();
      const data = await resp.json();
      expect(data.sessions.length, 'should have ≥1 active session').toBeGreaterThan(0);

      // 1w P1-29:每个 session 对象应有 is_flagged 字段(bool)
      const first = data.sessions[0];
      expect(first, 'session object must exist').toBeTruthy();
      expect(typeof first.is_flagged, 'is_flagged must be boolean').toBe('boolean');
      // 默认(未被 flag)应为 false
      expect(first.is_flagged, 'fresh session should not be flagged').toBe(false);
    } finally {
      await ctx.close();
    }
  });

  test('smoke:Redis flag 设置后,/api/sessions 返回 is_flagged=true', async ({ adminRequest, browser }) => {
    const ctx = await browser.newContext();
    const visitor = await ctx.newPage();
    await visitor.goto('/');
    await visitor.waitForTimeout(2500);

    try {
      // 拿到当前活跃 session ID
      const before = await adminRequest.get('/api/sessions');
      const beforeData = await before.json();
      const target = beforeData.sessions.find((s: { is_flagged?: boolean }) => !s.is_flagged);
      expect(target, 'should have an unflagged session to test with').toBeTruthy();
      const sessionId = target.session_id;
      expect(sessionId, 'session_id should be present').toBeTruthy();

      // 直连 Redis 设 flag(模拟 BehaviorTracker 触发)
      await setSessionFlag(sessionId, 'e2e-1w-smoke');

      // 重新拉取,验证 is_flagged=true + reason 正确
      const after = await adminRequest.get('/api/sessions');
      const afterData = await after.json();
      const updated = afterData.sessions.find((s: { session_id: string }) => s.session_id === sessionId);
      expect(updated, `session ${sessionId} should still appear after flag`).toBeTruthy();
      expect(updated.is_flagged, 'is_flagged must be true after setting Redis flag').toBe(true);
      expect(updated.flag_reason, 'flag_reason must be exposed').toBe('e2e-1w-smoke');

      // cleanup
      await clearSessionFlag(sessionId);
    } finally {
      await ctx.close();
    }
  });
});
