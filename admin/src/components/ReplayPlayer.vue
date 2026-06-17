<script setup lang="ts">
// rrweb-player 实时回放组件（动态 import 避免进首屏 bundle）
// 详见 docs/progress/2026-06-17-slice-1c-spec.md §Admin 实时回放

import { ref, watch, onUnmounted, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import type { EventPayload } from '../proto/events';

const { t } = useI18n();

// rrweb-player 的类型定义在 alpha 版不完整，使用 unknown 透传
type RRWebPlayerInstance = { append: (events: unknown[]) => void } & Record<string, unknown>;
type RRWebPlayerPack = {
  default: {
    new (opts: {
      target: HTMLElement;
      props: {
        events: unknown[];
        showDebug?: boolean;
        autoPlay?: boolean;
        skipInactive?: boolean;
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
let player: RRWebPlayerInstance | null = null;
let pack: RRWebPlayerPack | null = null;

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
      return;
    }
    const Player = playerPack.default;
    player = new Player({
      target: containerRef.value,
      props: {
        events: rrwebEvents,
        showDebug: false,
        autoPlay: true,
        skipInactive: true,
      },
    });
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
    // rrweb-player v2 alpha 的 append 接受 array of events
    player.append(rrwebEvents);
  } catch (e) {
    console.warn('[ReplayPlayer] append failed', e);
  }
}

// 监听 events 变化：首批重建，后续 append
let initialized = false;
watch(
  () => props.events,
  (newEvents, oldEvents) => {
    if (!initialized || !player) {
      rebuildPlayer();
      initialized = true;
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
    rebuildPlayer();
  },
);

onMounted(() => rebuildPlayer());

onUnmounted(() => {
  if (containerRef.value) {
    containerRef.value.replaceChildren();
  }
  player = null;
});

// 暴露给父组件：用于外部 append
defineExpose({ appendEvents });
</script>

<template>
  <div class="replay-player">
    <div v-if="loading" class="loading">{{ t('replay.loading') }}</div>
    <div v-else-if="errorMsg" class="error">{{ t('replay.play_failed') }}: {{ errorMsg }}</div>
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
</style>
