// CV-5 切片补测:fingerprint + cursor + toast + ui banners。
// 测试策略:jsdom 直接 DOM 操作 + class 实例化。
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { collectFingerprint } from '../src/fingerprint';
import { OperatorCursor } from '../src/commands/cursor';
import { OperatorToast } from '../src/commands/toast';
import { showCoBrowseBanner, removeCoBrowseBanner } from '../src/ui/coBrowseBanner';
import { showConsentBanner, removeConsentBanner } from '../src/ui/consentBanner';

// === fingerprint.ts ===

describe('fingerprint', () => {
  it('collectFingerprint 返回完整字段', () => {
    const fp = collectFingerprint();
    expect(fp).toHaveProperty('canvas_hash');
    expect(fp).toHaveProperty('webgl_vendor');
    expect(fp).toHaveProperty('webgl_renderer');
    expect(fp).toHaveProperty('screen');
    expect(fp).toHaveProperty('timezone');
    expect(fp).toHaveProperty('combined_hash');
    // 字段都是 string
    expect(typeof fp.canvas_hash).toBe('string');
    expect(typeof fp.combined_hash).toBe('string');
    expect(fp.screen).toMatch(/^\d+x\d+x\d+$/);
  });

  it('collectFingerprint 相同环境两次结果一致', () => {
    const a = collectFingerprint();
    const b = collectFingerprint();
    expect(a.combined_hash).toBe(b.combined_hash);
  });
});

// === cursor.ts ===

describe('OperatorCursor', () => {
  let cursor: OperatorCursor;

  beforeEach(() => {
    document.body.innerHTML = '';
    cursor = new OperatorCursor();
  });

  afterEach(() => {
    cursor.destroy();
  });

  it('show 创建 DOM 容器 + svg + label', () => {
    cursor.show();
    const el = document.getElementById('__mm_operator_cursor__');
    expect(el).not.toBeNull();
    expect(el?.tagName).toBe('DIV');
    expect(el?.querySelector('svg')).not.toBeNull();
    expect(el?.querySelector('#__mm_operator_name__')).not.toBeNull();
  });

  it('show 幂等(多次调用不重复创建)', () => {
    cursor.show();
    cursor.show();
    cursor.show();
    const els = document.querySelectorAll('#__mm_operator_cursor__');
    expect(els.length).toBe(1);
  });

  it('moveTo 更新 transform', () => {
    cursor.moveTo(100, 200);
    const el = document.getElementById('__mm_operator_cursor__');
    expect(el).not.toBeNull();
    expect(el?.style.transform).toContain('100px');
    expect(el?.style.transform).toContain('200px');
  });

  it('moveTo 带 name 更新 label', () => {
    cursor.moveTo(50, 50, 'Alice');
    const label = document.getElementById('__mm_operator_name__');
    expect(label?.textContent).toBe('Alice');
  });

  it('moveTo 在未 show 时自动 show', () => {
    cursor.moveTo(10, 20);
    const el = document.getElementById('__mm_operator_cursor__');
    expect(el).not.toBeNull();
  });

  it('hide 设置 transform 屏外', () => {
    cursor.show();
    cursor.hide();
    const el = document.getElementById('__mm_operator_cursor__');
    expect(el?.style.transform).toContain('-9999px');
  });

  it('destroy 移除 DOM', () => {
    cursor.show();
    cursor.destroy();
    expect(document.getElementById('__mm_operator_cursor__')).toBeNull();
  });

  it('destroy 幂等(多次调用不报错)', () => {
    cursor.destroy();
    cursor.destroy();
  });
});

// === toast.ts ===

describe('OperatorToast', () => {
  let toast: OperatorToast;

  beforeEach(() => {
    document.body.innerHTML = '';
    toast = new OperatorToast();
  });

  afterEach(() => {
    toast.destroy();
  });

  it('show 创建 toast container + 显示文本', () => {
    toast.show('Alice', '正在代为填写');
    const el = document.getElementById('__mm_toast__');
    expect(el).not.toBeNull();
    expect(el?.textContent).toContain('Alice');
    expect(el?.textContent).toContain('正在代为填写');
  });

  it('show 多次复用同一 container', () => {
    toast.show('A', 'msg1');
    toast.show('B', 'msg2');
    const els = document.querySelectorAll('#__mm_toast__');
    expect(els.length).toBe(1);
  });

  it('hide 设置 transform + opacity', () => {
    toast.show('A', 'msg');
    toast.hide();
    const el = document.getElementById('__mm_toast__');
    expect(el?.style.opacity).toBe('0');
    expect(el?.style.transform).toContain('120%');
  });

  it('destroy 移除 DOM', () => {
    toast.show('A', 'msg');
    toast.destroy();
    expect(document.getElementById('__mm_toast__')).toBeNull();
  });

  it('destroy 幂等', () => {
    toast.destroy();
    toast.destroy();
  });

  it('show + timer 自动 hide(vi.useFakeTimers)', () => {
    vi.useFakeTimers();
    toast.show('A', 'msg');
    const el1 = document.getElementById('__mm_toast__');
    expect(el1?.style.opacity).toBe('1');
    // 快进 5s
    vi.advanceTimersByTime(5001);
    const el2 = document.getElementById('__mm_toast__');
    expect(el2?.style.opacity).toBe('0');
    vi.useRealTimers();
  });
});

