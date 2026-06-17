// 事件批量器：100ms 或 50 events 阈值触发 flush
// 详见 docs/progress/2026-06-17-slice-1c-spec.md §事件批量

import type { EventPayload } from './proto/events';

export interface BatchOptions {
  /** 最大等待时间（毫秒），默认 100 */
  maxMs?: number;
  /** 最大累积事件数，默认 50 */
  maxEvents?: number;
}

export type BatchFlushCallback = (events: EventPayload[]) => void;

/**
 * Batch 累积事件，达 maxMs 或 maxEvents 时回调 flush。
 * 单线程安全（JS event loop）。
 */
export class Batch {
  private opts: Required<BatchOptions>;
  private onFlush: BatchFlushCallback;
  private buffer: EventPayload[] = [];
  private timer: ReturnType<typeof setTimeout> | null = null;

  constructor(onFlush: BatchFlushCallback, opts: BatchOptions = {}) {
    this.opts = {
      maxMs: opts.maxMs ?? 100,
      maxEvents: opts.maxEvents ?? 50,
    };
    this.onFlush = onFlush;
  }

  /** 加入一个事件。达阈值时自动 flush。 */
  push(event: EventPayload): void {
    this.buffer.push(event);
    if (this.buffer.length >= this.opts.maxEvents) {
      this.flush();
      return;
    }
    if (!this.timer) {
      this.timer = setTimeout(() => this.flush(), this.opts.maxMs);
    }
  }

  /** 立即 flush 缓冲。 */
  flush(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    if (this.buffer.length === 0) return;
    const items = this.buffer;
    this.buffer = [];
    this.onFlush(items);
  }

  /** 销毁，停止定时器。 */
  destroy(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.buffer = [];
  }
}
