<script setup lang="ts">
// rrweb-player 实时回放组件（动态 import 避免进首屏 bundle）
// 详见 docs/progress/2026-06-17-slice-1c-spec.md §Admin 实时回放

import { ref, watch, onUnmounted, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import type { EventPayload } from '@pinconsole/proto';
import { useResponsivePlayerSize } from '../composables/useResponsivePlayerSize';

const { t } = useI18n();

// rrweb-player v2 的类型定义虽然 2.0.1 已提供,但 Player extends SvelteComponent
// 的 $set(addEvent 等)签名与我们的 composable PlayerLike 不直接兼容。
// 仍用最小化 interface + 透传 unknown 模式,保留动态 import 的代码分割。
type RRWebPlayerInstance = {
  addEvent?: (e: unknown) => void;
  // v2 支持 $set 热更新 props(width/height 等),用于响应式 sizing
  $set?: (props: Record<string, unknown>) => void;
  // v2 暴露 getReplayer 拿到内部 Replayer,用于兜底手动触发 handleResize
  // (admin 没收到 meta event 时让 iframe 显示)
  getReplayer?: () => { handleResize?: (dimension: { width: number; height: number }) => void } | null;
} & Record<string, unknown>;
type RRWebPlayerPack = {
  default: {
    new (opts: {
      target: HTMLElement;
      props: {
        events: unknown[];
        showDebug?: boolean;
        autoPlay?: boolean;
        skipInactive?: boolean;
        showController?: boolean;
        rootContext?: unknown;
        // alpha.20 起传 UNSAFE_replayCanvas:true 让 rrweb 把 allow-scripts 加进
        // iframe sandbox,避免回放含 <script> 的页面时刷屏 sandbox warning。
        UNSAFE_replayCanvas?: boolean;
      };
    }): RRWebPlayerInstance;
  };
};

const props = defineProps<{
  /** 初始化事件（含最近 full snapshot + 后续 incremental） */
  events: EventPayload[];
  /** 当前订阅 sessionID（变化时重建） */
  sessionId: string | null;
}>();

const containerRef = ref<HTMLDivElement | null>(null);
const loading = ref(true);
const errorMsg = ref<string | null>(null);
// 是否有足够 events(≥2)创建 player。否则显示 "等待访客交互" 提示。
const hasEnoughEvents = ref(false);
let player: RRWebPlayerInstance | null = null;
let pack: RRWebPlayerPack | null = null;
let iframeObserver: MutationObserver | null = null;
// rebuildPlayer 的并发抑制 token。onMounted + watch(events) 可能在初始挂载时
// 几乎同时触发 rebuildPlayer,rebuildPlayer 内有 `await loadPack()`,在 await 期间
// player 还是 null,两次调用都跳过 `if (player)` 清理,各自创建一个 Player Svelte
// 实例 → DOM 里出现两个 `.rr-player` 元素 → flex 布局把外框各缩一半,iframe 仍按
// 原比例缩放并 overflow → "大片空白 + 一小块录屏" 视觉 bug。
// 每次 rebuildPlayer 递增 token,await 完成后检查自己是否仍是最新;若不是则放弃。
let rebuildToken = 0;
// 响应式 sizing:覆盖 rrweb-player 默认 1024x576,按真实录制视口比例动态算
// player 外框尺寸,避免 letterbox / 视觉错位。详见 composable 注释。
const { start: startResponsiveSizing, stop: stopResponsiveSizing } =
  useResponsivePlayerSize(containerRef, () => player);

// 1c-fix:rrweb-player v2 alpha.18 的 iframe 默认 sandbox="allow-same-origin",
// 缺 allow-scripts 导致 Replayer 内部脚本被浏览器阻断(player 渲染空 iframe)。
// 用 MutationObserver 监听 iframe 创建,自动补 allow-scripts + 强制 reload。
// 注:改 sandbox attribute 后必须 reload iframe 才生效。
//
// 2026-06-21 扩展:除了 childList(新 iframe 插入),也监听 attribute 变化
// (sandbox / srcdoc)。rrweb-player 在创建 iframe 后通过 setAttribute 设这两个
// 属性,如果只在 childList 触发 patch,会错过首次 srcdoc load,留下 "Blocked
// script execution in sandboxed iframe" warning 噪音。attribute filter 让 patch
// 在 srcdoc 真正 load 之前生效。
function setupIframeSandboxPatch(root: HTMLElement): void {
  if (iframeObserver) iframeObserver.disconnect();
  const patch = (iframe: HTMLIFrameElement) => {
    const cur = iframe.getAttribute('sandbox') ?? '';
    if (!cur.includes('allow-scripts')) {
      iframe.setAttribute('sandbox', cur ? `${cur} allow-scripts` : 'allow-scripts');
      // reload 让新 sandbox 生效(srcdoc iframe reload via re-set srcdoc)
      const srcdoc = iframe.getAttribute('srcdoc');
      if (srcdoc) {
        iframe.removeAttribute('srcdoc');
        iframe.setAttribute('srcdoc', srcdoc);
      }
    }
  };
  // patch 现有
  root.querySelectorAll('iframe').forEach(patch);
  // 监听新 iframe + 后期 attribute 变化(sandbox/srcdoc)
  iframeObserver = new MutationObserver((mutations) => {
    for (const m of mutations) {
      if (m.type === 'attributes') {
        const target = m.target;
        if (target instanceof HTMLIFrameElement) patch(target);
        continue;
      }
      // childList
      m.addedNodes.forEach((n) => {
        if (n.nodeName === 'IFRAME') patch(n as HTMLIFrameElement);
        if (n instanceof Element) n.querySelectorAll?.('iframe').forEach(patch);
      });
    }
  });
  iframeObserver.observe(root, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['sandbox', 'srcdoc'],
  });
}

