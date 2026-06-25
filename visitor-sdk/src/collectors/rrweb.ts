// rrweb 采集器：封装 rrweb v2 record() + 韧性
// 详见 docs/progress/2026-06-17-slice-1c-spec.md §SDK 韧性

import type { EventPayload, RRWebEvent } from '@pinconsole/proto';
import { sdkLogger } from '../logging';

// @pinconsole/replay-core 的 record 模块动态 import（与 PLAN.md "动态 import" 一致，
// 同时减小 SDK 首次加载体积）。SSR 安全（typeof window 检查）。
//
// 注意 replay-core(基于 rrweb 2.0.0-alpha.20) 把 takeFullSnapshot 从 top-level export 改为
// record.takeFullSnapshot(附加属性)。SDK 必须走 record.takeFullSnapshot,
// 否则 alpha.20+ 下静默返回 → 周期性 full snapshot 永远不发 → snapshot cache
// 过期(TTL 5min)→ admin subscribe 时只拿到 meta → events < 2 → player 不创建。
type RRWebRecordFn = ((opts: {
  emit: (event: RRWebEvent) => void;
  maskAllInputs?: boolean;
  maskInputOptions?: unknown;
  blockClass?: string | RegExp;
  ignoreClass?: string | RegExp;
  sampling?: unknown;
}) => () => void) & {
  takeFullSnapshot?: (isCheckout?: boolean) => void;
};

interface RRWebPack {
  record: RRWebRecordFn;
}

export interface RRWebCollectorOptions {
  /** 输入脱敏：maskAllInputs */
  maskAllInputs?: boolean;
  /** 输入脱敏：maskInputOptions（rrweb-snapshot MaskInputOptions） */
  maskInputOptions?: unknown;
  /** 跨域 iframe / canvas 元素的屏蔽 class */
  blockClass?: string;
  /** 隐藏 class（不记录但占位） */
  ignoreClass?: string;
  /** 最大重试次数（默认 3） */
  maxRetries?: number;
  /** 页面 visibility hidden 超过此时长（ms）后主动 takeFullSnapshot（默认 60s） */
  visibilityHiddenMs?: number;
}

/**
 * RRWebCollector 包装 rrweb v2 record()。
 *
 * 韧性：
 *   - record() 错误时 try/catch + 退避重试（最多 maxRetries 次）
 *   - page visibility 变化时（hidden → visible），如果 hidden > 60s，主动 takeFullSnapshot
 *   - 周期性 full snapshot：每 30s 或 50 incremental 触发（由调用方控制）
 *
 * emit 回调将 rrweb 事件转 EventPayload 后交给 batch。
 */
export class RRWebCollector {
  private emit: (event: EventPayload) => void;
  private opts: Required<RRWebCollectorOptions>;
  private stopFn: (() => void) | null = null;
  private pack: RRWebPack | null = null;
  private retries = 0;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private lastVisibleAt = Date.now();
  private visibilityHandler: (() => void) | null = null;
  private periodicFullTimer: ReturnType<typeof setInterval> | null = null;
  private incrementalSinceFull = 0;
  private active = false;

  constructor(emit: (event: EventPayload) => void, opts: RRWebCollectorOptions = {}) {
    this.emit = emit;
    this.opts = {
      maskAllInputs: opts.maskAllInputs ?? true,
      maskInputOptions: opts.maskInputOptions ?? {
        password: true,
        // 默认 mask 文本类输入；select/radio/checkbox 保留（业务上有意义）
        text: true,
        textarea: true,
        search: true,
        email: true,
        tel: true,
        url: true,
      },
      blockClass: opts.blockClass ?? 'mm-block',
      ignoreClass: opts.ignoreClass ?? 'mm-ignore',
      maxRetries: opts.maxRetries ?? 3,
      visibilityHiddenMs: opts.visibilityHiddenMs ?? 60_000,
    };
  }

  /** 启动采集（异步加载 rrweb）。 */
  async start(): Promise<void> {
    if (this.active) return;
    this.active = true;
    await this.loadPack();
    this.startInternal();
    this.attachVisibility();
    this.startPeriodicFull();
  }

