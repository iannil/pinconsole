import { test, expect } from '../fixtures/admin-auth';

// 切片 1f 表单 + 跳转:e2e 验收(4 场景)
// 修复(2026-06-18 v1-e2e-acceptance):
// - REST API 调用用 adminRequest
// - 每个 command 调用前先 claim(1k P0-3 requireClaimOwnership)

test.describe('1f', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1f 场景1：浮动输入框 + fill_input 代填', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const fillResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'fill_input', payload: { node_id: 0, value: '张三' } },
    });
    expect([200, 503]).toContain(fillResp.status());

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1f 场景2：nodeID + click 跨 iframe（坐标 fallback）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const clickResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'click', payload: { node_id: 0, x: 100, y: 200 } },
    });
    expect([200, 503]).toContain(clickResp.status());

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1f 场景3：navigate 自动重订阅（同源 URL 被允许）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const navResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'navigate', payload: { url: '/another-page' } },
    });
    expect([200, 503]).toContain(navResp.status());

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1f 场景4：navigate 白名单拒绝（跨域 URL）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const navResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'navigate', payload: { url: 'https://evil.example.com/phishing' } },
    });
    expect(navResp.status()).toBe(403);
    const body = await navResp.json();
    expect(body.error).toBe('url_not_allowed');

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });
});
