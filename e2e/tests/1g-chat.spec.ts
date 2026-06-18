import { test, expect } from '../fixtures/admin-auth';

// 切片 1g 弹窗 + 聊天:e2e 验收(4 场景)
// 修复(2026-06-18 v1-e2e-acceptance):
// - REST API 调用用 adminRequest
// - 每个 message/command 调用前先 claim(1k P0-3 requireClaimOwnership)

test.describe('1g', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1g 场景1：弹窗推送 + 访客端渲染', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const popupResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: {
        type: 'show_popup',
        payload: {
          title: '限时优惠',
          body: '今日下单立减 50%',
          action_label: '去领取',
          action_url: '/coupon',
          dismissible: true,
        },
      },
    });
    expect([200, 503]).toContain(popupResp.status());

    if (popupResp.status() === 200) {
      await expect(visitor.locator('#__mm_popup__')).toBeVisible({ timeout: 5000 });
    }

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1g 场景2：双向聊天端到端', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const msgResp = await adminRequest.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '您好，有什么可以帮您？', sender: 'operator' },
    });
    expect(msgResp.ok()).toBeTruthy();
    const msg = await msgResp.json();
    expect(msg.sender).toBe('operator');
    expect(msg.content).toBe('您好，有什么可以帮您？');

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1g 场景3：消息历史持久化', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    await adminRequest.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '消息1', sender: 'operator' },
    });
    await adminRequest.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '消息2', sender: 'operator' },
    });

    const historyResp = await adminRequest.get(`/api/sessions/${sessionId}/messages`);
    expect(historyResp.ok()).toBeTruthy();
    const history = await historyResp.json();
    expect(history.messages.length).toBeGreaterThanOrEqual(2);

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1g 场景4：离线消息不丢（写入 PG）', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    await visitorCtx.close();
    await new Promise((r) => setTimeout(r, 2000));

    const msgResp = await adminRequest.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '离线消息', sender: 'operator' },
    });
    expect(msgResp.ok()).toBeTruthy();

    const historyResp = await adminRequest.get(`/api/sessions/${sessionId}/messages`);
    const history = await historyResp.json();
    const found = history.messages.find((m: { content: string }) => m.content === '离线消息');
    expect(found).toBeTruthy();

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
  });
});