  /** 停止采集。 */
  stop(): void {
    this.active = false;
    if (this.stopFn) {
      try {
        this.stopFn();
      } catch {
        // ignore
      }
      this.stopFn = null;
    }
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
    if (this.visibilityHandler) {
      document.removeEventListener('visibilitychange', this.visibilityHandler);
      this.visibilityHandler = null;
    }
    if (this.periodicFullTimer) {
      clearInterval(this.periodicFullTimer);
      this.periodicFullTimer = null;
    }
  }

  /** 主动触发 full snapshot（用于周期性 / visibility 恢复） */
  takeFullSnapshot(): void {
    if (!this.pack?.record?.takeFullSnapshot) return;
    try {
      this.pack.record.takeFullSnapshot();
      this.incrementalSinceFull = 0;
    } catch (e) {
      sdkLogger.warn('take_full_snapshot_failed', { error: String(e) });
    }
  }

  private async loadPack(): Promise<void> {
    if (this.pack) return;
    try {
      // 动态 import：让 rrweb 体积仅在使用时计入
      const mod = (await import('@pinconsole/replay-core')) as unknown as RRWebPack;
      this.pack = mod;
    } catch (e) {
      sdkLogger.error('rrweb_load_failed', { error: String(e) });
      throw e;
    }
  }

  private startInternal(): void {
    if (!this.pack?.record) return;
    try {
      this.stopFn = this.pack.record({
        emit: (event) => {
          // 包装为 EventPayload，交调用方 batch
          const payload: EventPayload = {
            type: 'rrweb',
            ts: event.timestamp,
            rrweb: event,
          };
          if (event.type === 2) {
            // FullSnapshot
            this.incrementalSinceFull = 0;
          } else if (event.type === 3) {
            this.incrementalSinceFull++;
          }
          this.emit(payload);
        },
        maskAllInputs: this.opts.maskAllInputs,
        maskInputOptions: this.opts.maskInputOptions as never,
        blockClass: this.opts.blockClass,
        ignoreClass: this.opts.ignoreClass,
      });
      this.retries = 0;
    } catch (e) {
      sdkLogger.warn('rrweb_record_failed', { error: String(e) });
      this.scheduleRetry();
    }
  }

  private scheduleRetry(): void {
    if (this.retries >= this.opts.maxRetries) {
      sdkLogger.error('rrweb_exhausted_retries');
      return;
    }
    this.retries++;
    const delay = 1000 * 2 ** (this.retries - 1);
    if (this.retryTimer) clearTimeout(this.retryTimer);
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      if (!this.active) return;
      this.startInternal();
    }, delay);
  }

  private attachVisibility(): void {
    this.visibilityHandler = () => {
      if (document.visibilityState === 'visible') {
        const hiddenDuration = Date.now() - this.lastVisibleAt;
        if (hiddenDuration > this.opts.visibilityHiddenMs) {
          // 长时间隐藏：DOM 可能已变化，强制 full snapshot
          this.takeFullSnapshot();
        }
        this.lastVisibleAt = Date.now();
      } else {
        this.lastVisibleAt = Date.now();
      }
    };
    document.addEventListener('visibilitychange', this.visibilityHandler);
  }

  private startPeriodicFull(): void {
    // 每 10s 检查;若自上次 full snapshot 后有新 incremental(>= 1)就续期。
    //
    // 2026-06-21:双重修复
    // 1. 间隔从 30s 缩到 10s — 减少 admin subscribe 错过续期窗口的概率。
    // 2. 触发条件从 ">= 50 incremental" 改为 ">= 1 incremental"。
    //    原条件是优化手段(满 50 才发全量,避免流量爆炸),但实际副作用:
    //    visitor 低频交互(每 10s < 50 个增量)时永远不触发周期续期,
    //    snapshot_cache TTL 到期后过期,admin subscribe 拿不到 FullSnapshot,
    //    rrweb Replayer 缺初始 DOM,iframe 空白。
    //    改成 >= 1 后:只要有变化就续期,静默时(incrementalSinceFull=0)不发,
    //    流量与活跃度成正比,避免静默 visitor 的无谓全量推送。
    // 配合 server snapshot_cache TTL 30min(原 5min),即使 visitor 短暂静默,
    // admin 后续 subscribe 也能拿到最近一次 full snapshot。
    this.periodicFullTimer = setInterval(
      () => {
        if (this.incrementalSinceFull >= 1) {
          this.takeFullSnapshot();
        }
      },
      10_000,
    );
  }
}
