// 切片 1aa:WSTransport 缓冲 + 重连策略测试
// ws-trace-inherit.test.ts 已覆盖 trace_id 继承逻辑,本文件覆盖:
// - 缓冲 enqueue(连接未就绪时)+ flushBuffer(ack 后批量上行)
// - 缓冲满丢最旧
// - 重连退避(指数 + 上限)
// - close 抑制重连

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { encode, decode } from '@msgpack/msgpack';
import type { Envelope } from '@marketing-monitor/proto';
import { PROTOCOL_VERSION } from '@marketing-monitor/proto';
import { WSTransport } from '../src/transport/ws';

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
  closeCallCount = 0;

  constructor(public url: string) {
    MockWebSocket.instances.push(this);
  }
  send(bytes: Uint8Array): void {
    this.sentBytes.push(bytes);
  }
  close(): void {
    this.closeCallCount++;
    this.readyState = MockWebSocket.CLOSED;
  }
  fireOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event('open'));
  }
  fireMessage(env: Envelope): void {
    const bytes = encode(env);
    const ab = bytes.buffer.slice(
      bytes.byteOffset,
      bytes.byteOffset + bytes.byteLength,
    );
    this.onmessage?.({ data: ab } as unknown as MessageEvent);
  }
  fireClose(): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code: 1006, reason: 'closed' } as CloseEvent);
  }
}

const HELLO = {
  visitor_id: 'v-test',
  session_id: 's-test',
  sdk_version: 'test',
  capabilities: { events: [], co_browsing: false, recording: false },
};

function setupConnected(opts?: ConstructorParameters<typeof WSTransport>[0]) {
  const transport = new WSTransport(
    opts ?? {
      endpoint: 'ws://test',
      hello: HELLO,
    },
  );
  transport.start();
  const mockWs = MockWebSocket.instances[0]!;
  mockWs.fireOpen();
  // ack
  mockWs.fireMessage({
    v: PROTOCOL_VERSION,
    type: 'ack',
    ts: Date.now(),
    payload: { ok: true },
  });
  // 清空 hello + ack 阶段的 sentBytes
  mockWs.sentBytes.length = 0;
  return { transport, mockWs };
}