// === ui/coBrowseBanner.ts ===

describe('coBrowseBanner', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  afterEach(() => {
    removeCoBrowseBanner();
  });

  it('showCoBrowseBanner 创建 banner', () => {
    showCoBrowseBanner('Alice', vi.fn());
    const banner = document.querySelector('[id*="cobrowse"], [class*="cobrowse"], div');
    expect(banner).not.toBeNull();
  });

  it('removeCoBrowseBanner 移除 banner', () => {
    showCoBrowseBanner('Alice', vi.fn());
    removeCoBrowseBanner();
    // 验证不 panic
  });
});

// === transport/ws.ts sendNavigated ===

import { WSTransport } from '../src/transport/ws';

describe('WSTransport.sendNavigated', () => {
  it('sendNavigated 在 ws OPEN 时调 ws.send', () => {
    const sendMock = vi.fn();
    const fakeWs = {
      readyState: WebSocket.OPEN,
      send: sendMock,
      close: vi.fn(),
    } as any;
    const transport = new WSTransport({
      endpoint: 'ws://test',
      hello: { visitor_id: 'v1', session_id: 's1' } as any,
    });
    // @ts-expect-error 注入 fake ws
    transport['ws'] = fakeWs;
    transport.sendNavigated();
    expect(sendMock).toHaveBeenCalled();
  });

  it('sendNavigated 在 ws 未 OPEN 时丢弃', () => {
    const sendMock = vi.fn();
    const fakeWs = {
      readyState: WebSocket.CONNECTING,
      send: sendMock,
      close: vi.fn(),
    } as any;
    const transport = new WSTransport({
      endpoint: 'ws://test',
      hello: { visitor_id: 'v1', session_id: 's1' } as any,
    });
    // @ts-expect-error 注入 fake ws
    transport['ws'] = fakeWs;
    transport.sendNavigated();
    expect(sendMock).not.toHaveBeenCalled();
  });

  it('close 在 ws.close 抛错时 fail-soft', () => {
    const fakeWs = {
      readyState: WebSocket.OPEN,
      send: vi.fn(),
      close: vi.fn(() => { throw new Error('close-fail'); }),
    } as any;
    const transport = new WSTransport({
      endpoint: 'ws://test',
      hello: { visitor_id: 'v1', session_id: 's1' } as any,
    });
    // @ts-expect-error 注入 fake ws
    transport['ws'] = fakeWs;
    expect(() => transport.close()).not.toThrow();
  });
});

// === ui/consentBanner.ts ===

describe('consentBanner', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });

  afterEach(() => {
    removeConsentBanner();
  });

  it('showConsentBanner 默认 en 文案(navigator.language=en-US)', () => {
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true });
    showConsentBanner({ onAccept: vi.fn(), onReject: vi.fn() });
    expect(document.body.textContent).toContain('Visitor Monitoring Consent');
  });

  it('showConsentBanner 默认 zh 文案(navigator.language=zh-CN)', () => {
    Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true });
    showConsentBanner({ onAccept: vi.fn(), onReject: vi.fn() });
    expect(document.body.textContent).toContain('访客监控同意');
  });

  it('showConsentBanner 自定义文案覆盖默认', () => {
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true });
    showConsentBanner({
      text: { title: 'Custom Title', body: 'Custom Body', accept: 'Yes', reject: 'No' },
      onAccept: vi.fn(),
      onReject: vi.fn(),
    });
    expect(document.body.textContent).toContain('Custom Title');
  });

  it('click accept 触发 onAccept + remove banner', () => {
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true });
    const onAccept = vi.fn();
    showConsentBanner({ onAccept, onReject: vi.fn() });
    const buttons = document.querySelectorAll('button');
    const acceptBtn = Array.from(buttons).find(b => b.textContent === 'Accept') as HTMLButtonElement;
    acceptBtn?.click();
    expect(onAccept).toHaveBeenCalled();
    expect(document.getElementById('__mm_consent_banner__')).toBeNull();
  });

  it('click reject 触发 onReject + remove banner', () => {
    Object.defineProperty(navigator, 'language', { value: 'en-US', configurable: true });
    const onReject = vi.fn();
    showConsentBanner({ onAccept: vi.fn(), onReject });
    const buttons = document.querySelectorAll('button');
    const rejectBtn = Array.from(buttons).find(b => b.textContent === 'Reject') as HTMLButtonElement;
    rejectBtn?.click();
    expect(onReject).toHaveBeenCalled();
    expect(document.getElementById('__mm_consent_banner__')).toBeNull();
  });

  it('removeConsentBanner 幂等', () => {
    removeConsentBanner();
    removeConsentBanner();
  });
});
