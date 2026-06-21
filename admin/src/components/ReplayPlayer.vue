<script setup lang="ts">
// rrweb-player 实时回放组件（动态 import 避免进首屏 bundle）
// 详见 docs/progress/2026-06-17-slice-1c-spec.md §Admin 实时回放

import { ref, watch, onUnmounted, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import type { EventPayload } from '@pinconsole/proto';

const { t } = useI18n();

// rrweb-player 的类型定义在 alpha 版不完整，使用 unknown 透传
type RRWebPlayerInstance = { addEvent?: (e: unknown) => void } & Record<string, unknown>;
type RRWebPlayerPack = {
  default: {
    new (opts: {
      target: HTMLElement;
      props: {
        events: unknown[];
        showDebug?: boolean;
        autoPlay?: boolean;
        skipInactive?: boolean;
        // Phase 4:rrweb-player v2 alpha 支持 showController,
        // 设 false 隐藏原生控制器(实时模式由我们自己接管 UI)
        showController?: boolean;
        rootContext?: unknown;
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

// 动态加载 rrweb-player
async function loadPack(): Promise<RRWebPlayerPack> {
  if (pack) return pack;
  pack = (await import('rrweb-player')) as unknown as RRWebPlayerPack;
  return pack;
}

async function rebuildPlayer() {
  if (!containerRef.value) return;
  loading.value = true;
  errorMsg.value = null;

  // 销毁旧 player：用 replaceChildren 安全清空 DOM（无 XSS 风险）
  if (player) {
    try {
      containerRef.value.replaceChildren();
    } catch {
      // ignore
    }
    player = null;
  }

  if (props.events.length === 0) {
    loading.value = false;
    return;
  }

  try {
    const playerPack = await loadPack();
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
    const Player = playerPack.default;
    player = new Player({
      target: containerRef.value,
      props: {
        events: rrwebEvents,
        showDebug: false,
        autoPlay: true,
        skipInactive: true,
        // Phase 4:实时回放隐藏 rrweb-player 原生控制器
        // (live 模式不需要 play/pause/scrub;事件是被动推送的)
        showController: false,
      },
    });
    // rrweb-player v2 alpha 的 iframe sandbox 默认只设 allow-same-origin,
    // 导致 Replayer 内部脚本被浏览器阻断("frame is sandboxed and 'allow-scripts'
    // permission is not set")→ player 渲染空白。
    // 修复:用 MutationObserver 监听 iframe 创建,自动补 allow-scripts。
    setupIframeSandboxPatch(containerRef.value);
  } catch (e) {
    errorMsg.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

// 新事件 append（不重建）
function appendEvents(events: EventPayload[]) {
  if (!player) return;
  const rrwebEvents = extractRRWeb(events);
  if (rrwebEvents.length === 0) return;
  try {
    // rrweb-player v2 alpha 的 addEvent 接受单个 eventWithTime。
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
let initialized = false;
watch(
  () => props.events,
  (newEvents, oldEvents) => {
    if (!initialized || !player) {
      rebuildPlayer();
      // 只有 player 真创建后才标记 initialized,后续走 appendEvents
      if (player) initialized = true;
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
  overflow: auto;
  background: #f5f7fa;
}
/* rrweb-player 自带样式，给个最小高度 */
.player-container :deep(iframe),
.Player-wrapper {
  width: 100%;
  height: 100%;
  border: 0;
}

/* rrweb-player v2 alpha.18 bug:Replayer 创建的 mirror iframe 默认
   style="display:none; pointer-events:none",但实际它是用来渲染 snapshot
   DOM 的(不只是镜像)。强制 display:block 让用户看到回放内容。
   canvas.replayer-mouse-tail 仍正常显示鼠标轨迹。 */
.player-container :deep(iframe) {
  display: block !important;
  pointer-events: auto !important;
}
</style>
