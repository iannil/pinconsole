import { test, expect } from '@playwright/test';

// 切片 1h 认证 + 多运营:e2e 验收(4 场景)
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md

test.describe('1h', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1h 场景1：登录流端到端', async ({ request }) => {
    // 登录
    const loginResp = await request.post('/api/auth/login', {
      data: { email: 'admin@marketing-monitor.local', password: 'changeme123' },
    });
    expect(loginResp.ok()).toBeTruthy();
    const user = await loginResp.json();
    expect(user.email).toBe('admin@marketing-monitor.local');
    expect(user.role).toBe('admin');
  });

  test('1h 场景2：Claim/Release 锁定', async ({ request, browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();
    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await request.get('/api/sessions');
    const sessions = await sessionsResp.json();
    if (!sessions.sessions.length) return;
    const sessionId = sessions.sessions[0].session_id;

    // Claim
    const claimResp = await request.post(`/api/sessions/${sessionId}/claim`);
    expect(claimResp.ok()).toBeTruthy();

    // 验证被 claim
    const getClaimResp = await request.get(`/api/sessions/${sessionId}/claim`);
    const claimState = await getClaimResp.json();
    expect(claimState.claimed).toBe(true);

    // Release
    const releaseResp = await request.post(`/api/sessions/${sessionId}/release`);
    expect(releaseResp.ok()).toBeTruthy();

    // 验证已释放
    const getClaim2Resp = await request.get(`/api/sessions/${sessionId}/claim`);
    const claimState2 = await getClaim2Resp.json();
    expect(claimState2.claimed).toBe(false);

    await visitorCtx.close();
  });

  test('1h 场景3：访客端不受认证影响', async ({ browser }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    const logs: string[] = [];
    visitor.on('console', (m) => logs.push(m.text()));
    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    // SDK 应正常连接（不受认证影响）
    expect(logs.join('\n')).toContain('marketing-monitor');
    await visitorCtx.close();
  });

  test('1h 场景4：登出流', async ({ request }) => {
    const logoutResp = await request.post('/api/auth/logout');
    expect(logoutResp.ok()).toBeTruthy();
  });

  // ===== 切片 1i：反爬虫 =====
});
