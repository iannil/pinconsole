// 1z P1-1:验证 WSTransport 收到 operator command 后,后续事件 envelope 继承 trace_id。
//
// 通过 mock WebSocket 全局对象,模拟 server → SDK 推送 command envelope,
// 然后验证 SDK 后续 sendEvent 调用使用的 trace_id 与 command 一致(窗口内)。

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { encode, decode } from '@msgpack/msgpack';
import type { Envelope } from '@pinconsole/proto';
import { PROTOCOL_VERSION } from '@pinconsole/proto';
import { WSTransport } from '../src/transport/ws';

// Mock WebSocket:捕获 send 的 bytes,允许测试模拟 server 推送消息
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static OPEN = 1;
  static CLOSED = 3;

  readyState = MockWebSocket.OPEN;
  binaryType: string = 'arraybuffer';
  onopen: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;

  sentBytes: Uint8Array[] = [];

  constructor(public url: string) {
    MockWebSocket.instances.push(this);
  }

  send(bytes: Uint8Array): void {
    this.sentBytes.push(bytes);
  }

  close(): void {
    this.readyState = MockWebSocket.CLOSED;
  }

  /** 测试 helper:模拟 server 推送消息 */
  simulateMessage(env: Envelope): void {
    const bytes = encode(env);
    // 必须传 ArrayBuffer (sdk ws.ts onmessage 检查 instanceof ArrayBuffer)
    const ab = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
    this.onmessage?.({ data: ab } as unknown as MessageEvent);
  }

  /** 触发 onopen(测试手动调) */
  fireOpen(): void {
    this.onopen?.(new Event('open'));
  }
}

const HELLO = {
  visitor_id: 'fp-test',
  session_id: 's-test',
  sdk_version: 'test',
  capabilities: { events: [], co_browsing: false, recording: false },
};

describe('WSTransport trace_id inheritance', () => {
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    MockWebSocket.instances = [];
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
  });

  afterEach(() => {
    globalThis.WebSocket = originalWebSocket;
  });

  /**
   * 公共 setup:创建 transport + 启动 + 手动触发 onopen + 推 ack。
   * 返回 mockWs 供测试推 command + 验证 sentBytes。
   */
  function setupConnected(): { transport: WSTransport; mockWs: MockWebSocket } {
    const transport = new WSTransport({
      endpoint: 'ws://localhost:8080/ws/visitor',
      hello: HELLO,
    });
    transport.start();
    const mockWs = MockWebSocket.instances[0]!;
    mockWs.fireOpen();
    // 推 ack → helloAcked=true + status=connected
    mockWs.simulateMessage({
      v: PROTOCOL_VERSION, type: 'ack', ts: Date.now(), payload: { ok: true },
    });
    // 清空 hello envelope,只留测试期间发的
    mockWs.sentBytes.length = 0;
    return { transport, mockWs };
  }

  it('incoming command trace_id is inherited by subsequent events', () => {
    const { transport, mockWs } = setupConnected();
    mockWs.simulateMessage({
      v: PROTOCOL_VERSION, type: 'command', ts: Date.now(),
      trace_id: 'command-trace-1', payload: { type: 'cursor_highlight', ts: Date.now() },
    });

    transport.sendEvent({ type: 2, timestamp: Date.now(), data: {} });
    expect(mockWs.sentBytes.length).toBe(1);
    const env = decode(mockWs.sentBytes[0]!) as Envelope;
    expect(env.trace_id).toBe('command-trace-1');
  });

  it('multiple events within window all inherit command trace_id', () => {
    const { transport, mockWs } = setupConnected();
    mockWs.simulateMessage({
      v: PROTOCOL_VERSION, type: 'command', ts: Date.now(),
      trace_id: 'shared-trace', payload: { type: 'click', ts: Date.now() },
    });

    for (let i = 0; i < 3; i++) {
      transport.sendEvent({ type: 2, timestamp: Date.now(), data: { i } });
    }
    expect(mockWs.sentBytes.length).toBe(3);
    for (const bytes of mockWs.sentBytes) {
      const env = decode(bytes) as Envelope;
      expect(env.trace_id).toBe('shared-trace');
    }
  });

  it('event trace_id falls back to fresh generation after window expires', () => {
    vi.useFakeTimers();
    const { transport, mockWs } = setupConnected();
    mockWs.simulateMessage({
      v: PROTOCOL_VERSION, type: 'command', ts: Date.now(),
      trace_id: 'expired-trace', payload: { type: 'click', ts: Date.now() },
    });

    // 推进时间超过 5 秒 TTL
    vi.advanceTimersByTime(6000);

    transport.sendEvent({ type: 2, timestamp: Date.now(), data: {} });
    expect(mockWs.sentBytes.length).toBe(1);
    const env = decode(mockWs.sentBytes[0]!) as Envelope;
    expect(env.trace_id).not.toBe('expired-trace');
    expect(env.trace_id).toMatch(/^[0-9a-f]{32}$/);
    vi.useRealTimers();
  });

  it('event trace_id falls back after N events exhausted', () => {
    const { transport, mockWs } = setupConnected();
    mockWs.simulateMessage({
      v: PROTOCOL_VERSION, type: 'command', ts: Date.now(),
      trace_id: 'counted-trace', payload: { type: 'click', ts: Date.now() },
    });

    // 发 10 个事件(应继承)
    for (let i = 0; i < 10; i++) {
      transport.sendEvent({ type: 2, timestamp: Date.now(), data: { i } });
    }
    // 第 11 个应 fall back
    transport.sendEvent({ type: 2, timestamp: Date.now(), data: { i: 11 } });

    const lastBytes = mockWs.sentBytes[mockWs.sentBytes.length - 1]!;
    const lastEnv = decode(lastBytes) as Envelope;
    expect(lastEnv.trace_id).not.toBe('counted-trace');
    expect(lastEnv.trace_id).toMatch(/^[0-9a-f]{32}$/);
  });

  it('events before any command use fresh trace_id', () => {
    const { transport, mockWs } = setupConnected();
    transport.sendEvent({ type: 2, timestamp: Date.now(), data: {} });
    expect(mockWs.sentBytes.length).toBe(1);
    const env = decode(mockWs.sentBytes[0]!) as Envelope;
    expect(env.trace_id).toMatch(/^[0-9a-f]{32}$/);
  });

  it('command without trace_id does not set inheritance (graceful degradation)', () => {
    const { transport, mockWs } = setupConnected();
    // command 不带 trace_id (旧版 server 或异常场景)
    mockWs.simulateMessage({
      v: PROTOCOL_VERSION, type: 'command', ts: Date.now(),
      payload: { type: 'click', ts: Date.now() },
    });
    transport.sendEvent({ type: 2, timestamp: Date.now(), data: {} });
    const env = decode(mockWs.sentBytes[0]!) as Envelope;
    expect(env.trace_id).toMatch(/^[0-9a-f]{32}$/); // 不缓存,fresh 生成
  });
});