// 提取 rrweb 原生事件（FullSnapshot + IncrementalSnapshot 等）
function extractRRWeb(events: EventPayload[]): unknown[] {
  return events
    .filter((e) => e.type === 'rrweb' && e.rrweb)
    .map((e) => e.rrweb) as unknown[];
}

// 动态加载 rrweb-player(避免进首屏 bundle)
async function loadPack(): Promise<RRWebPlayerPack> {
  if (pack) return pack;
  pack = (await import('rrweb-player')) as unknown as RRWebPlayerPack;
  return pack;
}

async function rebuildPlayer() {
  if (!containerRef.value) return;
  // 拿到本次调用的 token。后续 await 完成后,如果 rebuildToken 已经被另一个
  // 更新的调用递增,说明我已被取代,直接放弃,避免双 mount 写出两个 .rr-player。
  const myToken = ++rebuildToken;
  loading.value = true;
  errorMsg.value = null;

  // **无条件**清空 DOM + 释放 observer(不放在 `if (player)` 守卫里)。
  // 原因:首次 mount 时 player 仍是 null,但若 onMounted 和 watch(events) 在
  // await 窗口内都调到 rebuildPlayer,两次都会跳过 if(player) 守卫,各自创建
  // 一个 Player Svelte 实例。无条件清空 + token 检查在 await 后,确保即便
  // 两次都跑到 new Ctor(),也只有一个能成功 append DOM。
  if (player) {
    stopResponsiveSizing();
    player = null;
  }
  containerRef.value.replaceChildren();

  if (props.events.length === 0) {
    if (myToken === rebuildToken) loading.value = false;
    return;
  }

  try {
    const playerPack = await loadPack();
    // await 完成后检查:如果期间有更新的 rebuildPlayer 调用接管,我必须放弃,
    // 避免和它同时 new Ctor() 导致 .player-container 里出现两个 .rr-player。
    if (myToken !== rebuildToken) return;
    const rrwebEvents = extractRRWeb(props.events);
    if (rrwebEvents.length === 0) {
      loading.value = false;
      hasEnoughEvents.value = false;
      return;
    }
    // rrweb Replayer 需要至少 2 events 才创建(full snapshot + 至少一个 meta/incremental)。
    // 只有 1 个 full snapshot 时会抛 "Replayer need at least 2 events"。
    // 这种情况发生在:访客刚同意 consent 触发 full snapshot,但还没产生交互(无 incremental)。
    // 等下一个 event 到来再创建 player(watch 会再触发 rebuildPlayer)。
    if (rrwebEvents.length < 2) {
      loading.value = false;
      hasEnoughEvents.value = false;  // 显示 "等待访客交互" 提示
      return;
    }
    hasEnoughEvents.value = true;
    const Ctor = playerPack.default;
    player = new Ctor({
      target: containerRef.value,
      props: {
        events: rrwebEvents,
        showDebug: false,
        autoPlay: true,
        skipInactive: true,
        // Phase 4:实时回放隐藏 rrweb-player 原生控制器
        // (live 模式不需要 play/pause/scrub;事件是被动推送的)
        showController: false,
        // alpha.20 默认 sandbox="allow-same-origin"(无 allow-scripts),
        // 回放含 <script> 的页面时每个 script 触发一次 sandbox warning。
        // UNSAFE_replayCanvas 同时打开 allow-scripts(名字唬人但 iframe 仍受
        // sandbox 隔离,且这是回放我们自己的访客录制,不是任意内容)。
        // 副作用:canvas/WebGL 录制内容也能正确回放(符合 PLAN.md 选择性截图策略)。
        UNSAFE_replayCanvas: true,
      },
    });
    // **关键**:player 成功创建后才标记 initialized。原代码在 watch callback 里
    // 同步 `if (player) initialized = true`,但 rebuildPlayer 是 async,同步检查时
    // player 还是 null,initialized 永远 false → 每个 WS 事件都触发完整 rebuild →
    // 浏览器对每次新建的 iframe 都报 sandbox warning(实测 123 条)。
    initialized = true;
    // rrweb-player v2 alpha 的 iframe sandbox 默认只设 allow-same-origin,
    // 导致 Replayer 内部脚本被浏览器阻断("frame is sandboxed and 'allow-scripts'
    // permission is not set")→ player 渲染空白。
    // 修复:用 MutationObserver 监听 iframe 创建,自动补 allow-scripts。
    setupIframeSandboxPatch(containerRef.value);
    // 启动响应式 sizing(覆盖 rrweb-player 默认 1024x576,从 iframe width/height 读真实录制视口)
    startResponsiveSizing();
  } catch (e) {
    errorMsg.value = (e as Error).message;
  } finally {
    // 只有本次调用仍是最新时才清 loading,避免被后续调用覆盖。
    if (myToken === rebuildToken) loading.value = false;
  }
}