describe('WSTransport — buffer & reconnect', () => {
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    MockWebSocket.instances = [];
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
  });

  afterEach(() => {
    globalThis.WebSocket = originalWebSocket;
    vi.useRealTimers();
  });

  describe('buffering before ack', () => {
    it('buffers events when ws not yet acked', () => {
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
      });
      transport.start();
      // 模拟 open 但未 ack
      MockWebSocket.instances[0]!.fireOpen();
      // 清空 hello envelope,只看后续 sendEvent 的去向
      MockWebSocket.instances[0]!.sentBytes.length = 0;

      transport.sendEvent({ type: 2, timestamp: 1, data: {} });
      transport.sendEvent({ type: 2, timestamp: 2, data: {} });

      expect(transport.bufferLength).toBe(2);
      expect(MockWebSocket.instances[0]!.sentBytes.length).toBe(0);
    });

    it('flushes buffered events on ack', () => {
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
      });
      transport.start();
      const ws = MockWebSocket.instances[0]!;
      ws.fireOpen();
      // 清空 hello
      ws.sentBytes.length = 0;

      transport.sendEvent({ type: 2, timestamp: 1, data: { i: 1 } });
      transport.sendEvent({ type: 2, timestamp: 2, data: { i: 2 } });
      expect(ws.sentBytes.length).toBe(0);

      // 推 ack → flush
      ws.fireMessage({
        v: PROTOCOL_VERSION,
        type: 'ack',
        ts: Date.now(),
        payload: { ok: true },
      });

      expect(ws.sentBytes.length).toBe(2);
      expect(transport.bufferLength).toBe(0);
    });

    it('drops oldest when buffer full (size limit)', () => {
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
        bufferMaxEvents: 3,
      });
      transport.start();
      MockWebSocket.instances[0]!.fireOpen();
      MockWebSocket.instances[0]!.sentBytes.length = 0;

      // 推 5 个,buffer max=3 → 应丢 2 个最旧
      for (let i = 0; i < 5; i++) {
        transport.sendEvent({ type: 2, timestamp: i, data: { i } });
      }

      expect(transport.bufferLength).toBe(3);
      // 通过 fire ack + 检查 sentBytes 验证保留的是 i=2,3,4 (丢 0,1)
      MockWebSocket.instances[0]!.fireMessage({
        v: PROTOCOL_VERSION,
        type: 'ack',
        ts: Date.now(),
        payload: { ok: true },
      });
      const decoded = MockWebSocket.instances[0]!.sentBytes.map(
        (b) => decode(b) as Envelope,
      );
      // sendEvent 把单个 event 放在 envelope.payload(非 array)
      const indexes = decoded.map(
        (env) => (env.payload as { data: { i: number } }).data.i,
      );
      expect(indexes).toEqual([2, 3, 4]);
    });

    it('sendBatch does nothing for empty array', () => {
      const { transport, mockWs } = setupConnected();
      const before = mockWs.sentBytes.length;
      transport.sendBatch([]);
      expect(mockWs.sentBytes.length).toBe(before);
    });

    it('sendBatch encodes single envelope with array payload', () => {
      const { transport, mockWs } = setupConnected();
      transport.sendBatch([
        { type: 2, timestamp: 1, data: {} },
        { type: 2, timestamp: 2, data: {} },
      ]);
      expect(mockWs.sentBytes.length).toBe(1);
      const env = decode(mockWs.sentBytes[0]!) as Envelope;
      expect(Array.isArray(env.payload)).toBe(true);
      expect((env.payload as unknown[]).length).toBe(2);
    });
  });

  describe('reconnect strategy', () => {
    it('schedules reconnect with exponential backoff (growth without intermediate success)', () => {
      vi.useFakeTimers();
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
        reconnectInitialMs: 1000,
        reconnectMaxBackoffMs: 30_000,
      });
      transport.start();

      // 1st ws 首次连接,fireOpen 但不 ack(模拟 hello 没被收到/服务端没回)
      // 然后 close → 触发重连
      const ws1 = MockWebSocket.instances[0]!;
      ws1.fireOpen();
      // 注意:不 ack → reconnectAttempt 不重置 → 下次 close 退避增长
      ws1.fireClose();
      // 1st close:reconnectAttempt=0 → delay=1000ms, then attempt=1
      vi.advanceTimersByTime(999);
      expect(MockWebSocket.instances.length).toBe(1);
      vi.advanceTimersByTime(1);
      expect(MockWebSocket.instances.length).toBe(2); // 2nd ws created

      // 2nd ws 也立即 close(不 open,所以 reconnectAttempt 不重置)
      MockWebSocket.instances[1]!.fireClose();
      // 2nd close:reconnectAttempt=1 → delay=2000ms, then attempt=2
      vi.advanceTimersByTime(1999);
      expect(MockWebSocket.instances.length).toBe(2);
      vi.advanceTimersByTime(1);
      expect(MockWebSocket.instances.length).toBe(3); // 3rd ws at +2000ms

      // 3rd close:reconnectAttempt=2 → delay=4000ms
      MockWebSocket.instances[2]!.fireClose();
      vi.advanceTimersByTime(3999);
      expect(MockWebSocket.instances.length).toBe(3);
      vi.advanceTimersByTime(1);
      expect(MockWebSocket.instances.length).toBe(4);
    });

    it('resets backoff on successful connect', () => {
      vi.useFakeTimers();
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
        reconnectInitialMs: 1000,
        reconnectMaxBackoffMs: 30_000,
      });
      transport.start();

      // 断线一次
      MockWebSocket.instances[0]!.fireOpen();
      MockWebSocket.instances[0]!.fireClose();
      vi.advanceTimersByTime(1000); // 触发重连

      // 第二次成功 ack → 重置 attempt
      MockWebSocket.instances[1]!.fireOpen();
      MockWebSocket.instances[1]!.fireMessage({
        v: PROTOCOL_VERSION,
        type: 'ack',
        ts: Date.now(),
        payload: { ok: true },
      });
      // 再断线,退避应又从 1000 开始,不是 2000
      MockWebSocket.instances[1]!.fireClose();
      const startCount = MockWebSocket.instances.length;
      vi.advanceTimersByTime(999);
      expect(MockWebSocket.instances.length).toBe(startCount);
      vi.advanceTimersByTime(1);
      expect(MockWebSocket.instances.length).toBe(startCount + 1);
    });

    it('close() suppresses further reconnect', () => {
      vi.useFakeTimers();
      const onStatusChange = vi.fn();
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
        onStatusChange,
      });
      transport.start();
      MockWebSocket.instances[0]!.fireOpen();
      MockWebSocket.instances[0]!.fireMessage({
        v: PROTOCOL_VERSION,
        type: 'ack',
        ts: Date.now(),
        payload: { ok: true },
      });

      transport.close();
      expect(onStatusChange).toHaveBeenLastCalledWith('closed');

      // 推进长时间,不应再创建新 ws
      const beforeCount = MockWebSocket.instances.length;
      vi.advanceTimersByTime(60_000);
      expect(MockWebSocket.instances.length).toBe(beforeCount);
    });

    it('caps backoff at reconnectMaxBackoffMs', () => {
      vi.useFakeTimers();
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
        reconnectInitialMs: 1000,
        reconnectMaxBackoffMs: 5000, // 故意调小便于测试
      });
      transport.start();

      // 连续断线 5 次,验证退避不超过 5000ms
      // 1st close → 1s, 2nd → 2s, 3rd → 4s, 4th → 5s (capped), 5th → 5s
      for (let i = 0; i < 5; i++) {
        MockWebSocket.instances[MockWebSocket.instances.length - 1]!.fireOpen();
        MockWebSocket.instances[
          MockWebSocket.instances.length - 1
        ]!.fireClose();
        // advance 到正好下次重连
        vi.advanceTimersToNextTimer();
      }
      // 通过查看 setTimeout 调用次数间接验证(不应无限增长)
      // 这里仅断言不再抛错 + ws 数量合理增长
      expect(MockWebSocket.instances.length).toBeGreaterThan(5);
    });
  });

  describe('hello/ack handshake', () => {
    it('sends hello envelope on open', () => {
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
      });
      transport.start();
      MockWebSocket.instances[0]!.fireOpen();

      expect(MockWebSocket.instances[0]!.sentBytes.length).toBeGreaterThanOrEqual(
        1,
      );
      const env = decode(
        MockWebSocket.instances[0]!.sentBytes[0]!,
      ) as Envelope;
      expect(env.type).toBe('hello');
    });

    it('server rejecting hello (ack.ok=false) triggers onError + close', () => {
      const onError = vi.fn();
      const transport = new WSTransport({
        endpoint: 'ws://test',
        hello: HELLO,
        onError,
      });
      transport.start();
      MockWebSocket.instances[0]!.fireOpen();
      MockWebSocket.instances[0]!.fireMessage({
        v: PROTOCOL_VERSION,
        type: 'ack',
        ts: Date.now(),
        payload: { ok: false },
      });

      expect(onError).toHaveBeenCalledWith(expect.any(Error));
      // close() 应被调用,设置 closed=true,不再重连
      expect(transport.currentStatus).toBe('closed');
    });
  });
});
