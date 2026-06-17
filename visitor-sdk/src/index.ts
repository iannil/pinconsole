// marketing-monitor 访客 SDK 入口。
//
// 切片 1a：仅自动初始化 + console.log。
// 切片 1b：4 类 collector + transport + 重连。
// 切片 1c：rrweb 全量采集 + 选择性截图 + 100ms/50 events 批量。

import { resolveConfig, type VisitorConfig } from './config';
import { getOrCreateVisitorId, initSession, getCachedSessionId, type SessionInfo } from './session';
import { WSTransport, type TransportStatus } from './transport/ws';
import type { HelloPayload, Envelope } from './proto/envelope';
import { Batch } from './batch';
import { RRWebCollector } from './collectors/rrweb';
import { ScreenshotCollector } from './collectors/screenshot';
import { CommandHandler } from './commands/handler';
import { ChatWidget } from './ui/chatWidget';
import { collectFingerprint } from './fingerprint';

const SDK_VERSION = '0.4.0';

class MarketingMonitorSDK {
  private config: VisitorConfig;
  private session: SessionInfo | null = null;
  private transport: WSTransport | null = null;
  private batch: Batch | null = null;
  private rrweb: RRWebCollector | null = null;
  private screenshot: ScreenshotCollector | null = null;
  private commandHandler: CommandHandler | null = null;
  private chatWidget: ChatWidget | null = null;
  private started = false;

  constructor() {
    this.config = resolveConfig();
  }

