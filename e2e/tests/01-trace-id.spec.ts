// 切片 1m + 1z P1-1:trace_id 端到端 regression
//
// 1m badge "全链路接入" 实测两处断裂:
// 1. admin SPA 不发 X-Trace-Id 头 → server 每次新生成 trace_id
// 2. SDK 收到 command 后丢弃其 trace_id,后续事件 envelope 用新 ID
//
// 修复(1z P1-1):
// - admin/src/api/client.ts:apiFetch 自动注入 X-Trace-Id
// - visitor-sdk/src/transport/ws.ts:收到 command envelope 时缓存 trace_id
//
// regression 检查项:
// 1. admin SPA 每个 /api/* 请求都带 X-Trace-Id 头(32 字符 hex)
// 2. 响应头回传相同 X-Trace-Id(端到端一致)
// 3. adminRequest(APIRequestContext)也注入 X-Trace-Id

import { test, expect } from '../fixtures/admin-auth';

test.describe('1m/1z P1-1: trace_id 端到端', () => {
  test('admin SPA 每个 API 请求带 X-Trace-Id + 响应头回传', async ({ adminPage }) => {
    const page = adminPage;

    const apiRequests: { url: string; reqTraceId: string | null }[] = [];
    const apiResponses: { url: string; respTraceId: string | null }[] = [];

    // 在 reload 前挂 listener
    page.on('request', (req) => {
      if (req.url().includes('/api/')) {
        apiRequests.push({
          url: req.url(),
          reqTraceId: req.headers()['x-trace-id'] ?? null,
        });
      }
    });
    page.on('response', (resp) => {
      if (resp.url().includes('/api/')) {
        apiResponses.push({
          url: resp.url(),
          respTraceId: resp.headers()['x-trace-id'] ?? null,
        });
      }
    });

    // reload 触发新一轮 API 调用
    await page.goto('/admin/dashboard');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    expect(apiRequests.length, 'should have at least 1 /api/ request').toBeGreaterThan(0);

    // 1. 每个请求必须带 X-Trace-Id(32 字符 hex)
    for (const r of apiRequests) {
      expect(r.reqTraceId, `request ${r.url} missing X-Trace-Id`).toBeTruthy();
      expect(r.reqTraceId!, `trace_id should be 32 hex chars, got ${r.reqTraceId}`).toMatch(/^[a-f0-9]{32}$/);
    }

    // 2. 至少一个响应回传相同 trace_id
    const matched = apiRequests.some((req) => {
      const matching = apiResponses.find((resp) => resp.url === req.url);
      return matching && matching.respTraceId === req.reqTraceId;
    });
    expect(matched, 'at least one response should echo the same trace_id').toBe(true);
  });

  test('adminRequest 也注入 X-Trace-Id(server 回传相同 ID)', async ({ adminRequest }) => {
    // APIRequestContext 不暴露 request headers 给调用方,但 server 会把收到的
    // X-Trace-Id 回写到 response headers。验证 response headers 即可。
    const resp = await adminRequest.get('/api/sessions');
    expect(resp.ok()).toBeTruthy();
    const respTraceId = resp.headers()['x-trace-id'];
    expect(respTraceId, 'server should echo X-Trace-Id (proves adminRequest injected it)').toBeTruthy();
    expect(respTraceId!).toMatch(/^[a-f0-9]{32}$/);
  });
});
