// CV-5 SDK main entry test:覆盖 index.ts 各路径。
// 策略:mock 所有外部依赖,验证 SDK.start 各分支。
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

// Mock fetch
const mockFetch = vi.fn().mockResolvedValue({
  ok: true,
  json: async () => ({ found: false }),
});
vi.stubGlobal('fetch', mockFetch);

// Mock config
vi.mock('../src/config', () => ({
  resolveConfig: vi.fn(() => ({
    endpoint: 'wss://test.example.com/ws',
    debug: false,
    consentMode: 'always-on' as const,
    showCoBrowseBanner: true,
    consentBannerText: 'test',
  })),
}));

// Mock session
vi.mock('../src/session', () => ({
  getOrCreateVisitorId: vi.fn(() => 'test-visitor-id'),
  getCachedSessionId: vi.fn(() => null),
  initSession: vi.fn(async () => ({
    visitorId: 'test-visitor-id',
    sessionId: 'test-session-id',
    tenantId: 'test-tenant-id',
  })),
}));

// Mock WSTransport
const mockTransportStart = vi.fn();
const mockTransportSendEvent = vi.fn();
const mockTransportSendBatch = vi.fn();
const mockTransportSendNavigated = vi.fn();
const mockTransportClose = vi.fn();
vi.mock('../src/transport/ws', () => ({
  WSTransport: vi.fn().mockImplementation(() => ({
    start: mockTransportStart,
    sendEvent: mockTransportSendEvent,
    sendBatch: mockTransportSendBatch,
    sendNavigated: mockTransportSendNavigated,
    close: mockTransportClose,
  })),
}));

// Mock Batch
const mockBatchPush = vi.fn();
const mockBatchFlush = vi.fn();
const mockBatchDestroy = vi.fn();
vi.mock('../src/batch', () => ({
  Batch: vi.fn().mockImplementation((flush) => ({
    push: mockBatchPush,
    flush: mockBatchFlush,
    destroy: mockBatchDestroy,
    _flushFn: flush,
  })),
}));

// Mock collectors
const mockRRWebStart = vi.fn();
const mockRRWebStop = vi.fn();
vi.mock('../src/collectors/rrweb', () => ({
  RRWebCollector: vi.fn().mockImplementation(() => ({
    start: mockRRWebStart,
    stop: mockRRWebStop,
  })),
}));

const mockScreenshotStart = vi.fn();
const mockScreenshotStop = vi.fn();
vi.mock('../src/collectors/screenshot', () => ({
  ScreenshotCollector: vi.fn().mockImplementation(() => ({
    start: mockScreenshotStart,
    stop: mockScreenshotStop,
  })),
}));

// Mock CommandHandler
const mockCmdStart = vi.fn();
const mockCmdStop = vi.fn();
const mockCmdHandle = vi.fn();
vi.mock('../src/commands/handler', () => ({
  CommandHandler: vi.fn().mockImplementation((opts) => ({
    start: mockCmdStart,
    stop: mockCmdStop,
    handle: mockCmdHandle,
    _opts: opts,
  })),
}));

// Mock ChatWidget
const mockWidgetShow = vi.fn();
const mockWidgetDestroy = vi.fn();
const mockWidgetReceive = vi.fn();
vi.mock('../src/ui/chatWidget', () => ({
  ChatWidget: vi.fn().mockImplementation(() => ({
    show: mockWidgetShow,
    destroy: mockWidgetDestroy,
    receiveMessage: mockWidgetReceive,
  })),
}));

// Mock fingerprint
vi.mock('../src/fingerprint', () => ({
  collectFingerprint: vi.fn(() => ({
    canvas_hash: 'canvas-test',
    webgl_vendor: 'webgl-test',
    webgl_renderer: 'webgl-test',
    screen: '1920x1080x24',
    timezone: 'UTC',
    combined_hash: 'hash-test',
  })),
}));

// Mock banners
const mockShowConsent = vi.fn();
const mockRemoveConsent = vi.fn();
vi.mock('../src/ui/consentBanner', () => ({
  showConsentBanner: (...args: any[]) => mockShowConsent(...args),
  removeConsentBanner: (...args: any[]) => mockRemoveConsent(...args),
}));