  /** 启动 SDK。DOMContentLoaded 后自动调用；也可显式调用。 */
  async start(): Promise<void> {
    if (this.started) return;
    this.started = true;

    const apiBase = this.inferApiBase();
    const wsBase = this.inferWsBase();

    try {
      this.session = await this.obtainSession(apiBase);
    } catch (e) {
      console.error('[marketing-monitor] session init failed', e);
      return;
    }

    // 1i：采集 fingerprint（canvas + WebGL + screen + tz）
    const fingerprint = collectFingerprint();
    console.log('[marketing-monitor] fingerprint', fingerprint.combined_hash);

    const hello: HelloPayload = {
      visitor_id: this.session.visitorId,
      session_id: this.session.sessionId,
      sdk_version: SDK_VERSION,
      capabilities: {
        events: ['rrweb'],
        co_browsing: false,
        recording: true,
      },
      // 1i：fingerprint 作为 hello 额外字段上报
      fingerprint,
    } as HelloPayload & { fingerprint: typeof fingerprint };

    this.transport = new WSTransport({
      endpoint: `${wsBase}/ws/visitor`,
      hello,
      onStatusChange: (s) => this.onStatus(s),
      onError: (e) => console.warn('[marketing-monitor] transport error', e),
      onMessage: (env) => this.onMessage(env),
    });
    this.transport.start();

    // 1e/1g：命令处理器（cursor_highlight / click / scroll / fill_input / navigate / show_popup / chat_message）
    // 1g：聊天 widget（右下角浮动气泡）
    this.chatWidget = new ChatWidget({
      onSend: (content) => {
        // 访客发消息：POST /api/sessions/:id/messages
        const apiBase = this.inferApiBase();
        void fetch(`${apiBase}/api/sessions/${this.session!.sessionId}/messages`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ content, sender: 'visitor' }),
          credentials: 'include',
        }).catch((e) => console.warn('[marketing-monitor] chat send failed', e));
      },
      onFetchMessages: async (sinceId) => {
        const apiBase = this.inferApiBase();
        const resp = await fetch(
          `${apiBase}/api/sessions/${this.session!.sessionId}/messages?since_id=${sinceId}`,
          { credentials: 'include' },
        );
        if (!resp.ok) return [];
        const data = await resp.json();
        return (data.messages ?? []).map((m: { id: number; sender: string; content: string; created_at: number }) => ({
          id: m.id,
          sender: m.sender as 'operator' | 'visitor',
          content: m.content,
          created_at: m.created_at,
        }));
      },
    });
    this.chatWidget.show();

    this.commandHandler = new CommandHandler({
      debug: !!this.config.debug,
      onReleased: () => {
        console.log('[marketing-monitor] co-browsing released by visitor');
        this.transport?.sendEvent({
          type: 'rrweb',
          ts: Date.now(),
          rrweb: {
            type: 99,
            timestamp: Date.now(),
            data: { kind: 'release_control', source: 'visitor_esc' },
          },
        });
      },
      onChatMessage: (msg) => {
        this.chatWidget?.receiveMessage(msg);
      },
    });
    this.commandHandler.start();

    // 批量器：100ms / 50 events 触发 flush
    this.batch = new Batch((events) => {
      this.transport?.sendBatch(events);
    });
    // 页面卸载前 flush 残余
    window.addEventListener('beforeunload', () => this.batch?.flush(), { once: true });

    // rrweb 采集器：默认 mask 所有输入
    this.rrweb = new RRWebCollector((e) => this.batch?.push(e));
    try {
      await this.rrweb.start();
    } catch (e) {
      console.warn('[marketing-monitor] rrweb start failed', e);
    }

    // 选择性截图：检测到 canvas/WebGL/iframe 才启动
    this.screenshot = new ScreenshotCollector((e) => this.batch?.push(e));
    this.screenshot.start();

    console.log('[marketing-monitor] SDK started', {
      version: SDK_VERSION,
      session_id: this.session.sessionId,
      visitor_id: this.session.visitorId,
      apiBase,
      wsBase,
    });
  }

  /** 显式停止（页面卸载时调用）。 */
  stop(): void {
    this.batch?.flush();
    this.batch?.destroy();
    this.batch = null;
    this.rrweb?.stop();
    this.rrweb = null;
    this.screenshot?.stop();
    this.screenshot = null;
    this.commandHandler?.stop();
    this.commandHandler = null;
    this.chatWidget?.destroy();
    this.chatWidget = null;
    this.transport?.close();
    this.transport = null;
    this.started = false;
  }

  /**
   * 1f：通知服务端访客即将跳转。
   * 服务端广播 presence.navigated 给 admin，admin 自动重订阅新 session。
   */
  notifyNavigated(): void {
    this.transport?.sendNavigated();
  }

  private async obtainSession(apiBase: string): Promise<SessionInfo> {
    const visitorId = getOrCreateVisitorId();
    void getCachedSessionId();
    return await initSession(visitorId, apiBase, navigator.userAgent);
  }

  private onStatus(s: TransportStatus): void {
    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.debug('[marketing-monitor] transport status', s);
    }
  }

  private onMessage(env: unknown): void {
    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.debug('[marketing-monitor] incoming message', env);
    }
    // 1e：command envelope 交给 CommandHandler
    if (env && typeof env === 'object' && 'type' in env && (env as { type: string }).type === 'command') {
      this.commandHandler?.handle(env as Envelope);
    }
  }

  private inferApiBase(): string {
    if (this.config.endpoint) {
      return this.config.endpoint.replace(/^ws/, 'http').replace(/\/ws$/, '');
    }
    return `${location.origin}`;
  }

  private inferWsBase(): string {
    if (this.config.endpoint) return this.config.endpoint.replace(/\/ws$/, '');
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${proto}//${location.host}`;
  }
}

const w = window as unknown as { __MM__?: MarketingMonitorSDK };
const sdk = new MarketingMonitorSDK();
w.__MM__ = sdk;

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => void sdk.start(), { once: true });
} else {
  void sdk.start();
}

// 1f：页面 unload 时尝试发 navigated（admin 自动重订阅新 session）
window.addEventListener('beforeunload', () => {
  try {
    sdk.notifyNavigated();
  } catch {
    // ignore
  }
  sdk.stop();
}, { once: true });

export { MarketingMonitorSDK };
