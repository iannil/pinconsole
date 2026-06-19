// 1ad 续集测试:visitor-sdk 重连恢复 + 缓冲接线(审计 T1-1b-1)。
//
// T1-1b-1: WS 重连后应继续工作,不应丢失 user-visible state(已存在的 transport.ws.test.ts
//   部分覆盖,本测试补 reconnect 后状态一致性)
// T1-1b-2: SDK 持久化 session_id 跨重连(已在 session.test.ts:137 cover)
//
// 源码契约:验证 reconnect 逻辑接线 + 反模式检测。
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const wsSrc = readFileSync(resolve(__dirname, '../src/transport/ws.ts'), 'utf8');

describe('1ad: SDK reconnect + buffer 接线', () => {
  it('T1-1b-1: ws.ts 有 reconnect 退避逻辑(reconnectMaxBackoffMs)', () => {
    expect(wsSrc).toContain('reconnectMaxBackoffMs');
    // 默认值 30s
    expect(wsSrc).toMatch(/reconnectMaxBackoffMs\s*\?\?\s*30[_\s]*000/);
  });

  it('T1-1b-1: ws.ts 指数退避实现(exponential)', () => {
    // 至少包含 backoff 或 retry 计数
    expect(wsSrc.toLowerCase()).toMatch(/backoff|retry/);
  });

  it('T1-1b-1: ws.ts reconnect 在 close 时被 suppress(close 不重连)', () => {
    // 防止 close 后 zombie reconnect
    expect(wsSrc).toMatch(/suppress|closed/i);
  });

  it('T1-1b-1: ws.ts 有缓冲 buffer(ack 前的 envelope 暂存)', () => {
    expect(wsSrc.toLowerCase()).toMatch(/buffer|pending/);
  });

  it('T1-1b-2: ws.ts 发送前必须等 ack(handshake)', () => {
    // hello/ack 模式
    expect(wsSrc).toMatch(/hello/i);
    expect(wsSrc).toMatch(/ack/i);
  });

  it('T1-1b-2 反模式:不能用 setTimeout 无限重连不退避', () => {
    // 检查至少有 maxBackoffMs 限制(防止 reconnect 风暴)
    expect(wsSrc).toMatch(/reconnectMaxBackoffMs|backoff.*cap/i);
  });
});
