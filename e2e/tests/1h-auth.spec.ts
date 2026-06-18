import { test, expect, ADMIN_EMAIL, ADMIN_PASSWORD } from '../fixtures/admin-auth';

// 切片 1h 认证 + 多运营:e2e 验收(4 场景)
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md
//
// 修复(2026-06-18 v1-e2e-acceptance):
// - 凭证用 ADMIN_EMAIL/ADMIN_PASSWORD(从根 .env 读,与 server 启动同源)
// - REST API 调用用 adminRequest(已带 cookie)
// - SDK 日志断言 source 字段(1r 切片换 JSON logger 后没有 marketing-monitor 字面量)

test.describe('1h', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });
  test('1h 场景1：登录流端到端', async ({ request }) => {
    const loginResp = await request.post('/api/auth/login', {
      data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
    });
    expect(loginResp.ok()).toBeTruthy();
    const user = await loginResp.json();
    expect(user.email).toBe(ADMIN_EMAIL);
    expect(user.role).toBe('admin');
  });

  test('1h 场景2：Claim/Release 锁定', async ({ browser, adminRequest }) => {
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();
    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    if (!sessions.sessions.length) {
      await visitorCtx.close();
      return;
    }
    const sessionId = sessions.sessions[0].session_id;

    const claimResp = await adminRequest.post(`/api/sessions/${sessionId}/claim`);
    expect(claimResp.ok()).toBeTruthy();

    const getClaimResp = await adminRequest.get(`/api/sessions/${sessionId}/claim`);
    const claimState = await getClaimResp.json();
    expect(claimState.claimed).toBe(true);

    const releaseResp = await adminRequest.post(`/api/sessions/${sessionId}/release`);
    expect(releaseResp.ok()).toBeTruthy();

    const getClaim2Resp = await adminRequest.get(`/api/sessions/${sessionId}/claim`);
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

    // 1r 切片后 SDK logger 输出 JSON,不再有 marketing-monitor 字面量;
    // 改用 source 字段标识(更稳定,不依赖品牌名)
    expect(logs.join('\n')).toContain('"source":"visitor-sdk"');
    await visitorCtx.close();
  });

  test('1h 场景4：登出流', async ({ adminRequest }) => {
    const logoutResp = await adminRequest.post('/api/auth/logout');
    expect(logoutResp.ok()).toBeTruthy();
  });
});
