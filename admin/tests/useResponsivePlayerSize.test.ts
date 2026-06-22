// useResponsivePlayerSize 单元测试
// 锁定响应式 sizing 行为,防止 rrweb-player 默认 1024x576 比例 bug 回归。
//
// 关键设计点:
// - 录制视口尺寸从 rrweb-player 创建的 iframe 的 width/height attribute 读取
//   (而非从 meta event,因为服务端 ws.go 只下发 full snapshot,不下发 meta)
// - MutationObserver 监听 iframe 插入 + width/height 属性变化
// - ResizeObserver 监听容器尺寸变化

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ref } from 'vue';
import { useResponsivePlayerSize } from '../src/composables/useResponsivePlayerSize';

// 工具:造一个带 width/height attribute 的 iframe 元素
function makeIframe(width: number | null, height: number | null): HTMLIFrameElement {
  const iframe = document.createElement('iframe');
  if (width !== null) iframe.setAttribute('width', String(width));
  if (height !== null) iframe.setAttribute('height', String(height));
  return iframe;
}

// 工具:造一个有 clientWidth/clientHeight 的 mock container
function makeContainer(cw: number, ch: number): HTMLElement {
  const div = document.createElement('div');
  Object.defineProperty(div, 'clientWidth', { value: cw, configurable: true });
  Object.defineProperty(div, 'clientHeight', { value: ch, configurable: true });
  return div;
}

// 工具:在 container 内查找 iframe(走真实 querySelector)
// 因为我们在测试里把 iframe 真插入到 container 中。

