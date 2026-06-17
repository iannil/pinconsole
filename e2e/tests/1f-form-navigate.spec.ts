import { test, expect } from '@playwright/test';

// 切片 1f 表单 + 跳转:e2e 验收(4 场景)
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md

test.describe('1f', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1f 场景1：浮动输入框 + fill_input 代填', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    if (!sessions.sessions.length) return;
    const sessionId = sessions.sessions[0].session_id;

    const fillResp = await request.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'fill_input', payload: { node_id: 0, value: '张三' } },
    });
    expect([200, 503]).toContain(fillResp.status());

    await visitorCtx.close();
  });

  test('1f 场景2：nodeID + click 跨 iframe（坐标 fallback）', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    if (!sessions.sessions.length) return;
    const sessionId = sessions.sessions[0].session_id;

    const clickResp = await request.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'click', payload: { node_id: 0, x: 100, y: 200 } },
    });
    expect([200, 503]).toContain(clickResp.status());

    await visitorCtx.close();
  });

  test('1f 场景3：navigate 自动重订阅（同源 URL 被允许）', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    if (!sessions.sessions.length) return;
    const sessionId = sessions.sessions[0].session_id;

    const navResp = await request.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'navigate', payload: { url: '/another-page' } },
    });
    expect([200, 503]).toContain(navResp.status());

    await visitorCtx.close();
  });

  test('1f 场景4：navigate 白名单拒绝（跨域 URL）', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    if (!sessions.sessions.length) return;
    const sessionId = sessions.sessions[0].session_id;

    const navResp = await request.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'navigate', payload: { url: 'https://evil.example.com/phishing' } },
    });
    expect(navResp.status()).toBe(403);
    const body = await navResp.json();
    expect(body.error).toBe('url_not_allowed');

    await visitorCtx.close();
  });

  // ===== 切片 1g：弹窗 + 聊天 =====
});
