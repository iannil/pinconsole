// 响应式 player sizing:监听容器尺寸 + 录制视口比例,动态算 player width/height
// 并通过 player.$set 热更新,避免 rrweb-player v2 默认 1024x576 (16:9) 与真实录制
// 视口比例(如 1579x904 ≈ 1.746:1)不一致导致的 letterbox / 视觉错位。
//
// 录制视口尺寸的来源:rrweb-player 创建的 iframe 元素的 width/height attribute。
// rrweb Replayer 内部用 meta event 的 width/height 设置 iframe 属性,即使 admin
// 没收到 meta event(服务端 ws.go:511-518 只缓存/下发 full snapshot,不缓存 meta),
// iframe 仍带有正确的录制视口尺寸 — 这是最可靠的信号源。
//
// 使用方:
//   const playerContainer = ref<HTMLDivElement | null>(null);
//   const { start, stop } = useResponsivePlayerSize(playerContainer, () => player);
//   // 创建 player 后:
//   start();
//   // 卸载/session 切换时:
//   stop();
//
// 适用于:
//   - admin/src/components/ReplayPlayer.vue(实时回放,可能缺 meta event)
//   - admin/src/views/ReplayViewer.vue(历史回放,events 完整)

import type { Ref } from 'vue';

export interface PlayerLike {
  // rrweb-player v2 提供 $set 热更新 props(width/height 等)
  $set?: (props: Record<string, unknown>) => void;
}

// 从 container 中找 rrweb-player 创建的渲染 iframe,读 width/height attribute。
// 返回 null 表示 iframe 还没创建或尺寸无效。
//
// 注:rrweb-player 可能创建多个 iframe(mirror + renderer),取第一个有合法
// width/height attribute 的。renderer iframe 的 width/height 来自 meta event,
// 就是访客录制时的实际 viewport。
function readRecordingDimsFromIframe(container: HTMLElement): { width: number; height: number } | null {
  const iframes = container.querySelectorAll('iframe');
  for (const iframe of Array.from(iframes)) {
    const w = Number(iframe.getAttribute('width'));
    const h = Number(iframe.getAttribute('height'));
    if (Number.isFinite(w) && Number.isFinite(h) && w > 0 && h > 0) {
      return { width: w, height: h };
    }
  }
  return null;
}

// 判断这批 mutations 是否需要重新 apply(只在 iframe 插入或 width/height 属性变化时)。
// 避免普通 DOM mutation(如 replay 过程中 iframe 内部 DOM 变化)频繁触发 sizing。
function shouldReapply(mutations: MutationRecord[]): boolean {
  for (const m of mutations) {
    if (m.type === 'attributes') {
      if (
        (m.attributeName === 'width' || m.attributeName === 'height') &&
        m.target instanceof HTMLIFrameElement
      ) {
        return true;
      }
    } else if (m.type === 'childList') {
      for (const node of Array.from(m.addedNodes)) {
        if (node.nodeName === 'IFRAME') return true;
        if (node instanceof Element && node.tagName !== 'IFRAME') {
          if (node.querySelector?.('iframe')) return true;
        }
      }
    }
  }
  return false;
}

/**
 * 启动响应式 sizing:监听 containerRef 尺寸 + iframe width/height,动态算
 * player width/height,通过 player.$set 热更新。
 *
 * - 容器更宽(宽高比 > 录制比):高度撑满,宽度按比例(左右留白)
 * - 容器更窄(宽高比 < 录制比):宽度撑满,高度按比例(上下留白)
 *
 * @param containerRef player 容器 ref
 * @param getPlayer    获取当前 player 实例的 getter(允许 null,内部判空)
 */
export function useResponsivePlayerSize(
  containerRef: Ref<HTMLElement | null>,
  getPlayer: () => PlayerLike | null,
) {
  let resizeObserver: ResizeObserver | null = null;
  let domObserver: MutationObserver | null = null;

  const apply = (): void => {
    const container = containerRef.value;
    const player = getPlayer();
    if (!container || !player) return;
    const cw = container.clientWidth;
    const ch = container.clientHeight;
    if (cw === 0 || ch === 0) return; // 容器未布局好
    const recDims = readRecordingDimsFromIframe(container);
    if (!recDims) return; // iframe 还没创建或尺寸无效
    const ratio = recDims.width / recDims.height;
    let newW: number;
    let newH: number;
    if (cw / ch > ratio) {
      // 容器更宽 → 高度撑满,宽度按比例
      newH = ch;
      newW = ch * ratio;
    } else {
      // 容器更窄 → 宽度撑满,高度按比例
      newW = cw;
      newH = cw / ratio;
    }
    player.$set?.({ width: Math.floor(newW), height: Math.floor(newH) });
  };

  /** 在 player 创建成功后调用:启动 ResizeObserver + DOM observer。 */
  function start(): void {
    if (resizeObserver) resizeObserver.disconnect();
    if (domObserver) domObserver.disconnect();
    const container = containerRef.value;
    if (!container) return;
    apply(); // 立即应用一次(若 iframe 已就绪)
    resizeObserver = new ResizeObserver(apply);
    resizeObserver.observe(container);
    // 监听 iframe 插入或 width/height 属性变化(visitor viewport 改变时重新算)
    domObserver = new MutationObserver((mutations) => {
      if (shouldReapply(mutations)) apply();
    });
    domObserver.observe(container, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['width', 'height'],
    });
  }

  /** 在 player 销毁/session 切换/组件卸载时调用:释放 observers。 */
  function stop(): void {
    resizeObserver?.disconnect();
    domObserver?.disconnect();
    resizeObserver = null;
    domObserver = null;
  }

  return { start, stop };
}
