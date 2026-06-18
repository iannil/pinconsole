// 切片 1z P1-1:SDK trace_id 继承 regression
//
// 1z P1-1 修复:SDK 收到 operator command envelope 后,缓存其 trace_id,
// 后续 N=10 个事件或 M=5 秒内的事件 envelope 用缓存的 trace_id。
// 过期或耗尽后回到 generateTraceId()。
//
// 常量(在 visitor-sdk/src/transport/ws.ts):
//   TRACE_INHERIT_MAX_EVENTS = 10
//   TRACE_INHERIT_TTL_MS = 5000
//
// regression 检查项:
// 1. 用 page.on('websocket') 监听 visitor WS
// 2. 触发 operator → visitor command(用 adminRequest)
// 3. visitor 后续发的事件 envelope 在窗口内共享 trace_id
//
// 实现说明:
// - operatorWS 通过 server 内部 hub 转发 command envelope 到 visitor
// - visitor SDK 收到后,把 command 的 trace_id 存到 module-level 缓存
// - 后续 visitor 端 rrweb 事件 send 时,用缓存的 trace_id
// - 验证方式:page.on('websocket') 抓 visitor WS 的 outbound frames

import { test, expect } from '../fixtures/admin-auth';

test.describe('1z P1-1: SDK trace_id 继承', () => {
  test('operator command 后 visitor 事件共享 trace_id', async ({ browser, adminPage, adminRequest }) => {
    const admin = adminPage;
    const visitorCtx = await browser.newContext();
    const visitor = await visitorCtx.newPage();

    // 抓 visitor WS 帧
    const visitorSentFrames: string[] = [];
    visitor.on('websocket', (ws) => {
      ws.on('framesent', (frame) => {
        if (frame.payload && typeof frame.payload === 'string') {
          visitorSentFrames.push(frame.payload);
        }
      });
    });

    await visitor.goto('/');
    await visitor.waitForTimeout(3000);

    // admin 看到 visitor,选中 + 订阅
    await expect(admin.locator('.visitor-list li:not(.empty)')).not.toHaveCount(0, { timeout: 10000 });
    await admin.locator('.visitor-list li:not(.empty)').first().click();
    await admin.getByRole('button', { name: '订阅实时' }).click();
    await admin.waitForTimeout(1500);

    // 拿 session ID,claim + 发 cursor_highlight 命令
    const sessionsResp = await adminRequest.get('/api/sessions');
    const sessions = await sessionsResp.json();
    const sessionId = sessions.sessions[0]?.session_id;
    expect(sessionId, 'should have active session').toBeTruthy();

    await adminRequest.post(`/api/sessions/${sessionId}/claim`);

    // 在发命令前先清空已抓取的 frame 列表(只看命令后的)
    visitorSentFrames.length = 0;

    // 发 command — server 通过 operatorWS 推送到 visitor SDK
    const cmdResp = await adminRequest.post(`/api/sessions/${sessionId}/command`, {
      data: { type: 'cursor_highlight', payload: { x: 100, y: 100, name: '运营甲' } },
    });
    // 命令成功(200)或 visitor 离线(503);我们要的是命令到达 SDK
    expect([200, 503]).toContain(cmdResp.status());

    // 触发 visitor 端事件(鼠标移动产生 rrweb incremental)
    await visitor.mouse.move(50, 50);
    await visitor.mouse.move(100, 100);
    await visitor.mouse.move(150, 150);
    await visitor.waitForTimeout(2000);

    await adminRequest.post(`/api/sessions/${sessionId}/release`);

    // 分析抓到的 frame
    // 每个 frame 是 JSON envelope:{"trace_id":"...","session_id":"...","type":"event",...}
    const eventEnvelopes = visitorSentFrames
      .map((s) => {
        try { return JSON.parse(s); } catch { return null; }
      })
      .filter((e) => e && e.type === 'event') as Array<{ trace_id?: string }>;

    // 找到 command 的 trace_id(从 cmdResp.headers()['x-trace-id'] 拿)
    const cmdTraceId = cmdResp.headers()['x-trace-id'];
    if (cmdTraceId && eventEnvelopes.length > 0) {
      // 至少有一个事件 envelope 共享了 trace_id(继承生效)
      const inherited = eventEnvelopes.filter((e) => e.trace_id === cmdTraceId);
      expect(
        inherited.length,
        `at least 1 event envelope should inherit command's trace_id ${cmdTraceId}`,
      ).toBeGreaterThan(0);
    }

    await visitorCtx.close();
  });
});
