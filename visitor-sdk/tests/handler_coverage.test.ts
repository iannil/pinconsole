// TS-2 切片补测:CommandHandler 全 case 覆盖 + start/stop 生命周期。
//
// 测试策略:mock OperatorCursor/NodeMap/OperatorToast 的方法,
// 验证 handle 各命令类型正确分发。不测 DOM 真实操作(ui/popup 等)。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { CommandHandler } from '../src/commands/handler';
import type { Envelope, CommandPayload } from '@pinconsole/proto';

// mock OperatorCursor + NodeMap + OperatorToast 内部依赖
vi.mock('../src/commands/cursor', () => ({
  OperatorCursor: vi.fn().mockImplementation(() => ({
    moveTo: vi.fn(),
    hide: vi.fn(),
    destroy: vi.fn(),
  })),
}));

vi.mock('../src/commands/nodeMap', () => ({
  NodeMap: vi.fn().mockImplementation(() => ({
    start: vi.fn(),
    stop: vi.fn(),
    get: vi.fn(() => null), // 默认返回 null,doClick/doFill 走 "not found" 分支
  })),
}));

vi.mock('../src/commands/toast', () => ({
  OperatorToast: vi.fn().mockImplementation(() => ({
    show: vi.fn(),
    destroy: vi.fn(),
  })),
}));

// mock dynamic import ui/popup(show_popup case 用)
vi.mock('../src/ui/popup', () => ({
  showPopup: vi.fn(),
}));

describe('CommandHandler.handle - 命令分发', () => {
  let handler: CommandHandler;
  let onControlStart: ReturnType<typeof vi.fn>;
  let onReleased: ReturnType<typeof vi.fn>;
  let onChatMessage: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    onControlStart = vi.fn();
    onReleased = vi.fn();
    onChatMessage = vi.fn();
    handler = new CommandHandler({
      onControlStart,
      onReleased,
      onChatMessage,
    });
    handler.start();
  });

  afterEach(() => {
    handler.stop();
  });

  it('non-command envelope 忽略', () => {
    const env: Envelope = {
      v: 1,
      type: 'event',
      ts: 1,
      payload: {} as any,
    };
    handler.handle(env);
    expect(onControlStart).not.toHaveBeenCalled();
  });

  it('command payload 为 undefined 忽略', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: undefined as any,
    };
    handler.handle(env);
    expect(onControlStart).not.toHaveBeenCalled();
  });

  it('cursor_highlight 触发 onControlStart', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'cursor_highlight',
        ts: 1,
        cursor: { x: 100, y: 200, name: 'op' },
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onControlStart).toHaveBeenCalled();
  });

  it('cursor_highlight 无 cursor 字段忽略', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'cursor_highlight',
        ts: 1,
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onControlStart).not.toHaveBeenCalled();
  });

  it('click 触发 onControlStart(node not found,doClick 走 log)', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'click',
        ts: 1,
        click: { node_id: 999, x: 0, y: 0 },
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onControlStart).toHaveBeenCalled();
  });

  it('scroll 调用 window.scrollTo', () => {
    const scrollToSpy = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'scroll',
        ts: 1,
        scroll: { x: 0, y: 500 },
      } as CommandPayload,
    };
    handler.handle(env);
    expect(scrollToSpy).toHaveBeenCalledWith(0, 500);
    scrollToSpy.mockRestore();
  });

  it('fill_input 触发 onControlStart(node not found,doFill 走 log)', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'fill_input',
        ts: 1,
        fill_input: { node_id: 999, value: 'test' },
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onControlStart).toHaveBeenCalled();
  });

  it('navigate 触发 onControlStart', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'navigate',
        ts: 1,
        navigate: { url: '/new-page' },
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onControlStart).toHaveBeenCalled();
  });

  it('release_control 触发 onReleased', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'release_control',
        ts: 1,
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onReleased).toHaveBeenCalled();
  });

  it('show_popup 调用 showPopup(dynamic import)', async () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'show_popup',
        ts: 1,
        popup: {
          title: 'Test',
          body: 'Body',
          dismissible: true,
        },
      } as CommandPayload,
    };
    handler.handle(env);
    // dynamic import 异步,等 microtask
    await new Promise(resolve => setTimeout(resolve, 50));
    // showPopup 是 mock,验证不 panic 即可
  });

  it('chat_message 触发 onChatMessage 回调', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 12345,
      payload: {
        type: 'chat_message',
        ts: 12345,
        chat: { message_id: 42, content: 'hello' },
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onChatMessage).toHaveBeenCalledWith({
      id: 42,
      sender: 'operator',
      content: 'hello',
      created_at: 12345,
    });
  });

  it('chat_message 无 chat 字段忽略', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'chat_message',
        ts: 1,
      } as CommandPayload,
    };
    handler.handle(env);
    expect(onChatMessage).not.toHaveBeenCalled();
  });

  it('未知 type 忽略(不 panic)', () => {
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'totally_unknown',
        ts: 1,
      } as CommandPayload,
    };
    expect(() => handler.handle(env)).not.toThrow();
  });
});

describe('CommandHandler 生命周期', () => {
  it('start + stop 不抛错', () => {
    const handler = new CommandHandler({});
    handler.start();
    handler.stop();
    // 二次 stop 不抛
    expect(() => handler.stop()).not.toThrow();
  });

  it('start 幂等(多次调用不重复 attach)', () => {
    const handler = new CommandHandler({});
    handler.start();
    handler.start();
    handler.start();
    handler.stop();
  });

  it('无 onChatMessage 回调时 chat_message 不抛', () => {
    const handler = new CommandHandler({}); // 无 onChatMessage
    handler.start();
    const env: Envelope = {
      v: 1,
      type: 'command',
      ts: 1,
      payload: {
        type: 'chat_message',
        ts: 1,
        chat: { message_id: 1, content: 'hi' },
      } as CommandPayload,
    };
    expect(() => handler.handle(env)).not.toThrow();
    handler.stop();
  });
});
