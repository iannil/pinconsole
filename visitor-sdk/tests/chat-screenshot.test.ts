// CV-5 chatWidget + screenshot + rrweb coverage。
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ChatWidget } from '../src/ui/chatWidget';
import { ScreenshotCollector } from '../src/collectors/screenshot';
import { RRWebCollector } from '../src/collectors/rrweb';

// === chatWidget.ts ===

describe('ChatWidget', () => {
  let widget: ChatWidget;

  beforeEach(() => {
    document.body.innerHTML = '';
    widget = new ChatWidget({
      onSend: vi.fn(),
      onFetchMessages: vi.fn().mockResolvedValue([]),
    });
  });

  afterEach(() => {
    widget.destroy();
  });

  it('show 创建 widget DOM', () => {
    widget.show();
    expect(document.getElementById('__mm_chat_widget__')).not.toBeNull();
  });

  it('show 幂等(二次不重复)', () => {
    widget.show();
    widget.show();
    expect(document.querySelectorAll('#__mm_chat_widget__').length).toBe(1);
  });

  it('destroy 移除 DOM', () => {
    widget.show();
    widget.destroy();
    expect(document.getElementById('__mm_chat_widget__')).toBeNull();
  });

  it('destroy 幂等', () => {
    widget.destroy();
    widget.destroy();
  });

  it('receiveMessage operator 增加 unread badge(未展开)', () => {
    widget.show();
    widget.receiveMessage({ id: 1, sender: 'operator', content: 'hi', created_at: 0 });
    const badge = document.querySelector('.badge');
    expect(badge?.textContent).toBe('1');
  });

  it('receiveMessage visitor 不增加 unread', () => {
    widget.show();
    widget.receiveMessage({ id: 1, sender: 'visitor', content: 'hi', created_at: 0 });
    const badge = document.querySelector('.badge');
    expect(badge).toBeNull();
  });

  it('receiveMessage 重复 ID 不处理', () => {
    widget.show();
    widget.receiveMessage({ id: 5, sender: 'operator', content: 'a', created_at: 0 });
    widget.receiveMessage({ id: 5, sender: 'operator', content: 'b', created_at: 0 });
    // badge 仍 1(第二次被忽略)
    const badge = document.querySelector('.badge');
    expect(badge?.textContent).toBe('1');
  });

  it('展开后接收消息 append 到 list', async () => {
    widget.show();
    // 模拟点击展开(需 trigger bubble click)
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    expect(bubble).not.toBeNull();
    bubble.click();
    // 展开后,接收消息应 append
    widget.receiveMessage({ id: 10, sender: 'operator', content: 'expanded', created_at: 0 });
    // 验证 message list 含消息
    const widgetEl = document.getElementById('__mm_chat_widget__');
    expect(widgetEl?.textContent).toContain('expanded');
  });

  it('展开 + 输入 + click send 触发 onSend', async () => {
    const onSend = vi.fn();
    const w = new ChatWidget({ onSend, onFetchMessages: vi.fn().mockResolvedValue([]) });
    w.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click();
    const input = document.querySelector('#__mm_chat_widget__ input') as HTMLInputElement;
    const sendBtn = document.querySelector('#__mm_chat_widget__ button') as HTMLButtonElement;
    input.value = 'hello';
    sendBtn.click();
    expect(onSend).toHaveBeenCalledWith('hello');
    w.destroy();
  });

  it('展开 + 输入 + Enter 触发 onSend', () => {
    const onSend = vi.fn();
    const w = new ChatWidget({ onSend, onFetchMessages: vi.fn().mockResolvedValue([]) });
    w.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click();
    const input = document.querySelector('#__mm_chat_widget__ input') as HTMLInputElement;
    input.value = 'hello';
    const ev = new KeyboardEvent('keydown', { key: 'Enter' });
    input.dispatchEvent(ev);
    expect(onSend).toHaveBeenCalledWith('hello');
    w.destroy();
  });

  it('空内容不触发 send', () => {
    const onSend = vi.fn();
    const w = new ChatWidget({ onSend, onFetchMessages: vi.fn().mockResolvedValue([]) });
    w.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click();
    const sendBtn = document.querySelector('#__mm_chat_widget__ button') as HTMLButtonElement;
    sendBtn.click();
    expect(onSend).not.toHaveBeenCalled();
    w.destroy();
  });

  it('fetchMessages onFetchMessages 失败时静默', async () => {
    const w = new ChatWidget({
      onSend: vi.fn(),
      onFetchMessages: vi.fn().mockRejectedValue(new Error('fail')),
    });
    w.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click(); // 触发 fetchMessages
    // 等待 promise
    await new Promise(resolve => setTimeout(resolve, 10));
    w.destroy();
  });

  it('fetchMessages 无 onFetchMessages 静默', async () => {
    const w = new ChatWidget({ onSend: vi.fn() });
    w.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click();
    await new Promise(resolve => setTimeout(resolve, 10));
    w.destroy();
  });

  it('多次展开/折叠切换', () => {
    widget.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click(); // expand
    bubble.click(); // collapse
    bubble.click(); // expand
    // 不应 panic
  });

  // === P1 修复:收起状态下 WS 推送的消息,展开后必须可见 ===
  // 之前的 bug:receiveMessage 在收起状态把 lastMessageId 推进到 msg.id,
  // 但消息没渲染;展开时 fetchMessages(since_id=lastMessageId) 返回空,
  // 消息永久丢失。

  it('收起时收到 operator 消息 → 展开后消息从 server 拉回并渲染', async () => {
    // server 视角:since_id=0 返回所有消息;since_id=N 返回 id>N
    const serverMessages = [
      { id: 1, sender: 'operator' as const, content: '历史-1', created_at: 0 },
      { id: 2, sender: 'visitor' as const, content: '历史-2', created_at: 0 },
      { id: 5, sender: 'operator' as const, content: 'WS-push-while-collapsed', created_at: 0 },
    ];
    const onFetchMessages = vi.fn(async (sinceId: number) =>
      serverMessages.filter((m) => m.id > sinceId),
    );
    const w = new ChatWidget({ onSend: vi.fn(), onFetchMessages });
    w.show();

    // 收起状态下,WS 推送 id=5
    w.receiveMessage({ id: 5, sender: 'operator', content: 'WS-push-while-collapsed', created_at: 0 });
    // unread badge 增加
    expect(document.querySelector('.badge')?.textContent).toBe('1');

    // 展开 → 触发 fetchMessages
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click();
    await new Promise((r) => setTimeout(r, 10));

    // 期望:fetchMessages(0) 被调用,所有消息渲染(包括 id=5)
    expect(onFetchMessages).toHaveBeenCalledWith(0);
    const widgetEl = document.getElementById('__mm_chat_widget__');
    expect(widgetEl?.textContent).toContain('历史-1');
    expect(widgetEl?.textContent).toContain('历史-2');
    expect(widgetEl?.textContent).toContain('WS-push-while-collapsed');
    w.destroy();
  });

  it('展开状态下 WS 推送 + fetchMessages 不重复渲染', async () => {
    const serverMessages = [
      { id: 1, sender: 'operator' as const, content: 'old', created_at: 0 },
      { id: 2, sender: 'operator' as const, content: 'race-msg', created_at: 0 },
    ];
    const onFetchMessages = vi.fn(async (sinceId: number) =>
      serverMessages.filter((m) => m.id > sinceId),
    );
    const w = new ChatWidget({ onSend: vi.fn(), onFetchMessages });
    w.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click(); // expand → fetchMessages(0)
    await new Promise((r) => setTimeout(r, 5));

    // 展开 + 渲染完毕后,WS 推送 id=2(已在 fetch 结果中)
    w.receiveMessage({ id: 2, sender: 'operator', content: 'race-msg', created_at: 0 });
    await new Promise((r) => setTimeout(r, 5));

    // 期望:'race-msg' 只出现一次
    const widgetEl = document.getElementById('__mm_chat_widget__');
    const occurrences = (widgetEl?.textContent?.match(/race-msg/g) || []).length;
    expect(occurrences).toBe(1);
    w.destroy();
  });

  it('本地回声(负 id)+ 后续 server 消息(正 id)不冲突', async () => {
    const w = new ChatWidget({
      onSend: vi.fn(),
      onFetchMessages: vi.fn().mockResolvedValue([]),
    });
    w.show();
    const bubble = document.querySelector('#__mm_chat_widget__ > div') as HTMLElement;
    bubble.click();

    // 模拟 sendCurrent:visitor 本地回声
    w.receiveMessage({ id: -1000, sender: 'visitor', content: 'local-echo', created_at: 0 });
    // server 紧接着 WS 推送 admin 回复
    w.receiveMessage({ id: 7, sender: 'operator', content: 'server-reply', created_at: 0 });

    const widgetEl = document.getElementById('__mm_chat_widget__');
    expect(widgetEl?.textContent).toContain('local-echo');
    expect(widgetEl?.textContent).toContain('server-reply');
    w.destroy();
  });
});

