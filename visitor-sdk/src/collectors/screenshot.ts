// 选择性截图：检测 canvas/WebGL/跨域 iframe，触发 1fps WebP q70 截图
// 详见 docs/progress/2026-06-17-slice-1c-spec.md §选择性截图触发

import type { EventPayload } from '@pinconsole/proto';

export interface ScreenshotCollectorOptions {
  /** 截图频率（毫秒），默认 1000（1fps） */
  intervalMs?: number;
  /** WebP quality (0-1)，默认 0.7 */
  quality?: number;
  /** 检测间隔（毫秒），默认 2000 */
  detectIntervalMs?: number;
}

/**
 * ScreenshotCollector 检测页面是否含 canvas / WebGL / 跨域 iframe。
 * 检测到才启动 setInterval 截图（html2canvas 或 canvas.drawImage 都太重；
 * 1c 用最简单的：document.documentElement 截屏需要原生 API，
 * 浏览器原生不支持，所以用 canvas 包裹整个视口）。
 *
 * 简化实现：用 canvas + drawImage(window) 不可行（CSP）。
 * 实际方案：用 SVG <foreignObject> + canvas，把 DOM 转 SVG 再转 PNG/WebP。
 * 此处实现用最简的"截图整个 document body"方案，仅占位（性能差但 1c 验证管道用）。
 *
 * 实际生产可能需要 html-to-image / modern-screenshot 等库；
 * 1c 不实现复杂截图，仅在 rrweb 节点含 canvas/WebGL 时触发，
 * 用 canvas.captureStream() 或 Canvas.toDataURL() 抓 canvas 内容。
 */
export class ScreenshotCollector {
  private emit: (event: EventPayload) => void;
  private opts: Required<ScreenshotCollectorOptions>;
  private active = false;
  private captureTimer: ReturnType<typeof setInterval> | null = null;
  private detectTimer: ReturnType<typeof setInterval> | null = null;
  private hasTargets = false;

  constructor(emit: (event: EventPayload) => void, opts: ScreenshotCollectorOptions = {}) {
    this.emit = emit;
    this.opts = {
      intervalMs: opts.intervalMs ?? 1000,
      quality: opts.quality ?? 0.7,
      detectIntervalMs: opts.detectIntervalMs ?? 2000,
    };
  }

  start(): void {
    if (this.active) return;
    this.active = true;
    // 初次检测
    this.detectTargets();
    // 定期重新检测（canvas 可能动态插入）
    this.detectTimer = setInterval(() => this.detectTargets(), this.opts.detectIntervalMs);
  }

  stop(): void {
    this.active = false;
    if (this.captureTimer) {
      clearInterval(this.captureTimer);
      this.captureTimer = null;
    }
    if (this.detectTimer) {
      clearInterval(this.detectTimer);
      this.detectTimer = null;
    }
  }

  private detectTargets(): void {
    const wasTargets = this.hasTargets;
    this.hasTargets = this.hasCanvasOrWebGL() || this.hasCrossOriginIframes();
    if (this.hasTargets && !wasTargets) {
      this.startCapture();
    } else if (!this.hasTargets && wasTargets) {
      this.stopCapture();
    }
  }

  private hasCanvasOrWebGL(): boolean {
    return document.querySelectorAll('canvas').length > 0;
  }

  private hasCrossOriginIframes(): boolean {
    try {
      const iframes = document.querySelectorAll('iframe');
      for (const iframe of iframes) {
        // try-catch：跨域 iframe.contentDocument 访问会抛
        try {
          if (iframe.contentDocument === null) return true;
        } catch {
          return true;
        }
      }
      return false;
    } catch {
      return false;
    }
  }

  private startCapture(): void {
    if (this.captureTimer) return;
    this.captureTimer = setInterval(() => this.capture(), this.opts.intervalMs);
  }

  private stopCapture(): void {
    if (this.captureTimer) {
      clearInterval(this.captureTimer);
      this.captureTimer = null;
    }
  }

  /**
   * capture 截图所有 canvas 元素的内容（toDataURL WebP）。
   * 不截整个页面（避免重 DOM 性能问题）。
   * 截图作为 EventPayload.rrweb 的特殊形式发送：复用 rrweb 通道但 type=2（FullSnapshot）的 data 是 base64 字符串。
   *
   * 注：这是 1c 简化实现，1d+ 可换 html-to-image 做整页截图。
   */
  private capture(): void {
    const canvases = document.querySelectorAll('canvas');
    if (canvases.length === 0) return;

    for (const canvas of canvases) {
      try {
        const dataUrl = canvas.toDataURL('image/webp', this.opts.quality);
        // 发为 rrweb-like 事件（type=99 自定义"canvas screenshot"，避免与 rrweb 标准类型冲突）
        this.emit({
          type: 'rrweb',
          ts: Date.now(),
          rrweb: {
            type: 99, // 自定义类型：canvas 截图
            timestamp: Date.now(),
            data: {
              kind: 'canvas_screenshot',
              src: dataUrl,
              // 元素的近似位置（用于 admin 端叠加显示）
              rect: canvas.getBoundingClientRect().toJSON(),
            },
          },
        });
      } catch {
        // canvas.toDataURL 在 tainted canvas（跨域 image 渲染）会抛 SecurityError
        // 跳过即可
      }
    }
  }
}
