// 切片 1aa:useWs composable 测试
// 覆盖 connect → onopen → connected status, subscribe 记忆,close 抑制重连。
//
// useWs 内部调 onUnmounted,需在组件 setup 上下文中跑。
// 用 @vue/test-utils mount 一个轻量包装组件。
// 需 jsdom 环境(@vue/test-utils 内部依赖 document)。

// @vitest-environment jsdom

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { defineComponent, h } from 'vue';
import { mount } from '@vue/test-utils';
import { encode } from '@msgpack/msgpack';
import type { Envelope, PresencePayload } from '@pinconsole/proto';
import { PROTOCOL_VERSION } from '@pinconsole/proto';
import { useWs, type UseWsOptions } from '../src/composables/useWs';

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 3;

  readyState = MockWebSocket.CONNECTING;
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
    this.onclose?.({ code: 1006, reason: 'test' } as CloseEvent);
  }
}

function mountWithUseWs(opts: UseWsOptions) {
  let captured!: ReturnType<typeof useWs>;
  const Wrapper = defineComponent({
    setup() {
      captured = useWs(opts);
      return () => h('div');
    },
  });
  const wrapper = mount(Wrapper);
  return { wrapper, composable: captured! };
}

describe('useWs composable', () => {
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    MockWebSocket.instances = [];
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
  });

  afterEach(() => {
    globalThis.WebSocket = originalWebSocket;
  });

  it('transitions to connected on open', async () => {
    const statuses: string[] = [];
    const { composable } = mountWithUseWs({
      endpoint: 'ws://test',
      onStatusChange: (s) => statuses.push(s),
    });

    composable.connect();
    expect(statuses).toContain('connecting');

    MockWebSocket.instances[0]!.fireOpen();

    expect(composable.status.value).toBe('connected');
    expect(statuses).toContain('connected');
  });

  it('auto-resubscribes sessions on reconnect', async () => {
    const { composable } = mountWithUseWs({
      endpoint: 'ws://test',
    });
    composable.connect();
    MockWebSocket.instances[0]!.fireOpen();
    composable.subscribe('s1');
    // 清空 sentBytes(包含第一次 subscribe)
    MockWebSocket.instances[0]!.sentBytes.length = 0;

    // 模拟重连:第二个 WS instance
    MockWebSocket.instances[0]!.fireClose();
    // 立即调度重连 timer,需要 fake timers
    // 简化:直接创建新连接(跳过 timer 检查)
    expect(composable.status.value).not.toBe('connected');

    // 重连一次
    composable.connect();
    const second = MockWebSocket.instances[1]!;
    second.fireOpen();
    // 第二次 open 应触发 resubscribe
    expect(second.sentBytes.length).toBeGreaterThan(0);
  });

  it('onPresence callback fires when presence envelope arrives', async () => {
    const presences: PresencePayload[] = [];
    const { composable } = mountWithUseWs({
      endpoint: 'ws://test',
      onPresence: (p) => presences.push(p),
    });
    composable.connect();
    MockWebSocket.instances[0]!.fireOpen();

    MockWebSocket.instances[0]!.fireMessage({
      v: PROTOCOL_VERSION,
      type: 'presence',
      ts: Date.now(),
      payload: { event: 'online', session_id: 's1', visitor_id: 'v1' },
    } as unknown as Envelope);

    expect(presences.length).toBe(1);
    expect((presences[0] as unknown as { event: string }).event).toBe('online');
  });

  it('subscribe() before open does not throw and persists for reconnect', async () => {
    const { composable } = mountWithUseWs({ endpoint: 'ws://test' });
    // connect 但不 fireOpen
    composable.connect();
    composable.subscribe('s1');

    // ws 还未 OPEN,sendSubscribe 直接 return
    expect(MockWebSocket.instances[0]!.sentBytes.length).toBe(0);

    MockWebSocket.instances[0]!.fireOpen();
    // open 后应触发 resubscribe
    expect(MockWebSocket.instances[0]!.sentBytes.length).toBeGreaterThan(0);
  });

  it('close() suppresses reconnect', async () => {
    const statuses: string[] = [];
    const { composable } = mountWithUseWs({
      endpoint: 'ws://test',
      onStatusChange: (s) => statuses.push(s),
    });
    composable.connect();
    MockWebSocket.instances[0]!.fireOpen();

    composable.close();
    MockWebSocket.instances[0]!.fireClose();

    // 应该只有一次 closed,没有 reconnecting
    const closedIdx = statuses.lastIndexOf('closed');
    const reconnectingAfterClose = statuses
      .slice(closedIdx)
      .some((s) => s === 'reconnecting');
    expect(reconnectingAfterClose).toBe(false);
  });

  it('error handler sets error.value', async () => {
    const { composable } = mountWithUseWs({ endpoint: 'ws://test' });
    composable.connect();
    MockWebSocket.instances[0]!.fireOpen();

    MockWebSocket.instances[0]!.onerror?.(new Event('error'));

    expect(composable.error.value).toBeTruthy();
  });

  it('decode failure logs warning and does not crash', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const { composable } = mountWithUseWs({
      endpoint: 'ws://test',
      onPresence: () => {
        throw new Error('should not be called');
      },
    });
    composable.connect();
    MockWebSocket.instances[0]!.fireOpen();

    // 推一段非法 bytes
    const ab = new ArrayBuffer(4);
    const view = new DataView(ab);
    view.setInt32(0, 0xdeadbeef);
    MockWebSocket.instances[0]!.onmessage?.({ data: ab } as MessageEvent);

    expect(warnSpy).toHaveBeenCalled();
    warnSpy.mockRestore();
  });
});
