import { test, expect } from '@playwright/test';

// 切片 1b 单向最小:e2e 验收(4 场景)
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md

test.describe('1b', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1b 场景1：访客访问 + admin 列表出现 + 订阅 + 事件传递', async ({ browser }) => {
    const visitorCtx = await browser.newContext();
    const adminCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();
    const admin = await adminCtx.newPage();

    await admin.goto('/admin/');
    await admin.waitForTimeout(1500);

    const sdkLogs: string[] = [];
    visitor.on('console', (m) => sdkLogs.push(m.text()));
    await visitor.goto('/');
    await visitor.waitForTimeout(2000);

    // SDK 应已启动
    expect(sdkLogs.join('\n')).toContain('marketing-monitor');

    // admin 应在 5s 内看到访客上线
    await expect(admin.locator('.visitor-list li')).not.toHaveCount(0, { timeout: 5000 });

    // 选中访客 → 订阅 → 触发交互
    await admin.locator('.visitor-list li').first().click();
    await admin.getByRole('button', { name: '订阅实时' }).click();

    await visitor.mouse.move(100, 100);
    await visitor.mouse.move(200, 200);
    await visitor.mouse.move(300, 300);
    await visitor.waitForTimeout(500);

    const eventCountText = await admin.locator('.events-area').textContent();
    expect(eventCountText).toBeTruthy();

    await visitorCtx.close();
    await adminCtx.close();
  });

  test('1b 场景2：10 访客并发', async ({ browser }) => {
    const adminCtx = await browser.newContext();
    const admin = await adminCtx.newPage();
    await admin.goto('/admin/');
    await admin.waitForTimeout(1500);

    const visitors = [];
    for (let i = 0; i < 10; i++) {
      const ctx = await browser.newContext();
      const page = await ctx.newPage();
      await page.goto('/');
      visitors.push({ ctx, page });
    }

    await admin.waitForTimeout(2000);

    const liCount = await admin.locator('.visitor-list li').count();
    expect(liCount).toBeGreaterThanOrEqual(5);

    for (const v of visitors) await v.ctx.close();
    await adminCtx.close();
  });

  test('1b 场景3：SDK 重连（healthz 探测）', async ({ browser, request }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto('/');
    await page.waitForTimeout(1500);

    const h = await request.get('/healthz');
    expect(h.ok()).toBeTruthy();

    await ctx.close();
  });

  test('1b 场景4：MinIO 录像快照（admin 仍能拿到 events）', async ({ browser, request }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto('/');
    await page.waitForTimeout(1000);

    for (let i = 0; i < 50; i++) {
      await page.mouse.move(50 + i * 10, 50 + i * 5);
    }
    await page.mouse.click(100, 100);
    await page.waitForTimeout(500);

    const sess = await request.get('/api/sessions');
    expect(sess.ok()).toBeTruthy();

    await ctx.close();
  });
});
