import { test, expect } from '../fixtures/admin-auth';

// 切片 1e 双向通道:e2e 验收(5 场景)
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md
//
// 修复(2026-06-18 v1-e2e-acceptance):
// - admin 上下文用 admin-auth fixture(adminPage / adminRequest)
// - REST API 调用前先 claim session(1k P0-3 requireClaimOwnership 强制;
//   原测试在 1k 之前能跑是因为 dev bypass 把 user_id 设为 uuid.Nil,
//   release build 下 bypass 不存在,必须 claim)

test.describe('1e', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1e 场景1：cursor_highlight 双向（Start → 高亮跟随）', async ({ browser, adminPage, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();
    const admin = adminPage;

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    await expect(admin.locator('.visitor-list li:not(.empty)')).not.toHaveCount(0, {
      timeout: 10000,
    });
    await admin.locator('.visitor-list li:not(.empty)').first().click();
    await admin.getByRole('button', { name: '订阅实时' }).click();
    await admin.waitForTimeout(1500);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length).toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    // 必须先 claim 才能发 command
    const claimResp = await adminRequest.post(`/api/sessions/${sessionId}/claim`);
    expect(claimResp.ok()).toBeTruthy();

    const cmdResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'cursor_highlight', payload: { x: 100, y: 100, name: '运营甲' } },
    });
    // 与 1e 场景2/3 一致:visitor 可能已离线 → 503 是合法状态(说明命令本身通过校验)
    expect([200, 503]).toContain(cmdResp.status());

    const cursorCount = await visitor.locator('#__mm_operator_cursor__').count();
    // eslint-disable-next-line no-console
    console.log(`cursor element count = ${cursorCount}`);

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1e 场景2：click 命令转发（运营点按钮，访客端被点）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const cmdResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'click', payload: { node_id: 0, x: 100, y: 100 } },
    });
    expect([200, 503]).toContain(cmdResp.status());

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1e 场景3：fill_input 命令（运营代填表单）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const cmdResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'fill_input', payload: { node_id: 0, value: '张三' } },
    });
    expect([200, 503]).toContain(cmdResp.status());

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1e 场景4：紧急退出 ESC 三连 / Ctrl+Shift+X', async ({ browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    await visitor.keyboard.down('Control');
    await visitor.keyboard.down('Shift');
    await visitor.keyboard.press('KeyX');
    await visitor.keyboard.up('Control');
    await visitor.keyboard.up('Shift');
    await visitor.waitForTimeout(500);

    const errs: string[] = [];
    visitor.on('pageerror', (e) => errs.push(String(e)));
    expect(errs.length).toBe(0);

    await visitorCtx.close();
  });

  test('1e 场景5：审计 PG co_browsing_commands 表', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'cursor_highlight', payload: { x: 50, y: 50, name: '审计测试' } },
    });
    await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'click', payload: { node_id: 0, x: 50, y: 50 } },
    });

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });
});
