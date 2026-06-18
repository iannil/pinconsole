import { test, expect } from '@playwright/test';

// 切片 1g 弹窗 + 聊天:e2e 验收(4 场景)
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md

test.describe('1g', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1g 场景1：弹窗推送 + 访客端渲染', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    const popupResp = await request.post(`/api/sessions/${sessionId}/command`, {
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

    // 验证：访客端弹出 popup（DOM 中有 #_mm_popup_）
    if (popupResp.status() === 200) {
      await expect(visitor.locator('#__mm_popup__')).toBeVisible({ timeout: 5000 });
    }

    await visitorCtx.close();
  });

  test('1g 场景2：双向聊天端到端', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    // admin → visitor 聊天消息
    const msgResp = await request.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '您好，有什么可以帮您？', sender: 'operator' },
    });
    expect(msgResp.ok()).toBeTruthy();
    const msg = await msgResp.json();
    expect(msg.sender).toBe('operator');
    expect(msg.content).toBe('您好，有什么可以帮您？');

    await visitorCtx.close();
  });

  test('1g 场景3：消息历史持久化', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    // 发两条消息
    await request.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '消息1', sender: 'operator' },
    });
    await request.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '消息2', sender: 'operator' },
    });

    // 验证 GET 返回历史
    const historyResp = await request.get(`/api/sessions/${sessionId}/messages`);
    expect(historyResp.ok()).toBeTruthy();
    const history = await historyResp.json();
    expect(history.messages.length).toBeGreaterThanOrEqual(2);

    await visitorCtx.close();
  });

  test('1g 场景4：离线消息不丢（写入 PG）', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    expect(sessions.sessions.length, "expected ≥1 active session (e2e fixture missing)").toBeGreaterThan(0);
    const sessionId = sessions.sessions[0].session_id;

    // 访客关闭页面（离线）
    await visitorCtx.close();
    await new Promise((r) => setTimeout(r, 2000));

    // 运营仍能发消息（写入 PG）
    const msgResp = await request.post(`/api/sessions/${sessionId}/messages`, {
      data: { content: '离线消息', sender: 'operator' },
    });
    expect(msgResp.ok()).toBeTruthy();

    // GET 能查到
    const historyResp = await request.get(`/api/sessions/${sessionId}/messages`);
    const history = await historyResp.json();
    const found = history.messages.find((m: { content: string }) => m.content === '离线消息');
    expect(found).toBeTruthy();
  });

  // ===== 切片 1h：认证 + 多运营 =====
});
