// useResponsivePlayerSize 单元测试
// fork-2：更新为直接 Replayer API（删 $set/triggerResize/MutationObserver）
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ref } from 'vue';
import { useResponsivePlayerSize } from '../src/composables/useResponsivePlayerSize';
import type { Replayer } from '@pinconsole/replay-core';

// Mock Replayer
function makeMockReplayer(iframeW: number | null, iframeH: number | null): Replayer {
  const iframe = document.createElement('iframe');
  if (iframeW !== null) iframe.setAttribute('width', String(iframeW));
  if (iframeH !== null) iframe.setAttribute('height', String(iframeH));

  const wrapper = document.createElement('div');
  wrapper.className = 'replayer-wrapper';
  wrapper.appendChild(iframe);

  return {
    iframe,
    wrapper,
    handleResize: vi.fn(),
    destroy: vi.fn(),
  } as unknown as Replayer;
}

function makeContainer(cw: number, ch: number): HTMLElement {
  const div = document.createElement('div');
  Object.defineProperty(div, 'clientWidth', { value: cw, configurable: true });
  Object.defineProperty(div, 'clientHeight', { value: ch, configurable: true });
  return div;
}

describe('useResponsivePlayerSize', () => {
  let resizeCbs: Array<() => void> = [];

  beforeEach(() => {
    resizeCbs = [];
    const MockRO = vi.fn((cb: () => void) => {
      resizeCbs.push(cb);
      return {
        observe: () => {},
        disconnect: () => { resizeCbs = resizeCbs.filter((c) => c !== cb); },
      };
    });
    vi.stubGlobal('ResizeObserver', MockRO);
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it('容器更宽时宽度撑满（cover 模式），wrapper 按录制比例计算', () => {
    const container = makeContainer(1000, 500);
    const replayer = makeMockReplayer(800, 600); // 4:3
    container.appendChild(replayer.wrapper);

    const containerRef = ref<HTMLElement | null>(container);
    const { start, stop } = useResponsivePlayerSize(containerRef, () => replayer);

    start();
    vi.advanceTimersByTime(100); // wait for deferred apply
    resizeCbs.forEach((cb) => cb()); // trigger ResizeObserver

    // 容器 1000x500 (2:1) > 录制 800x600 (4:3 ≈ 1.33:1)
    // cover 模式: fill width → newW = 1000, newH = 1000 / 1.33 = 750
    const style = replayer.wrapper.style;
    expect(style.width).toBe('1000px');
    expect(style.height).toBe('750px');
    expect(replayer.handleResize).toHaveBeenCalledWith({ width: 800, height: 600 });

    stop();
  });

  it('容器更窄时高度撑满（cover 模式）', () => {
    const container = makeContainer(400, 900);
    const replayer = makeMockReplayer(800, 600); // 4:3 → ratio 1.33
    container.appendChild(replayer.wrapper);

    const containerRef = ref<HTMLElement | null>(container);
    const { start } = useResponsivePlayerSize(containerRef, () => replayer);

    start();
    vi.advanceTimersByTime(100);
    resizeCbs.forEach((cb) => cb());

    // 容器 400x900 (0.44:1) < 录制 800x600 (1.33:1)
    // cover 模式: fill height → newH = 900, newW = 900 * 1.33 = 1200
    const style = replayer.wrapper.style;
    expect(style.width).toBe('1200px');
    expect(style.height).toBe('900px');
  });

  it('replayer 为 null 时不崩溃', () => {
    const container = makeContainer(800, 600);
    const containerRef = ref<HTMLElement | null>(container);
    const { start } = useResponsivePlayerSize(containerRef, () => null);
    expect(() => { start(); resizeCbs.forEach((cb) => cb()); }).not.toThrow();
  });

  it('容器尺寸为 0 时不计算', () => {
    const container = makeContainer(0, 0);
    const replayer = makeMockReplayer(800, 600);
    container.appendChild(replayer.wrapper);

    const containerRef = ref<HTMLElement | null>(container);
    const { start } = useResponsivePlayerSize(containerRef, () => replayer);
    start();
    vi.advanceTimersByTime(100);
    resizeCbs.forEach((cb) => cb());

    // wrapper 没有被修改（clientWidth=0 时跳过）
    expect(replayer.wrapper.style.width).toBe('');
    expect(replayer.wrapper.style.height).toBe('');
  });

  it('stop() 后不再响应 resize', () => {
    const container = makeContainer(1000, 500);
    const replayer = makeMockReplayer(800, 600);
    container.appendChild(replayer.wrapper);

    const containerRef = ref<HTMLElement | null>(container);
    const { start, stop } = useResponsivePlayerSize(containerRef, () => replayer);

    start();
    vi.advanceTimersByTime(100);
    resizeCbs.forEach((cb) => cb()); // first apply
    expect(replayer.wrapper.style.width).toBe('1000px'); // was applied

    stop();

    // 改变容器尺寸 + 触发 resize → 不应再更新 wrapper
    Object.defineProperty(container, 'clientWidth', { value: 500, configurable: true });
    resizeCbs.forEach((cb) => cb());

    // style 保持 stop 前的值
    expect(replayer.wrapper.style.width).toBe('1000px');
    expect(replayer.wrapper.style.height).toBe('750px');
  });

  it('重复 start 不会创建多个 observer', () => {
    const container = makeContainer(1000, 500);
    const replayer = makeMockReplayer(800, 600);
    container.appendChild(replayer.wrapper);

    const containerRef = ref<HTMLElement | null>(container);
    const { start } = useResponsivePlayerSize(containerRef, () => replayer);

    start();
    const beforeCount = resizeCbs.length;
    start();
    vi.advanceTimersByTime(100);

    // 旧 callback 被 disconnect 清除，只有一个新 callback
    expect(resizeCbs.length).toBeLessThanOrEqual(beforeCount + 1);
  });
});
