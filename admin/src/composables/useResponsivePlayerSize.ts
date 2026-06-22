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
  // v2 暴露 getReplayer 拿到内部 Replayer 实例(用于兜底手动触发 handleResize)
  getReplayer?: () => { handleResize?: (dimension: { width: number; height: number }) => void } | null;
  // v2 暴露 triggerResize:用当前 width/height props 重算 .replayer-wrapper 的
  // transform:scale。必须在 $set({width,height}) 之后调用 —— rrweb-player 的响应式
  // 块(rrweb-player.js:14873)只在 width/height 变化时更新 .rr-player 外框的 inline
  // 尺寸,**不会**重算 wrapper scale(updateScale 只在 replayer 'resize' 事件或全屏时跑)。
  // 不手动触发 triggerResize,$set 改了外框尺寸但 scale 停留在上一次的值 → 外框与
  // 内容缩放不一致(如外框 707×483 但 scale 仍是 961 宽时的 0.617)。
  triggerResize?: () => void;
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

// 从 iframe.contentWindow 读 innerWidth/innerHeight。用于 iframe 没有
// width/height attribute 时的兜底(meta event 缺失导致 handleResize 没触发)。
//
// iframe 即使 display:none,contentWindow.innerWidth 仍可读(布局视口,非视觉视口)。
// 这些值反映 rrweb 写入 iframe.contentDocument 的 DOM 渲染视口,等同访客录制时的 viewport。
// 需要 sandbox 含 allow-same-origin(rrweb-player 默认设置),否则跨域被拦。
function readRecordingDimsFromContentWindow(container: HTMLElement): { width: number; height: number } | null {
  const iframe = container.querySelector('iframe');
  const cw = iframe?.contentWindow?.innerWidth;
  const ch = iframe?.contentWindow?.innerHeight;
  if (typeof cw === 'number' && typeof ch === 'number' && cw > 0 && ch > 0) {
    return { width: cw, height: ch };
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
    let recDims = readRecordingDimsFromIframe(container);
    if (!recDims) {
      // iframe 没有 width/height attribute,通常是 admin 没收到 meta event
      // (服务端旧版只缓存 full snapshot 不缓存 meta,或 meta TTL 过期)。
      // 兜底:从 iframe.contentWindow 读真实视口尺寸,手动触发 replayer.handleResize,
      // 让 rrweb-player 内部把 iframe 设为 display:inherit + 写入 width/height attribute。
      // 后续 apply() 再走正常路径读 attribute。
      const fallbackDims = readRecordingDimsFromContentWindow(container);
      if (fallbackDims) {
        const replayer = player.getReplayer?.();
        replayer?.handleResize?.(fallbackDims);
        // handleResize 同步设了 attribute,再读一次
        recDims = readRecordingDimsFromIframe(container) ?? fallbackDims;
      }
    }
    if (!recDims) return; // 仍无尺寸,放弃此次 apply
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
    // **关键**:$set 只更新 .rr-player 外框 inline 尺寸,不重算 wrapper 的
    // transform:scale。必须显式 triggerResize 用新 width/height 重算 scale,
    // 否则改尺寸(如进入/退出协助导致容器变宽变窄)后内容缩放与外框不一致,
    // 表现为"退出协助后录屏没占满外框 / 外框没占满容器"。
    player.triggerResize?.();
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
