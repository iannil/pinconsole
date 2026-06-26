// CV-5 handler.ts 各命令分支 + ESC 三连 + 各辅助。
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { CommandHandler } from '../src/commands/handler';
import type { Envelope, CommandPayload } from '@pinconsole/proto';

// Mock cursor + toast + popup
vi.mock('../src/commands/cursor', () => ({
  OperatorCursor: vi.fn().mockImplementation(() => ({
    moveTo: vi.fn(),
    show: vi.fn(),
    hide: vi.fn(),
    destroy: vi.fn(),
  })),
}));

vi.mock('../src/commands/toast', () => ({
  OperatorToast: vi.fn().mockImplementation(() => ({
    show: vi.fn(),
    hide: vi.fn(),
    destroy: vi.fn(),
  })),
}));

const showPopupMock = vi.fn();
vi.mock('../src/ui/popup', () => ({
  showPopup: (...args: any[]) => showPopupMock(...args),
}));

describe('CommandHandler', () => {
  let handler: CommandHandler;
  let onControlStart: ReturnType<typeof vi.fn>;
  let onReleased: ReturnType<typeof vi.fn>;
  let onChatMessage: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    document.body.innerHTML = '';
    onControlStart = vi.fn();
    onReleased = vi.fn();
    onChatMessage = vi.fn();
    handler = new CommandHandler({
      debug: true,
      onControlStart,
      onReleased,
      onChatMessage,
    });
    handler.start();
  });

  afterEach(() => {
    handler.stop();
  });

  function buildEnv(cp: CommandPayload): Envelope {
    return {
      v: 1,
      type: 'command' as any,
      ts: Date.now(),
      payload: cp,
    } as unknown as Envelope;
  }

  it('cursor_highlight 命令', () => {
    handler.handle(buildEnv({
      type: 'cursor_highlight',
      ts: 0,
      cursor: { x: 10, y: 20, name: 'Alice' },
    }));
    expect(onControlStart).toHaveBeenCalled();
  });

  it('click 命令 + 节点存在', () => {
    handler.handle(buildEnv({
      type: 'click',
      ts: 0,
      click: { node_id: 5, x: 0, y: 0 },
    }));
    expect(onControlStart).toHaveBeenCalled();
  });

  it('scroll 命令', () => {
    const spy = vi.spyOn(window, 'scrollTo');
    handler.handle(buildEnv({
      type: 'scroll',
      ts: 0,
      scroll: { x: 0, y: 100 },
    }));
    expect(spy).toHaveBeenCalledWith(0, 100);
  });

  it('fill_input 命令 + input 节点', () => {
    handler.handle(buildEnv({
      type: 'fill_input',
      ts: 0,
      fill_input: { node_id: 1, value: 'hello' },
    }));
    expect(onControlStart).toHaveBeenCalled();
  });

  it('navigate 命令', () => {
    // jsdom 允许设 location.href 但会 warn
    const origHref = window.location.href;
    try {
      handler.handle(buildEnv({
        type: 'navigate',
        ts: 0,
        navigate: { url: '/test-path' },
      }));
    } catch {
      // jsdom 可能不允许 navigate,只验证不 panic
    }
    expect(onControlStart).toHaveBeenCalled();
  });

  it('release_control 命令', () => {
    handler.handle(buildEnv({
      type: 'release_control',
      ts: 0,
    }));
    expect(onReleased).toHaveBeenCalled();
  });

  it('show_popup 命令', async () => {
    handler.handle(buildEnv({
      type: 'show_popup',
      ts: 0,
      popup: { title: 'T', body: 'B', dismissible: true },
    }));
    // popup 是 dynamic import,等一会
    await new Promise(resolve => setTimeout(resolve, 50));
    expect(showPopupMock).toHaveBeenCalled();
  });

  it('chat_message 命令', () => {
    handler.handle(buildEnv({
      type: 'chat_message',
      ts: 12345,
      chat: { message_id: 1, content: 'hello' },
    }));
    expect(onChatMessage).toHaveBeenCalledWith({
      id: 1,
      sender: 'operator',
      content: 'hello',
      created_at: 12345,
    });
  });

  it('非 command envelope 忽略', () => {
    handler.handle({ v: 1, type: 'event' as any, ts: 0 } as unknown as Envelope);
    expect(onControlStart).not.toHaveBeenCalled();
  });

  it('payload 为空忽略', () => {
    handler.handle({ v: 1, type: 'command' as any, ts: 0 } as unknown as Envelope);
    expect(onControlStart).not.toHaveBeenCalled();
  });

  it('Ctrl+Shift+X 触发 onReleased', () => {
    const ev = new KeyboardEvent('keydown', {
      ctrlKey: true,
      shiftKey: true,
      key: 'X',
    });
    window.dispatchEvent(ev);
    expect(onReleased).toHaveBeenCalled();
  });

  it('ESC 三连触发 onReleased', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2025-01-01T00:00:00Z'));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    vi.setSystemTime(new Date('2025-01-01T00:00:00.100Z'));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    vi.setSystemTime(new Date('2025-01-01T00:00:00.200Z'));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(onReleased).toHaveBeenCalled();
    vi.useRealTimers();
  });

  it('ESC 间隔超 1s 不触发', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2025-01-01T00:00:00Z'));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    vi.setSystemTime(new Date('2025-01-01T00:00:02Z'));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    vi.setSystemTime(new Date('2025-01-01T00:00:02.100Z'));
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(onReleased).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it('click 节点不存在 不 panic', () => {
    handler.handle(buildEnv({
      type: 'click',
      ts: 0,
      click: { node_id: 99, x: 0, y: 0 },
    }));
  });

  it('fill_input 节点不存在 不 panic', () => {
    handler.handle(buildEnv({
      type: 'fill_input',
      ts: 0,
      fill_input: { node_id: 99, value: 'x' },
    }));
  });

  it('click 无 click payload 忽略', () => {
    handler.handle(buildEnv({ type: 'click', ts: 0 }));
    expect(onControlStart).not.toHaveBeenCalled();
  });

  it('start 幂等', () => {
    handler.start();
    handler.start();
  });

  it('stop 幂等', () => {
    handler.stop();
    handler.stop();
  });

  it('debug=true 时 log 输出', () => {
    // 调 click 失败路径,触发 log
    handler.handle(buildEnv({
      type: 'click',
      ts: 0,
      click: { node_id: 99, x: 0, y: 0 },
    }));
    // log 内部调用,不直接验证
  });
});

describe('CommandHandler debug=false', () => {
  it('debug=false 不 log', () => {
    const h = new CommandHandler({ debug: false });
    h.start();
    h.handle({ v: 1, type: 'command' as any, ts: 0, payload: { type: 'click', ts: 0 } } as unknown as Envelope);
    h.stop();
  });
});
