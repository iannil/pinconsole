<script setup lang="ts">
// replay-core 实时回放组件：直接持有 Replayer 实例（钻穿方案，不经过 Svelte rrweb-player）
// fork-2：删除所有 Svelte/rrweb-player hack——MutationObserver sandbox 补丁、
// UNSAFE_replayCanvas、rebuildToken 竞态抑制、$set/triggerResize/getReplayer 代理层、
// requestAnimationFrame 等待——全部消解。
import { ref, watch, onUnmounted, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import type { EventPayload } from '@pinconsole/proto';
import { Replayer, type playerConfig } from '@pinconsole/replay-core';
import { useResponsivePlayerSize } from '../composables/useResponsivePlayerSize';

const { t } = useI18n();

const props = defineProps<{
  events: EventPayload[];
  sessionId: string | null;
}>();

const containerRef = ref<HTMLDivElement | null>(null);
const loading = ref(true);
const errorMsg = ref<string | null>(null);
const hasEnoughEvents = ref(false);

let replayer: Replayer | null = null;
let initialized = false;

// 响应式 sizing
const { start: startResponsiveSizing, stop: stopResponsiveSizing } =
  useResponsivePlayerSize(containerRef, () => replayer);

// 提取 rrweb 事件
function extractRRWeb(events: EventPayload[]): unknown[] {
  return events
    .filter((e) => e.type === 'rrweb' && e.rrweb)
    .map((e) => e.rrweb) as unknown[];
}

async function rebuildPlayer() {
  if (!containerRef.value) return;

  loading.value = true;
  errorMsg.value = null;

  // 清理旧 player
  if (replayer) {
    stopResponsiveSizing();
    replayer.destroy();
    replayer = null;
  }
  containerRef.value.replaceChildren();

  if (props.events.length === 0) {
    loading.value = false;
    return;
  }

  const rrwebEvents = extractRRWeb(props.events);
  if (rrwebEvents.length === 0) {
    loading.value = false;
    hasEnoughEvents.value = false;
    return;
  }

  // Replayer 需要至少 2 events（full snapshot + meta/incremental）
  if (rrwebEvents.length < 2) {
    loading.value = false;
    hasEnoughEvents.value = false;
    return;
  }

  hasEnoughEvents.value = true;

  try {
    const config: Partial<playerConfig> = {
      speed: 1,
      root: containerRef.value,
      skipInactive: true,
      showDebug: false,
      showController: false,
      triggerFocus: true,
      pauseAnimation: false,
    };

    replayer = new Replayer(rrwebEvents, config);
    initialized = true;

    // listen for finish → switch to live mode
    replayer.on('finish', () => {
      replayer?.startLive(Date.now() + 365 * 24 * 60 * 60 * 1000);
    });

    // 启动响应式 sizing
    startResponsiveSizing();
  } catch (e) {
    errorMsg.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

// 追加新事件
function appendEvents(events: EventPayload[]) {
  if (!replayer) return;
  const rrwebEvents = extractRRWeb(events);
  for (const ev of rrwebEvents) {
    replayer.addEvent(ev as never);
  }
}

// 监听 events：首批重建，后续 append
watch(
  () => props.events,
  (newEvents, oldEvents) => {
    if (!initialized || !replayer) {
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

// sessionId 变化 → 强制重建
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
  if (replayer) {
    stopResponsiveSizing();
    replayer.destroy();
    replayer = null;
  }
});

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

<style>
/* replay-core Replayer CSS（继承自 rrweb，放在全局作用域让 iframe 内部生效） */
.replayer-wrapper {
  transform-origin: top left;
  left: 50%;
  top: 50%;
  position: relative;
}
</style>

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
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Replayer iframe 必须可见；不设 width/height:100%（与 transform 缩放冲突） */
.player-container :deep(iframe) {
  display: block !important;
  pointer-events: auto !important;
}
</style>