// 新事件 append（不重建）
function appendEvents(events: EventPayload[]) {
  if (!player) return;
  const rrwebEvents = extractRRWeb(events);
  if (rrwebEvents.length === 0) return;
  try {
    // rrweb-player v2 的 addEvent 接受单个 eventWithTime。
    // 注意 v2 没有 append(批量) 方法 — 之前的 player.append 是 bug,
    // 静默吞掉 catch 导致 incremental events 永远不进 player。
    // 修复:逐个 addEvent。
    for (const ev of rrwebEvents) {
      (player as unknown as { addEvent?: (e: unknown) => void }).addEvent?.(ev);
    }
  } catch (e) {
    console.warn('[ReplayPlayer] append failed', e);
  }
}

// 监听 events 变化：首批重建,后续 append。
// 注意 rrweb Replayer 需 ≥ 2 events 才能实例化,所以 events<2 时 rebuildPlayer
// 内部跳过创建 player,这时不能标记 initialized=true,否则下次新 events 来时
// 走 appendEvents 分支(player=null 时 return),player 永远不创建。
// initialized 由 rebuildPlayer 内部在 player 成功创建后设置(不能在这里同步检查,
// 因为 rebuildPlayer 是 async,同步检查时 player 还是 null)。
let initialized = false;
watch(
  () => props.events,
  (newEvents, oldEvents) => {
    if (!initialized || !player) {
      rebuildPlayer();
      return;
    }
    if (newEvents.length > (oldEvents?.length ?? 0)) {
      const fresh = newEvents.slice(oldEvents?.length ?? 0);
      appendEvents(fresh);
    }
  },
  { deep: false },
);

// sessionId 变化时强制重建
watch(
  () => props.sessionId,
  () => {
    initialized = false;
    hasEnoughEvents.value = false;
    loading.value = true;
    errorMsg.value = null;
    rebuildPlayer();
  },
);

onMounted(() => rebuildPlayer());

onUnmounted(() => {
  if (containerRef.value) {
    containerRef.value.replaceChildren();
  }
  iframeObserver?.disconnect();
  iframeObserver = null;
  stopResponsiveSizing();
  player = null;
});

// 暴露给父组件：用于外部 append
defineExpose({ appendEvents });
</script>

<template>
  <div class="replay-player">
    <div v-if="loading" class="loading">{{ t('replay.loading') }}</div>
    <div v-else-if="errorMsg" class="error">{{ t('replay.play_failed') }}: {{ errorMsg }}</div>
    <div v-else-if="!hasEnoughEvents" class="loading">
      {{ t('replay.waiting_events') }}
    </div>
    <div ref="containerRef" class="player-container"></div>
  </div>
</template>

<style scoped>
.replay-player {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.loading,
.error {
  padding: 1rem;
  text-align: center;
  color: #909399;
  font-size: 0.85rem;
}
.error {
  color: #f56c6c;
}
.player-container {
  flex: 1;
  /* 响应式 sizing 下 player 外框已按容器+录制比例算好,
     通常不会超出。保留 auto 作为兜底(异常尺寸时仍可滚动)。 */
  overflow: auto;
  background: #f5f7fa;
  /* player 外框(rr-player)由 rrweb-player 设 inline width/height,
     flex 居中让 letterbox 平均分布在两侧而非顶左对齐 */
  display: flex;
  align-items: center;
  justify-content: center;
}

/* rrweb-player v2 alpha.18 bug:Replayer 创建的 mirror iframe 默认
   style="display:none; pointer-events:none",但实际它是用来渲染 snapshot
   DOM 的(不只是镜像)。强制 display:block 让用户看到回放内容。
   canvas.replayer-mouse-tail 仍正常显示鼠标轨迹。 */
.player-container :deep(iframe) {
  display: block !important;
  pointer-events: auto !important;
}

/* 不要给 iframe 设 width/height:100% — rrweb-player 通过 iframe 的
   width/height attribute(1913×904) + .replayer-wrapper 的 transform:scale
   来缩放。如果设 width:100% height:100%,iframe 会试撑满 .replayer-wrapper,
   而 wrapper 没有显式高度,height:100% 无法 resolve,浏览器 fallback 到
   iframe 默认 150px,导致 iframe 高度只有 150px 视觉压扁。
   旧 CSS hack(width:100% height:100%)是 1c 切片为旧 layout 加的,
   与当前 transform 缩放机制冲突,移除。 */
</style>
