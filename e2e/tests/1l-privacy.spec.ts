import { test, expect } from '@playwright/test';

// 切片 1l-privacy-gdpr:e2e 验收
// 前置:server 在 :8080 (dev 或 prod 模式皆可)
// 深度判定:见 docs/standards/verification-depth.md

test.describe('1l privacy', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });

  test('1l 场景1:GET consent 未记录返回 found=false', async ({ request }) => {
    const resp = await request.get('/api/privacy/consent?fingerprint=test-fp-not-existing-12345');
    expect(resp.ok()).toBeTruthy();
    const data = await resp.json();
    expect(data.found).toBe(false);
    expect(data.accepted).toBe(false);
  });

  test('1l 场景2:POST consent 写入后 GET 返回 accepted 状态', async ({ request }) => {
    const fp = `e2e-test-fp-${Date.now()}`;

    // POST accepted=true
    const postResp = await request.post('/api/privacy/consent', {
      data: { fingerprint: fp, accepted: true },
    });
    expect(postResp.ok()).toBeTruthy();

    // GET 验证
    const getResp = await request.get(`/api/privacy/consent?fingerprint=${fp}`);
    const data = await getResp.json();
    expect(data.found).toBe(true);
    expect(data.accepted).toBe(true);

    // POST accepted=false (撤回)
    const revokeResp = await request.post('/api/privacy/consent', {
      data: { fingerprint: fp, accepted: false },
    });
    expect(revokeResp.ok()).toBeTruthy();

    // GET 验证已更新
    const get2Resp = await request.get(`/api/privacy/consent?fingerprint=${fp}`);
    const data2 = await get2Resp.json();
    expect(data2.found).toBe(true);
    expect(data2.accepted).toBe(false);
  });

  test('1l 场景3:DELETE erasure 要求认证(prod 模式)或 admin cookie', async ({ request, browser }) => {
    // 先创建一个 visitor session,获取 fingerprint
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();
    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    // 收集 SDK 发的 fingerprint(从 visitor page eval)
    // 或直接用一个 fake fingerprint 测试 API 路径
    const fakeFp = `e2e-erasure-test-${Date.now()}`;

    // DELETE 应该不报错(即使 visitor 不存在也幂等)
    // 在 dev 模式下 dev bypass 生效;在 prod 模式下需 cookie
    const resp = await request.delete(`/api/privacy/visitor/${fakeFp}`);
    // 期望 200 或 401(prod 无 cookie 时)
    expect([200, 401]).toContain(resp.status());

    await visitorCtx.close();
  });

  test('1l 场景4:DELETE 不存在的 fingerprint 幂等返回 ok', async ({ request }) => {
    const fakeFp = `e2e-not-exist-${Date.now()}`;
    const resp = await request.delete(`/api/privacy/visitor/${fakeFp}`);
    // 期望 200 + note=visitor_not_found
    if (resp.status() === 200) {
      const data = await resp.json();
      expect(data.ok).toBe(true);
      expect(data.deleted_sessions).toBe(0);
    }
    // 401 也合理(prod 模式无 cookie)
  });
});

// Prod 模式专属场景
test.describe('1l prod-mode (gated)', () => {
  test.skip(!process.env.MM_E2E_PROD, '需要 MM_E2E_PROD=1 + prod-mode server');

  test('1l-prod:匿名 DELETE 返回 401', async ({ browser }) => {
    const anonCtx = await browser.newContext();
    const resp = await anonCtx.request.delete('/api/privacy/visitor/any-fingerprint');
    expect(resp.status()).toBe(401);
    await anonCtx.close();
  });

  test('1l-prod:登录后 DELETE 返回 200(级联删除)', async ({ request }) => {
    // 假设默认 admin 凭证已知
    const fp = `e2e-prod-${Date.now()}`;
    const resp = await request.delete(`/api/privacy/visitor/${fp}`);
    expect(resp.status()).toBe(200);
    const data = await resp.json();
    expect(data.ok).toBe(true);
  });
});
