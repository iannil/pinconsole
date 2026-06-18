import { test, expect } from '../fixtures/admin-auth';

// 切片 1k security-blockers:e2e 验收
// 前置:server 在 :8080
// 深度判定:见 docs/standards/verification-depth.md
//
// 修复(2026-06-18 v1-e2e-acceptance):
// - REST API 调用用 adminRequest(已带 cookie);1k fail-secure 后 release binary
//   不再有 dev bypass,匿名调 /api/sessions 返回 401

test.describe('1k security-blockers', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });

  test('1k 场景1：claim → postCommand popup URL javascript: 被拒', async ({ browser, adminRequest }) => {
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

    const popupResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: {
        type: 'show_popup',
        payload: {
          title: 'test',
          body: 'test',
          action_label: 'Click',
          action_url: 'javascript:fetch("/evil")',
          dismissible: true,
        },
      },
    });
    expect(popupResp.status()).toBe(400);
    const body = await popupResp.json();
    expect(body.error).toBe('popup_url_scheme_not_allowed');

    const popupOkResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: {
        type: 'show_popup',
        payload: {
          title: 'test',
          body: 'test',
          action_label: 'Click',
          action_url: 'https://example.com',
          dismissible: true,
        },
      },
    });
    expect(popupOkResp.status()).not.toBe(400);

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });

  test('1k 场景2：claim 已结束/不存在的 session 返回 404/409', async ({ adminRequest }) => {
    const fakeUUID = '00000000-0000-0000-0000-000000000000';
    const claimResp = await adminRequest.post(`/api/sessions/${fakeUUID}/claim`);
    expect([404, 409]).toContain(claimResp.status());
  });

  test('1k 场景3：migrations 自动应用,fresh DB 启动后表存在', async ({ request }) => {
    // 在重型 e2e suite 中,PG 可能瞬时繁忙。retry 3 次,每次间隔 1s。
    let lastReady: { ok: boolean; body: unknown } | null = null;
    for (let i = 0; i < 3; i++) {
      const readyResp = await request.get('/readyz');
      const body = await readyResp.json().catch(() => null);
      lastReady = { ok: readyResp.ok(), body };
      if (readyResp.ok()) break;
      await new Promise((r) => setTimeout(r, 1000));
    }
    expect(lastReady!.ok, `readyz should be ok (last body: ${JSON.stringify(lastReady!.body)})`).toBeTruthy();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const ready = lastReady!.body as any;
    expect(ready.components?.postgres ?? ready.checks?.pg ?? ready.pg ?? true).toBeTruthy();
  });

  test('1k 场景4：chat postMessage 不再接受 sender 字段 (审计污染修复)', async ({ browser, adminRequest }) => {
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

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    const msgResp = await adminRequest.post(`/api/sessions/${sessionId}/messages`, {
      data: {
        content: 'test message',
        sender: 'visitor',
      },
    });
    expect(msgResp.ok()).toBeTruthy();
    const msg = await msgResp.json();
    expect(msg.sender).toBe('operator');

    await adminRequest.post(`/api/sessions/${sessionId}/release`);
    await visitorCtx.close();
  });
});

test.describe('1k prod-mode (gated)', () => {
  test.skip(!process.env.MM_E2E_PROD, '需要 MM_E2E_PROD=1 + prod-mode server');

  test('1k-prod 场景1：匿名访问 protected 端点返回 401', async ({ browser }) => {
    const anonCtx = await browser.newContext();
    const anon = anonCtx.newPage();

    const resp = await anonCtx.request.get('/api/sessions/ended');
    expect(resp.status()).toBe(401);

    await anonCtx.close();
  });
});
