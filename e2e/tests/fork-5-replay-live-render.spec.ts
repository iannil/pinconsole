// fork-5: ReplayPlayer live mode verification (post fork-2 钻穿 + live mode fix).
//
// 验证:
// 1. admin 选中在线访客后 ReplayPlayer 挂载并正确渲染 iframe
// 2. iframe body 非空（snapshot 已重建 DOM）
// 3. 非 loading/error 状态
//
// 前置条件: ops.sh start 已启动服务,docker infra 就绪

import { test, expect } from '../fixtures/admin-auth';

test.describe('fork-5 replay live render', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(120_000);
  });

  test('replay-player renders iframe with content for live visitor', async ({ browser, adminPage: admin }) => {
    // 1. 创建访客页面,产生 rrweb 事件
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    // 等待 SDK 初始化 + 首次收集（full snapshot + 增量）
    // opt-in 模式下需要先点击同意横幅
    await visitor.waitForTimeout(1500);
    const acceptBtn = visitor.locator('[data-pinconsole="consent-card"] button').last();
    if (await acceptBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await acceptBtn.click();
    }
    await visitor.waitForTimeout(2000);

    // 2. admin 等访客出现在列表中
    await expect(admin.locator('.visitor-list li:not(.empty)')).not.toHaveCount(0, {
      timeout: 15_000,
    });

    // 3. admin 选中该访客
    await admin.locator('.visitor-list li:not(.empty)').first().click();

    // 4. 访客产生更多事件（触发 rebuildPlayer 下一次 watch 时拿到 ≥2 events）
    for (let i = 0; i < 10; i++) {
      await visitor.mouse.move(100 + i * 50, 200 + i * 10);
      await visitor.waitForTimeout(200);
    }
    await visitor.mouse.click(300, 200);
    await visitor.waitForTimeout(3000);

    // 5. 验证 ReplayPlayer 挂载
    await expect(admin.locator('.replay-player')).toBeVisible({ timeout: 10_000 });

    // 6. 检查 replay-player 的状态：不应是 error
    await expect(admin.locator('.replay-player .error')).toHaveCount(0, { timeout: 3000 });

    // 7. 等待 player-container 内有 iframe（Replayer 创建需要 ≥2 rrweb events）
    const container = admin.locator('.player-container');
    await expect(container).toBeVisible({ timeout: 5000 });

    const iframe = container.locator('iframe').first();
    await expect(iframe).toBeAttached({ timeout: 20_000 });
    await expect(iframe).toBeVisible({ timeout: 5000 });

    // 8. 验证 iframe body 非空（snapshot 已重建真实 DOM）
    const frameContent = iframe.contentFrame();
    await expect(frameContent.locator('body')).toBeAttached({ timeout: 10_000 });
    const bodyHTML = await frameContent.locator('body').innerHTML({ timeout: 5000 });
    expect(bodyHTML.length).toBeGreaterThan(50);

    await visitorCtx.close();
  });
});
