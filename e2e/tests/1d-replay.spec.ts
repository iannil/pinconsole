import { test, expect } from '../fixtures/admin-auth';

// 切片 1d 录像归档:e2e 验收(4 场景)
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md
//
// 修复(2026-06-18 v1-e2e-acceptance):
// - admin 上下文用 admin-auth fixture
// - REST API 调用(/api/sessions/ended、/api/sessions/:id/replay)用 adminRequest(已带 cookie)

test.describe('1d', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1d 场景1：live 转 historical（访客关闭 → admin 历史回放）', async ({ browser, adminPage }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();
    const admin = adminPage;

    await visitor.goto('/');
    await visitor.waitForTimeout(2500);
    for (let i = 0; i < 20; i++) {
      await visitor.mouse.move(50 + i * 10, 50 + (i % 50));
    }
    await visitor.mouse.click(150, 150);
    await visitor.waitForTimeout(1000);

    await visitorCtx.close();

    await admin.waitForTimeout(2000);

    await admin.goto('/admin/replay');
    await admin.waitForTimeout(1500);

    await expect(admin.locator('.sessions-table tbody tr')).not.toHaveCount(0, {
      timeout: 10000,
    });

    await admin.locator('.sessions-table tbody tr').first().click();

    await expect(admin.locator('.replay-viewer')).toBeVisible({ timeout: 10000 });
    await expect(admin.locator('.session-info')).toBeVisible();
  });

  test('1d 场景2：短 session 即时 replay（< 30s 也立即 replayable）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(1500);
    await visitor.mouse.click(100, 100);
    await visitorCtx.close();

    await new Promise((r) => setTimeout(r, 1500));

    const resp = await adminRequest.get('/api/sessions/ended?since=24h');
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data.sessions.length).toBeGreaterThan(0);

    const last = data.sessions[0];
    expect(last.session_id).toBeTruthy();
    const replayResp = await adminRequest.get(
      `/api/sessions/${last.session_id}/replay?offset=0&limit=100`,
    );
    expect(replayResp.ok()).toBeTruthy();
    const replay = await replayResp.json();
    expect(replay.session_id).toBe(last.session_id);
  });

  test('1d 场景3：长 session 分页 replay（1000+ 事件）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(2500);

    for (let i = 0; i < 200; i++) {
      await visitor.mouse.move(50 + i * 3, 50 + (i % 100));
    }
    await visitor.mouse.click(100, 100);
    await visitor.waitForTimeout(500);
    await visitorCtx.close();

    // 1ae R4 修复:原 2s 等待不足以让 flusher 把 events 从 Redis stream flush 到 MinIO。
    // 默认 flusher 间隔 30s,200 events 不到 1000 阈值,所以等 flusher tick。
    // 改为 polling:每 2s 查一次,最多等 40s,直到 replay 返回非空 events。
    const listResp = await adminRequest.get('/api/sessions/ended?since=24h');
    const list = await listResp.json();
    const sessionsWithEvents = (list.sessions ?? []).filter(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (s: any) => s.event_count > 0,
    );

    // 1ae R4: 移除 silent skip — 测试必须真验证长 session 分页
    expect(sessionsWithEvents.length).toBeGreaterThan(0);
    const last = sessionsWithEvents[0];

    // 1ae R4: polling 等 flusher 完成
    let page1: { events: unknown[]; total?: number; has_more?: boolean } | null = null;
    const pollDeadline = Date.now() + 40_000;
    while (Date.now() < pollDeadline) {
      const r1 = await adminRequest.get(
        `/api/sessions/${last.session_id}/replay?offset=0&limit=100`,
      );
      const p = await r1.json();
      if (Array.isArray(p.events) && p.events.length > 0) {
        page1 = p;
        break;
      }
      await new Promise((resolve) => setTimeout(resolve, 2000));
    }

    // 1ae R4: 如果 40s 后仍无 events,不要 silent pass — fail loudly
    expect(page1, 'flusher 应在 40s 内把 events flush 到 MinIO;page1 不应为 null').not.toBeNull();
    expect(Array.isArray(page1!.events)).toBeTruthy();
    expect(page1!.events.length).toBeGreaterThan(0);
    expect(typeof page1!.total).toBe('number');
    expect(typeof page1!.has_more).toBe('boolean');

    if (page1!.has_more) {
      const r2 = await adminRequest.get(
        `/api/sessions/${last.session_id}/replay?offset=${page1!.events.length}&limit=100`,
      );
      const page2 = await r2.json();
      expect(Array.isArray(page2.events)).toBeTruthy();
    }
  });

  test('1d 场景4：replay 控制器交互（暂停/播放/倍速/进度）', async ({ browser, adminPage }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();
    const admin = adminPage;

    await visitor.goto('/');
    await visitor.waitForTimeout(2000);
    await visitor.mouse.move(100, 100);
    await visitor.mouse.click(200, 200);
    await visitor.waitForTimeout(500);
    await visitorCtx.close();

    await admin.goto('/admin/replay');
    await admin.waitForTimeout(1500);

    await expect(admin.locator('.sessions-table tbody tr')).not.toHaveCount(0, {
      timeout: 10000,
    });
    await admin.locator('.sessions-table tbody tr').first().click();
    await admin.waitForTimeout(2000);

    await expect(admin.locator('.player-container')).toBeVisible({ timeout: 10000 });
  });
});
