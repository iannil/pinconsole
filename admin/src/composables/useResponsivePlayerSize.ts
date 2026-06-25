// 响应式 player sizing:监听容器尺寸 + 录制视口比例,动态算 player 尺寸。
//
// fork-2：改为直接操作 Replayer 实例（删 $set/triggerResize/getReplayer 代理层）。
// 录制视口尺寸从 replay-core Replayer 的 iframe width/height attribute 读取。
//
// 使用方:
//   const containerRef = ref<HTMLDivElement | null>(null);
//   const { start, stop } = useResponsivePlayerSize(containerRef, () => replayer);
//   start();  // player 创建后
//   stop();   // 卸载/session 切换时

import type { Ref } from 'vue';
import type { Replayer } from '@pinconsole/replay-core';

// 从 Replayer.wrapper 中找 iframe，读 width/height attribute
function readRecordingDims(replayer: Replayer): { width: number; height: number } | null {
  const iframe = replayer.iframe;
  if (!iframe) return null;
  const w = Number(iframe.getAttribute('width'));
  const h = Number(iframe.getAttribute('height'));
  if (Number.isFinite(w) && Number.isFinite(h) && w > 0 && h > 0) {
    return { width: w, height: h };
  }
  // fallback: contentWindow
  const cw = iframe.contentWindow?.innerWidth;
  const ch = iframe.contentWindow?.innerHeight;
  if (typeof cw === 'number' && typeof ch === 'number' && cw > 0 && ch > 0) {
    return { width: cw, height: ch };
  }
  return null;
}

export function useResponsivePlayerSize(
  containerRef: Ref<HTMLElement | null>,
  getReplayer: () => Replayer | null,
) {
  let resizeObserver: ResizeObserver | null = null;
  let deferredTimer: ReturnType<typeof setTimeout> | null = null;

  const apply = (): void => {
    const container = containerRef.value;
    const replayer = getReplayer();
    if (!container || !replayer) return;

    const cw = container.clientWidth;
    const ch = container.clientHeight;
    if (cw === 0 || ch === 0) return;

    const recDims = readRecordingDims(replayer);
    if (!recDims) return;

    const ratio = recDims.width / recDims.height;
    let newW: number;
    let newH: number;
    if (cw / ch > ratio) {
      newH = ch;
      newW = ch * ratio;
    } else {
      newW = cw;
      newH = cw / ratio;
    }

    // 直接设 wrapper 尺寸（不再走 $set + triggerResize）
    const wrapper = replayer.wrapper;
    if (wrapper) {
      wrapper.style.width = `${Math.floor(newW)}px`;
      wrapper.style.height = `${Math.floor(newH)}px`;
    }

    // let replay-core handle internal iframe scaling
    replayer.handleResize({ width: recDims.width, height: recDims.height });
  };

  function start(): void {
    if (resizeObserver) resizeObserver.disconnect();
    const container = containerRef.value;
    if (!container) return;

    // defer: wait for Replayer iframe to be ready
    deferredTimer = setTimeout(apply, 100);

    resizeObserver = new ResizeObserver(apply);
    resizeObserver.observe(container);
  }

  function stop(): void {
    resizeObserver?.disconnect();
    resizeObserver = null;
    if (deferredTimer) { clearTimeout(deferredTimer); deferredTimer = null; }
  }

  return { start, stop };
}