describe('useResponsivePlayerSize', () => {
  let resizeCbs: Array<() => void> = [];
  let mutationCbs: Array<(mutations: MutationRecord[]) => void> = [];
  let observeTargets: HTMLElement[] = [];

  beforeEach(() => {
    resizeCbs = [];
    mutationCbs = [];
    observeTargets = [];

    const MockRO = vi.fn((cb: () => void) => {
      resizeCbs.push(cb);
      return {
        observe: (target: HTMLElement) => observeTargets.push(target),
        disconnect: () => {
          resizeCbs = resizeCbs.filter((c) => c !== cb);
        },
      };
    });
    vi.stubGlobal('ResizeObserver', MockRO);

    const MockMO = vi.fn((cb: (mutations: MutationRecord[]) => void) => {
      mutationCbs.push(cb);
      return {
        observe: (target: HTMLElement) => observeTargets.push(target),
        disconnect: () => {
          mutationCbs = mutationCbs.filter((c) => c !== cb);
        },
      };
    });
    vi.stubGlobal('MutationObserver', MockMO);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  function fireResize(): void {
    resizeCbs.forEach((cb) => cb());
  }

  function fireMutation(mutations: MutationRecord[]): void {
    mutationCbs.forEach((cb) => cb(mutations));
  }

  it('start() 立即 apply:iframe 已就绪时算出正确尺寸', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080); // 16:9
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();

    // 容器 1024x768 (1.333),录制 16:9 (1.778),容器更窄
    // → 宽度撑满 1024,高度 1024/1.778 = 575.7 → 575
    expect($set).toHaveBeenCalledWith({ width: 1024, height: 576 });
  });

  it('apply 后调用 triggerResize 重算 wrapper scale($set 不会自己重算)', () => {
    // 回归:rrweb-player 的 $set({width,height}) 只更新 .rr-player 外框 inline 尺寸,
    // 不重算 .replayer-wrapper 的 transform:scale。必须显式 triggerResize,
    // 否则改尺寸(进入/退出协助、窗口 resize)后内容缩放与外框不一致。
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    const calls: string[] = [];
    const $set = vi.fn(() => calls.push('set'));
    const triggerResize = vi.fn(() => calls.push('resize'));
    const player = { $set, triggerResize };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();

    expect($set).toHaveBeenCalledWith({ width: 1024, height: 576 });
    expect(triggerResize).toHaveBeenCalledTimes(1);
    // 顺序:必须先 $set(更新 width/height props)再 triggerResize(用新值重算 scale)
    expect(calls).toEqual(['set', 'resize']);
  });

  it('容器更宽(16:10)装下 4:3 录制 → 高度撑满', () => {
    const container = makeContainer(1920, 1080); // 16:9 ≈ 1.778
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1024, 768); // 4:3 ≈ 1.333
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();

    // 容器 1.778 > 录制 1.333,容器更宽 → 高度撑满 1080,宽度 1080*1.333=1440
    expect($set).toHaveBeenCalledWith({ width: 1440, height: 1080 });
  });

  it('iframe 未插入时 start() 不 apply,等待 MutationObserver', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    // 不 append iframe

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    expect($set).not.toHaveBeenCalled();

    // 模拟 iframe 后续插入
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    // 触发 MutationObserver callback(模拟 iframe 插入的 mutation)
    const fakeMutation: MutationRecord = {
      type: 'childList',
      target: container,
      addedNodes: [iframe] as unknown as NodeList,
      removedNodes: [] as unknown as NodeList,
      attributeName: null,
      attributeNamespace: null,
      oldValue: null,
      nextSibling: null,
      previousSibling: null,
    };
    fireMutation([fakeMutation]);

    expect($set).toHaveBeenCalledWith({ width: 1024, height: 576 });
  });

  it('iframe width/height attribute 变化时重新算(visitor resize)', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    expect($set).toHaveBeenLastCalledWith({ width: 1024, height: 576 });
    $set.mockClear();

    // visitor 把视口从 16:9 改成 4:3
    iframe.setAttribute('width', '1024');
    iframe.setAttribute('height', '768');

    const fakeMutation: MutationRecord = {
      type: 'attributes',
      target: iframe,
      attributeName: 'width',
      attributeNamespace: null,
      addedNodes: [] as unknown as NodeList,
      removedNodes: [] as unknown as NodeList,
      oldValue: '1920',
      nextSibling: null,
      previousSibling: null,
    };
    fireMutation([fakeMutation]);

    // 容器 1024x768 (1.333) == 录制 1024x768 (1.333),等比 → 1024x768
    expect($set).toHaveBeenLastCalledWith({ width: 1024, height: 768 });
  });

  it('容器 resize 时重新算', () => {
    const container = makeContainer(800, 600);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1600, 900); // 16:9
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    // 容器 800x600 (1.333),录制 1.778,容器更窄 → 宽 800,高 800/1.778=449.4 → 449
    expect($set).toHaveBeenLastCalledWith({ width: 800, height: 450 });
    $set.mockClear();

    // 模拟容器 resize 到 1920x1080
    Object.defineProperty(container, 'clientWidth', { value: 1920, configurable: true });
    Object.defineProperty(container, 'clientHeight', { value: 1080, configurable: true });
    fireResize();

    // 容器 1920x1080 (1.778) == 录制 1.778,等比
    expect($set).toHaveBeenLastCalledWith({ width: 1920, height: 1080 });
  });

  it('player 为 null 时不 apply(getter 守卫)', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    const { start } = useResponsivePlayerSize(containerRef, () => null);

    expect(() => start()).not.toThrow();
  });

  it('容器 0x0 不 apply(避免初始 0x0 闪现)', () => {
    const container = makeContainer(0, 0);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    expect($set).not.toHaveBeenCalled();
  });

  it('iframe width/height 缺失或 0 不 apply', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    // iframe 没有 width/height attribute
    const iframe = makeIframe(null, null);
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    expect($set).not.toHaveBeenCalled();
  });

  it('多个 iframe 时取第一个有合法尺寸的', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    // 第一个 iframe 无效(可能是 mirror iframe),第二个有效
    const bad = makeIframe(0, 0);
    const good = makeIframe(1920, 1080);
    container.appendChild(bad);
    container.appendChild(good);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    expect($set).toHaveBeenCalledWith({ width: 1024, height: 576 });
  });

  it('stop() 释放 observers,后续 mutation/resize 不再触发 apply', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start, stop } = useResponsivePlayerSize(containerRef, () => player);

    start();
    expect(resizeCbs).toHaveLength(1);
    expect(mutationCbs).toHaveLength(1);
    stop();
    expect(resizeCbs).toHaveLength(0);
    expect(mutationCbs).toHaveLength(0);
  });

  it('再次 start() 先释放旧 observer 再建新的', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    expect(resizeCbs).toHaveLength(1);
    expect(mutationCbs).toHaveLength(1);

    start(); // 再调一次
    // 应该还是各 1 个(disconnect 旧的 + 新建)
    expect(resizeCbs).toHaveLength(1);
    expect(mutationCbs).toHaveLength(1);
  });

  it('shouldReapply 内置过滤:非 iframe 相关 mutation 不触发 apply', () => {
    const container = makeContainer(1024, 768);
    const containerRef = ref<HTMLElement | null>(container);
    const iframe = makeIframe(1920, 1080);
    container.appendChild(iframe);

    const $set = vi.fn();
    const player = { $set };
    const { start } = useResponsivePlayerSize(containerRef, () => player);

    start();
    $set.mockClear();

    // 模拟一个非 iframe 的 DOM mutation(replay 过程中 iframe 内部 DOM 变化)
    const div = document.createElement('div');
    const fakeMutation: MutationRecord = {
      type: 'childList',
      target: iframe, // 但 addedNode 是 div,不是 iframe
      addedNodes: [div] as unknown as NodeList,
      removedNodes: [] as unknown as NodeList,
      attributeName: null,
      attributeNamespace: null,
      oldValue: null,
      nextSibling: null,
      previousSibling: null,
    };
    fireMutation([fakeMutation]);

    // 不应该重新 apply(div 插入不代表 iframe 尺寸变化)
    expect($set).not.toHaveBeenCalled();
  });
});