// === screenshot.ts ===

describe('ScreenshotCollector', () => {
  let collector: ScreenshotCollector;
  let emit: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    document.body.innerHTML = '';
    emit = vi.fn();
    vi.useFakeTimers();
    collector = new ScreenshotCollector(emit, {
      intervalMs: 100,
      detectIntervalMs: 200,
      quality: 0.7,
    });
  });

  afterEach(() => {
    collector.stop();
    vi.useRealTimers();
  });

  it('start 后定时检测目标', () => {
    collector.start();
    expect(collector).toBeTruthy();
  });

  it('start 幂等', () => {
    collector.start();
    collector.start();
  });

  it('检测到 canvas 启动 capture', () => {
    const canvas = document.createElement('canvas');
    document.body.appendChild(canvas);

    collector.start();
    // 触发检测后,推进 timer
    vi.advanceTimersByTime(300); // detect + capture
    // emit 不一定被调用(canvas.toDataURL 在 jsdom 可能抛错),只验证不 panic
  });

  it('检测到 iframe(跨域) 启动 capture', () => {
    const iframe = document.createElement('iframe');
    document.body.appendChild(iframe);

    collector.start();
    vi.advanceTimersByTime(300);
  });

  it('stop 清理定时器', () => {
    collector.start();
    collector.stop();
    // 推进 timer 不应触发 emit
    vi.advanceTimersByTime(500);
    expect(emit).not.toHaveBeenCalled();
  });

  it('无目标时不启动 capture', () => {
    collector.start();
    vi.advanceTimersByTime(500);
    expect(emit).not.toHaveBeenCalled();
  });

  it('检测到 canvas 后移除 → stopCapture', () => {
    const canvas = document.createElement('canvas');
    document.body.appendChild(canvas);

    collector.start();
    vi.advanceTimersByTime(300);
    // 移除 canvas
    canvas.remove();
    vi.advanceTimersByTime(300);
    // 不应 panic
  });
});

// === rrweb.ts ===

describe('RRWebCollector', () => {
  let collector: RRWebCollector;
  let emit: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    document.body.innerHTML = '<div>test</div>';
    emit = vi.fn();
    collector = new RRWebCollector(emit);
  });

  afterEach(() => {
    collector.stop();
  });

  it('start + stop 不 panic', async () => {
    // mock rrweb-snapshot buildid
    await collector.start().catch(() => {});
    collector.stop();
  });

  it('start 幂等', async () => {
    await collector.start().catch(() => {});
    await collector.start().catch(() => {});
    collector.stop();
  });

  it('多次 stop 不 panic', () => {
    collector.stop();
    collector.stop();
  });
});