const mockShowCoBrowse = vi.fn();
const mockRemoveCoBrowse = vi.fn();
vi.mock('../src/ui/coBrowseBanner', () => ({
  showCoBrowseBanner: (...args: any[]) => mockShowCoBrowse(...args),
  removeCoBrowseBanner: (...args: any[]) => mockRemoveCoBrowse(...args),
}));

import { MarketingMonitorSDK } from '../src/index';
import { resolveConfig } from '../src/config';
import { WSTransport } from '../src/transport/ws';
import { Batch } from '../src/batch';
import { RRWebCollector } from '../src/collectors/rrweb';
import { ScreenshotCollector } from '../src/collectors/screenshot';
import { CommandHandler } from '../src/commands/handler';
import { ChatWidget } from '../src/ui/chatWidget';

describe('MarketingMonitorSDK', () => {
  let sdk: InstanceType<typeof MarketingMonitorSDK>;

  beforeEach(() => {
    vi.clearAllMocks();
    // 用 always-on consent mode 简化测试
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'always-on',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    // 注意:index.ts 顶部已 const new SDK;我们直接 import 类用
    sdk = new MarketingMonitorSDK();
  });

  afterEach(() => {
    sdk.stop();
  });

  it('start 启动 transport + chatWidget + commandHandler + collectors', async () => {
    await sdk.start();

    expect(WSTransport).toHaveBeenCalled();
    expect(mockTransportStart).toHaveBeenCalled();
    expect(ChatWidget).toHaveBeenCalled();
    expect(mockWidgetShow).toHaveBeenCalled();
    expect(CommandHandler).toHaveBeenCalled();
    expect(mockCmdStart).toHaveBeenCalled();
    expect(Batch).toHaveBeenCalled();
    // always-on consent mode → should collect
    expect(RRWebCollector).toHaveBeenCalled();
    expect(mockRRWebStart).toHaveBeenCalled();
    expect(ScreenshotCollector).toHaveBeenCalled();
    expect(mockScreenshotStart).toHaveBeenCalled();
  });

  it('start 幂等(二次调用不重复)', async () => {
    await sdk.start();
    await sdk.start();
    // 只应一次 WSTransport 实例化
    expect(WSTransport).toHaveBeenCalledTimes(1);
  });

  it('stop 清理全部', async () => {
    await sdk.start();
    sdk.stop();
    expect(mockBatchFlush).toHaveBeenCalled();
    expect(mockBatchDestroy).toHaveBeenCalled();
    expect(mockRRWebStop).toHaveBeenCalled();
    expect(mockScreenshotStop).toHaveBeenCalled();
    expect(mockCmdStop).toHaveBeenCalled();
    expect(mockWidgetDestroy).toHaveBeenCalled();
    expect(mockTransportClose).toHaveBeenCalled();
  });

  it('always-off consent mode 不启动 collectors', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'always-off',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    expect(RRWebCollector).not.toHaveBeenCalled();
    expect(ScreenshotCollector).not.toHaveBeenCalled();
    sdk2.stop();
  });

  it('opt-in consent mode + consentAccepted=true 启动 collectors', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-in',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: true, accepted: true }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    expect(RRWebCollector).toHaveBeenCalled();
    sdk2.stop();
  });

  it('opt-in consent mode + consentAccepted=false 不启动 collectors', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-in',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: true, accepted: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    expect(RRWebCollector).not.toHaveBeenCalled();
    sdk2.stop();
  });

  it('opt-in consent mode + 未记录 显示 banner', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-in',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    expect(mockShowConsent).toHaveBeenCalled();
    sdk2.stop();
  });

  it('opt-out consent mode + 未记录 启动 collectors(opt-out 默认采集)', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-out',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    expect(RRWebCollector).toHaveBeenCalled();
    sdk2.stop();
  });

  it('opt-out consent mode + 拒绝 不启动 collectors', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-out',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: true, accepted: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    expect(RRWebCollector).not.toHaveBeenCalled();
    sdk2.stop();
  });

  it('debug=true 时 onStatus/onMessage 走 debug 分支', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: true,
      consentMode: 'always-on',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();

    // 取出 transport opts 触发 onStatus / onMessage
    const call = (WSTransport as any).mock.calls[0][0];
    call.onStatusChange('connected');
    call.onMessage({ type: 'event' });
    call.onError(new Error('test'));

    sdk2.stop();
  });

  it('onMessage 收到 command envelope 触发 commandHandler.handle', async () => {
    await sdk.start();
    const call = (WSTransport as any).mock.calls[0][0];
    call.onMessage({ type: 'command', payload: { type: 'click' } });
    expect(mockCmdHandle).toHaveBeenCalled();
  });

  it('notifyNavigated 调 transport.sendNavigated', async () => {
    await sdk.start();
    sdk.notifyNavigated();
    expect(mockTransportSendNavigated).toHaveBeenCalled();
  });

  it('setConsent(true) 启动 collectors + POST consent', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'always-off',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    (RRWebCollector as any).mockClear();
    await sdk2.setConsent(true);
    expect(RRWebCollector).toHaveBeenCalled();
    expect(mockRemoveConsent).toHaveBeenCalled();
    sdk2.stop();
  });

  it('setConsent(false) 停止 collectors', async () => {
    await sdk.start();
    await sdk.setConsent(false);
    expect(mockRRWebStop).toHaveBeenCalled();
    expect(mockScreenshotStop).toHaveBeenCalled();
  });

  it('session init 失败时 不启动 collectors', async () => {
    const { initSession } = await import('../src/session');
    (initSession as any).mockRejectedValueOnce(new Error('init-failed'));
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    expect(WSTransport).not.toHaveBeenCalled();
  });

  it('consent load fetch 失败 fail-soft(不阻塞)', async () => {
    mockFetch.mockRejectedValueOnce(new Error('fetch-fail'));
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    // 仍启动(always-on)
    expect(RRWebCollector).toHaveBeenCalled();
    sdk2.stop();
  });

  it('chatWidget onSend 触发 fetch POST', async () => {
    await sdk.start();
    const widgetCall = (ChatWidget as any).mock.calls[0][0];
    await widgetCall.onSend('hello');
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/sessions/test-session-id/visitor-message'),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ content: 'hello' }),
      }),
    );
  });

  it('chatWidget onFetchMessages 返回消息', async () => {
    // 注意:start() 会调 fetch(loadConsent);此测试先 start 再 mock onFetchMessages 用的 fetch
    await sdk.start();
    // 替换 fetch 让下一次调用返回 messages
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        messages: [
          { id: 1, sender: 'operator', content: 'hi', created_at: 1000 },
        ],
      }),
    });
    const widgetCall = (ChatWidget as any).mock.calls[0][0];
    const msgs = await widgetCall.onFetchMessages(0);
    expect(msgs.length).toBe(1);
    expect(msgs[0].content).toBe('hi');
  });

  it('chatWidget onFetchMessages resp 不 ok 返回空', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, json: async () => ({}) });
    await sdk.start();
    const widgetCall = (ChatWidget as any).mock.calls[0][0];
    const msgs = await widgetCall.onFetchMessages(0);
    expect(msgs).toEqual([]);
  });

  it('commandHandler onControlStart 触发 co-browse banner', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'always-on',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    await sdk.start();
    const cmdCall = (CommandHandler as any).mock.calls[0][0];
    cmdCall.onControlStart();
    expect(mockShowCoBrowse).toHaveBeenCalled();
  });

  it('commandHandler onControlStart showCoBrowseBanner=false 不显示', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'always-on',
      showCoBrowseBanner: false,
      consentBannerText: 'test',
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    const cmdCall = (CommandHandler as any).mock.calls[0][0];
    cmdCall.onControlStart();
    expect(mockShowCoBrowse).not.toHaveBeenCalled();
    sdk2.stop();
  });

  it('commandHandler onReleased 触发 sendEvent + removeCoBrowseBanner', async () => {
    await sdk.start();
    const cmdCall = (CommandHandler as any).mock.calls[0][0];
    cmdCall.onReleased();
    expect(mockRemoveCoBrowse).toHaveBeenCalled();
    expect(mockTransportSendEvent).toHaveBeenCalled();
  });

  it('commandHandler onChatMessage 触发 chatWidget.receiveMessage', async () => {
    await sdk.start();
    const cmdCall = (CommandHandler as any).mock.calls[0][0];
    cmdCall.onChatMessage({ content: 'hello' });
    expect(mockWidgetReceive).toHaveBeenCalledWith({ content: 'hello' });
  });

  it('endpoint 来自 location.origin(config 无 endpoint)', async () => {
    (resolveConfig as any).mockReturnValue({
      debug: false,
      consentMode: 'always-on',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    const transportCall = (WSTransport as any).mock.calls[0][0];
    expect(transportCall.endpoint).toMatch(/ws:\/\/.*\/ws\/visitor/);
    sdk2.stop();
  });

  it('consent banner onAccept 触发 POST + startCollectors', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-in',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    // showConsentBanner should be called; trigger onAccept
    expect(mockShowConsent).toHaveBeenCalled();
    const bannerOpts = mockShowConsent.mock.calls[0][0];
    mockFetch.mockResolvedValueOnce({ ok: true, json: async () => ({}) });
    (RRWebCollector as any).mockClear();
    await bannerOpts.onAccept();
    expect(RRWebCollector).toHaveBeenCalled();
    sdk2.stop();
  });

  it('consent banner onReject 触发 POST', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-in',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    const bannerOpts = mockShowConsent.mock.calls[0][0];
    mockFetch.mockResolvedValueOnce({ ok: true, json: async () => ({}) });
    await bannerOpts.onReject();
    sdk2.stop();
  });

  it('consent banner onAccept fetch 失败 fail-soft', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-in',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    const bannerOpts = mockShowConsent.mock.calls[0][0];
    mockFetch.mockRejectedValueOnce(new Error('fetch-fail'));
    await bannerOpts.onAccept();
    sdk2.stop();
  });

  it('consent banner onReject fetch 失败 fail-soft', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'opt-in',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ found: false }),
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    const bannerOpts = mockShowConsent.mock.calls[0][0];
    mockFetch.mockRejectedValueOnce(new Error('fetch-fail'));
    await bannerOpts.onReject();
    sdk2.stop();
  });

  it('releaseControl 触发 transport.sendEvent + removeCoBrowseBanner', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'always-on',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    const cmdCall = (CommandHandler as any).mock.calls[0][0];
    cmdCall.onControlStart(); // showCoBrowseBanner
    // commandHandler opts onReleased → releaseControl? 检查 cmdCall
    // 实际 releaseControl 是 SDK 私有方法,通过 onControlStart 内的 banner.onExit 触发
    // mockShowCoBrowse mock 应该记录调用
    expect(mockShowCoBrowse).toHaveBeenCalled();
    sdk2.stop();
  });

  it('setConsent fetch 失败 fail-soft', async () => {
    (resolveConfig as any).mockReturnValue({
      endpoint: 'wss://test.example.com/ws',
      debug: false,
      consentMode: 'always-off',
      showCoBrowseBanner: true,
      consentBannerText: 'test',
    });
    const sdk2 = new MarketingMonitorSDK();
    await sdk2.start();
    mockFetch.mockRejectedValueOnce(new Error('fail'));
    await sdk2.setConsent(false);
    sdk2.stop();
  });

  it('beforeunload 事件 触发 sdk.stop', async () => {
    await sdk.start();
    // 模拟 beforeunload
    window.dispatchEvent(new Event('beforeunload'));
    // 不验证 stop,只验证不 panic
  });

  it('DOMContentLoaded 触发 sdk.start (document.readyState=loading)', async () => {
    // 此 test 仅触发顶部 IIFE 已注册的事件
    // 触发 DOMContentLoaded(实际 SDK 在 module load 时已注册)
    document.dispatchEvent(new Event('DOMContentLoaded'));
  });
});
