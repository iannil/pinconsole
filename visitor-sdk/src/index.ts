// marketing-monitor 访客 SDK 入口。
//
// 切片 1a：仅自动初始化 + console.log 验证加载链路。
// 切片 1b：WebSocket 连接 + 上行鼠标/点击事件。
// 切片 1c：rrweb 全量采集。
// 切片 1d：录像归档（与后端配合）。
// 切片 1e-f：双向通道、co-browsing 命令执行。

import { resolveConfig, type VisitorConfig } from './config';

class MarketingMonitorSDK {
  private config: VisitorConfig;
  private startedAt: number;

  constructor() {
    this.config = resolveConfig();
    this.startedAt = Date.now();
  }

  /** 启动 SDK。DOMContentLoaded 后自动调用；也可显式调用。 */
  start(): void {
    const endpoint = this.config.endpoint ?? this.inferEndpoint();
    // eslint-disable-next-line no-console
    console.log('[marketing-monitor] SDK loaded', {
      version: '0.1.0',
      endpoint,
      tenantId: this.config.tenantId,
      pageId: this.config.pageId,
      enableRecording: this.config.enableRecording,
      startedAt: new Date(this.startedAt).toISOString(),
    });
    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.debug('[marketing-monitor] full config', this.config);
    }
  }

  /** 推断默认 WebSocket 端点。 */
  private inferEndpoint(): string {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${proto}//${location.host}/ws`;
  }
}

// 暴露到 window 便于调试与显式调用
const w = window as unknown as { __MM__?: MarketingMonitorSDK };
const sdk = new MarketingMonitorSDK();
w.__MM__ = sdk;

// 自动初始化：DOM 已 ready 时立即启动；否则等待 DOMContentLoaded
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => sdk.start(), { once: true });
} else {
  sdk.start();
}

export { MarketingMonitorSDK };
