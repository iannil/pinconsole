// pinconsole 访客 SDK 入口。
//
// 切片 1a：仅自动初始化 + console.log。
// 切片 1b：4 类 collector + transport + 重连。
// 切片 1c：rrweb 全量采集 + 选择性截图 + 100ms/50 events 批量。

import { resolveConfig, type VisitorConfig } from './config';
import { getOrCreateVisitorId, initSession, getCachedSessionId, type SessionInfo } from './session';
import { WSTransport, type TransportStatus } from './transport/ws';
import type { HelloPayload, Envelope } from '@pinconsole/proto';
import { Batch } from './batch';
import { RRWebCollector } from './collectors/rrweb';
import { ScreenshotCollector } from './collectors/screenshot';
import { CommandHandler } from './commands/handler';
import { ChatWidget } from './ui/chatWidget';
import { collectFingerprint } from './fingerprint';
import { showConsentBanner, removeConsentBanner } from './ui/consentBanner';
import { showCoBrowseBanner, removeCoBrowseBanner } from './ui/coBrowseBanner';
import { injectTokenStyles } from './styles/tokens';
import { sdkLogger } from './logging';

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
  // 1l:consent 状态(是否允许采集 surveillance 数据)
  private consentAccepted: boolean | null = null;
  private fingerprint: ReturnType<typeof collectFingerprint> | null = null;

  constructor() {
    this.config = resolveConfig();
  }

  /** 启动 SDK。DOMContentLoaded 后自动调用；也可显式调用。 */
  async start(): Promise<void> {
    if (this.started) return;
    this.started = true;

    // Phase 5:注入 --pinconsole-* token styles 到 :root(必须在任何 UI 创建前)
    injectTokenStyles();

    const apiBase = this.inferApiBase();
    const wsBase = this.inferWsBase();

    try {
      this.session = await this.obtainSession(apiBase);
    } catch (e) {
      sdkLogger.error('session_init_failed', { error: String(e) });
      return;
    }

    // 1i：采集 fingerprint（canvas + WebGL + screen + tz）
    this.fingerprint = collectFingerprint();
    sdkLogger.info('fingerprint', { hash: this.fingerprint.combined_hash });

    // 1l:从服务端查 consent 状态
    await this.loadConsent(apiBase);

    // 1l:根据 consentMode + consentAccepted 决定是否启动 surveillance
    const shouldCollect = this.shouldCollectSurveillance();
    if (!shouldCollect && this.config.consentMode === 'opt-in') {
      // opt-in 模式下未同意 → 显示 banner 等用户决策
      this.showConsentBannerIfNeeded(apiBase);
    }

    const hello = {
      visitor_id: this.session.visitorId,
      session_id: this.session.sessionId,
      sdk_version: SDK_VERSION,
      capabilities: {
        events: shouldCollect ? ['rrweb'] : [],
        co_browsing: true,
        recording: shouldCollect,
      },
      // 1i：fingerprint 作为 hello 额外字段上报
      fingerprint: this.fingerprint,
    } as HelloPayload & { fingerprint: NonNullable<MarketingMonitorSDK['fingerprint']> };

    this.transport = new WSTransport({
      endpoint: `${wsBase}/ws/visitor`,
      hello,
      onStatusChange: (s) => this.onStatus(s),
      onError: (e) => sdkLogger.warn('transport_error', { error: String(e) }),
      onMessage: (env) => this.onMessage(env),
    });
    this.transport.start();

    // 1e/1g：命令处理器（cursor_highlight / click / scroll / fill_input / navigate / show_popup / chat_message）
    // 1g：聊天 widget（右下角浮动气泡）
    this.chatWidget = new ChatWidget({
      onSend: (content) => {
        // 访客发消息：POST /api/sessions/:id/visitor-message（公开端点，
        // server 固定 sender="visitor"；旧 /messages 端点挂在 admin AuthMiddleware
        // 后,visitor 无 admin cookie 会 403）
        const apiBase = this.inferApiBase();
        void fetch(`${apiBase}/api/sessions/${this.session!.sessionId}/visitor-message`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ content }),
          credentials: 'include',
        }).catch((e) => sdkLogger.warn('chat_send_failed', { error: String(e) }));
      },
      onFetchMessages: async (sinceId) => {
        // 拉历史聊天消息。GET /messages 挂在 admin AuthMiddleware 下;
        // 单租户同源部署下(admin SPA + visitor SDK 同 origin),浏览器共享
        // cookie,本端点能 200。多租户/跨域部署下会 403,fetchMessages 静默
        // 返回空,只依赖 WS 实时推送(详见 server/internal/api/router.go 注释)。
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
      onControlStart: () => {
        // 1l:co-browsing 接管横幅(GDPR Art.22 透明度)
        if (this.config.showCoBrowseBanner) {
          showCoBrowseBanner({
            onExit: () => this.releaseControl(),
          });
        }
      },
      onReleased: () => {
        sdkLogger.info('cobrowse_released', { source: 'visitor' });
        // 1l:移除横幅
        removeCoBrowseBanner();
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

    // 1l:仅当 consent 通过才启动 surveillance
    if (shouldCollect) {
      await this.startCollectors();
    }

    sdkLogger.info('sdk_started', {
      version: SDK_VERSION,
      session_id: this.session.sessionId,
      visitor_id: this.session.visitorId,
      api_base: apiBase,
      ws_base: wsBase,
      consent: String(this.consentAccepted),
      consent_mode: String(this.config.consentMode),
    });
  }

  /** 1l:启动 surveillance 采集器(rrweb + screenshot)。 */
  private async startCollectors(): Promise<void> {
    if (this.rrweb || this.screenshot) return; // 已启动
    // rrweb 采集器：默认 mask 所有输入(隐私基线)。
    // 部署方显式开启 unmaskInputs 后,展示访客输入文本,但 password 始终脱敏。
    const unmask = this.config.unmaskInputs === true;
    this.rrweb = new RRWebCollector(
      (e) => this.batch?.push(e),
      unmask
        ? {
            maskAllInputs: false,
            // 即使开启 unmask,password 仍强制脱敏(安全/合规底线)
            maskInputOptions: { password: true },
          }
        : {},
    );
    try {
      await this.rrweb.start();
    } catch (e) {
      sdkLogger.warn('rrweb_start_failed', { error: String(e) });
    }

    // 选择性截图：检测到 canvas/WebGL/iframe 才启动
    this.screenshot = new ScreenshotCollector((e) => this.batch?.push(e));
    this.screenshot.start();
  }

  /** 1l:从服务端查 consent 状态。 */
  private async loadConsent(apiBase: string): Promise<void> {
    if (!this.fingerprint) return;
    try {
      const resp = await fetch(
        `${apiBase}/api/privacy/consent?fingerprint=${encodeURIComponent(this.fingerprint.combined_hash)}`,
      );
      if (!resp.ok) return;
      const data = await resp.json();
      if (data.found) {
        this.consentAccepted = !!data.accepted;
      } else {
        this.consentAccepted = null; // 未记录
      }
    } catch (e) {
      sdkLogger.warn('consent_load_failed', { error: String(e) });
    }
  }

  /** 1l:根据 consentMode + consentAccepted 决定是否采集 surveillance。 */
  private shouldCollectSurveillance(): boolean {
    switch (this.config.consentMode) {
      case 'always-on':
        return true;
      case 'always-off':
        return false;
      case 'opt-out':
        // 默认采集,除非显式拒绝
        return this.consentAccepted !== false;
      case 'opt-in':
      default:
        // 默认不采集,除非显式同意
        return this.consentAccepted === true;
    }
  }

  /** 1l:opt-in 模式下未同意时显示 banner。 */
  private showConsentBannerIfNeeded(apiBase: string): void {
    if (this.consentAccepted !== null) return; // 已有记录(接受或拒绝)
    showConsentBanner({
      text: this.config.consentBannerText,
      onAccept: async () => {
        this.consentAccepted = true;
        // POST 到服务端持久化
        try {
          await fetch(`${apiBase}/api/privacy/consent`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              fingerprint: this.fingerprint!.combined_hash,
              accepted: true,
            }),
          });
        } catch (e) {
          sdkLogger.warn('consent_persist_failed', { error: String(e) });
        }
        // 启动 surveillance
        await this.startCollectors();
      },
      onReject: async () => {
        this.consentAccepted = false;
        try {
          await fetch(`${apiBase}/api/privacy/consent`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              fingerprint: this.fingerprint!.combined_hash,
              accepted: false,
            }),
          });
        } catch (e) {
          sdkLogger.warn('consent_persist_failed', { error: String(e) });
        }
      },
    });
  }

  /** 1l:访客点退出按钮触发(等同 ESC 三连)。 */
  private releaseControl(): void {
    // 调 commandHandler 触发 release_control 流程
    // CommandHandler.onReleased 已绑定,直接调用会绕过内部清理
    // 简化:发一个 release 事件给后端,等后端广播 release_control
    this.transport?.sendEvent({
      type: 'rrweb',
      ts: Date.now(),
      rrweb: {
        type: 99,
        timestamp: Date.now(),
        data: { kind: 'release_control', source: 'visitor_button' },
      },
    });
    // 直接本地清理(不等服务端确认)
    removeCoBrowseBanner();
  }

  /** 1l:公开 API — 访客/部署方手动设置 consent。 */
  async setConsent(accepted: boolean): Promise<void> {
    this.consentAccepted = accepted;
    const apiBase = this.inferApiBase();
    try {
      await fetch(`${apiBase}/api/privacy/consent`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          fingerprint: this.fingerprint?.combined_hash ?? '',
          accepted,
        }),
      });
    } catch (e) {
      sdkLogger.warn('set_consent_persist_failed', { error: String(e) });
    }
    if (accepted) {
      removeConsentBanner();
      await this.startCollectors();
    } else {
      // 撤回同意 → 停止 surveillance
      this.rrweb?.stop();
      this.rrweb = null;
      this.screenshot?.stop();
      this.screenshot = null;
    }
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
      sdkLogger.debug('transport_status', { status: String(s) });
    }
  }

  private onMessage(env: unknown): void {
    if (this.config.debug) {
      // eslint-disable-next-line no-console
      sdkLogger.debug('incoming_message');
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
