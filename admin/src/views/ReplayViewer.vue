<script setup lang="ts">
// 单会话历史回放 + Custom Calm Chrome 控制栏
// fork-2：钻穿方案——直接持 Replayer 实例，删 Svelte rrweb-player 壳。
// 自建控制栏：Play/Pause + scrubber + time + speed + skip inactive。
import { ref, onMounted, watch, onUnmounted, computed } from 'vue';
import { useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import {
  PhArrowLeft,
  PhPlay,
  PhPause,
  PhSkipForward,
  PhWarningCircle,
  PhSpinner,
} from '@phosphor-icons/vue';
import { getSessionReplay, type RRWebEvent } from '../api/sessions';
import { Replayer, type playerConfig } from '@pinconsole/replay-core';
import { useResponsivePlayerSize } from '../composables/useResponsivePlayerSize';

const { t } = useI18n();
const route = useRoute();

const sessionId = ref<string>(String(route.params.session_id ?? ''));
const loading = ref(true);
const loadingMore = ref(false);
const error = ref<string | null>(null);
const total = ref(0);
const hasMore = ref(false);
const events = ref<RRWebEvent[]>([]);

const playerContainer = ref<HTMLDivElement | null>(null);
let replayer: Replayer | null = null;

const { start: startResponsiveSizing, stop: stopResponsiveSizing } =
  useResponsivePlayerSize(playerContainer, () => replayer);

// ===== 自建控制栏状态 =====
const isPlaying = ref(false);
const currentTimeMs = ref(0);
const totalTimeMs = ref(0);
const speed = ref(1);
const skipInactive = ref(true);
let timePoll: ReturnType<typeof setInterval> | null = null;

const speedOptions = [1, 2, 4, 8];

const progressPercent = computed(() =>
  totalTimeMs.value > 0 ? Math.min(100, (currentTimeMs.value / totalTimeMs.value) * 100) : 0,
);

function formatTime(ms: number): string {
  if (!ms || ms < 0) return '0:00';
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

// ===== 加载事件 =====
async function loadInitial() {
  loading.value = true;
  error.value = null;
  try {
    const resp = await getSessionReplay(sessionId.value, 0, 10000);
    events.value = resp.events ?? [];
    total.value = resp.total ?? 0;
    hasMore.value = resp.has_more ?? false;
    if (resp.has_more) loadMore();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function loadMore() {
  if (!hasMore.value || loadingMore.value) return;
  loadingMore.value = true;
  try {
    const resp = await getSessionReplay(sessionId.value, events.value.length, 10000);
    const moreEvents = resp.events ?? [];
    events.value.push(...moreEvents);
    total.value = resp.total ?? events.value.length;
    hasMore.value = resp.has_more ?? false;
    if (resp.has_more) setTimeout(() => loadMore(), 50);
  } catch (e) {
    console.warn('loadMore failed', e);
  } finally {
    loadingMore.value = false;
  }
}

// ===== Player =====
function initPlayer() {
  if (!playerContainer.value || events.value.length === 0) return;

  // 清理旧 player
  if (replayer) {
    stopPolling();
    stopResponsiveSizing();
    replayer.destroy();
    replayer = null;
  }
  playerContainer.value.replaceChildren();

  if (events.value.length < 2) {
    return; // 等 events 就绪
  }

  try {
    const config: Partial<playerConfig> = {
      speed: 1,
      root: playerContainer.value,
      skipInactive: true,
      showDebug: false,
      showController: false,
      triggerFocus: true,
      pauseAnimation: false,
    };

    replayer = new Replayer(events.value, config);

    // 元数据
    const meta = replayer.getMetaData();
    totalTimeMs.value = meta.totalTime;

    // 状态变化跟踪
    replayer.on('state-change', (e: unknown) => {
      const payload = (e as { payload?: string })?.payload;
      isPlaying.value = payload === 'playing';
    });

    // 时间轮询（Replayer 不推送连续时间事件）
    startPolling();

    // 响应式 sizing
    startResponsiveSizing();
  } catch (e) {
    console.error('Replayer init failed', e);
    error.value = t('replay.play_failed');
  }
}

function startPolling() {
  stopPolling();
  timePoll = setInterval(() => {
    if (!replayer || !isPlaying.value) return;
    const t = replayer.getCurrentTime();
    if (typeof t === 'number') currentTimeMs.value = t;
  }, 100);
}

function stopPolling() {
  if (timePoll) { clearInterval(timePoll); timePoll = null; }
}

// ===== 控制方法 =====
function togglePlay() {
  if (!replayer) return;
  if (isPlaying.value) {
    replayer.pause();
  } else {
    replayer.play(currentTimeMs.value);
  }
}

function onScrub(e: Event) {
  if (!replayer) return;
  const target = e.target as HTMLInputElement;
  const newTime = Number(target.value);
  currentTimeMs.value = newTime;
  replayer.goto(newTime);
}

function setSpeed(s: number) {
  if (!replayer) return;
  speed.value = s;
  replayer.setConfig({ speed: s } as Partial<playerConfig>);
}

function toggleSkipInactive() {
  if (!replayer) return;
  skipInactive.value = !skipInactive.value;
  replayer.setConfig({ skipInactive: skipInactive.value } as Partial<playerConfig>);
}

// ===== 生命周期 =====
watch(events, (newEvents) => {
  if (newEvents.length >= 2 && !replayer) initPlayer();
}, { deep: false });

onMounted(loadInitial);

onUnmounted(() => {
  stopPolling();
  if (replayer) {
    stopResponsiveSizing();
    replayer.destroy();
    replayer = null;
  }
});

watch(() => route.params.session_id, (newId) => {
  if (newId && newId !== sessionId.value) {
    sessionId.value = String(newId);
    stopPolling();
    if (replayer) { stopResponsiveSizing(); replayer.destroy(); replayer = null; }
    currentTimeMs.value = 0;
    totalTimeMs.value = 0;
    isPlaying.value = false;
    loadInitial();
  }
});
</script>

<template>
  <div class="replay-viewer">
    <header class="viewer-header">
      <RouterLink to="/replay" class="back-link">
        <PhArrowLeft :size="16" weight="regular" aria-hidden="true" />
        <span>{{ t('replay.back') }}</span>
      </RouterLink>
      <div class="session-meta">
        <span class="label">{{ t('replay.session_label') }}</span>
        <code class="pc-mono session-id">{{ sessionId }}</code>
        <span class="sep" aria-hidden="true">·</span>
        <span class="label">{{ t('replay.events_label') }}</span>
        <span class="value">{{ total }}</span>
        <PhSpinner
          v-if="loadingMore"
          :size="14"
          weight="regular"
          aria-hidden="true"
          class="loading-spin"
        />
      </div>
    </header>

    <div class="player-area">
      <div v-if="loading" class="state">
        <PhSpinner :size="24" weight="regular" aria-hidden="true" class="loading-spin" />
        <span>{{ t('replay.loading') }}</span>
      </div>
      <div v-else-if="error" class="state error-state" role="alert">
        <PhWarningCircle :size="20" weight="regular" aria-hidden="true" />
        <span>{{ error }}</span>
      </div>
      <div v-else-if="events.length === 0" class="state">
        <span>{{ t('replay.no_events') }}</span>
      </div>

      <div ref="playerContainer" class="player-container"></div>
    </div>

    <!-- Custom Calm Chrome 控制栏 -->
    <footer v-if="!loading && !error && events.length > 0" class="controls">
      <button
        type="button"
        class="ctrl-btn play-btn"
        :aria-label="isPlaying ? 'Pause' : 'Play'"
        @click="togglePlay"
      >
        <component :is="isPlaying ? PhPause : PhPlay" :size="18" weight="fill" aria-hidden="true" />
      </button>

      <span class="time current">{{ formatTime(currentTimeMs) }}</span>

      <input
        type="range"
        class="scrubber"
        min="0"
        :max="totalTimeMs || 1"
        step="100"
        :value="currentTimeMs"
        :style="{ '--progress': progressPercent + '%' }"
        :aria-label="'seek'"
        @input="onScrub"
      />

      <span class="time total">{{ formatTime(totalTimeMs) }}</span>

      <div class="speed-toggle" role="group">
        <button
          v-for="s in speedOptions"
          :key="s"
          type="button"
          class="seg"
          :class="{ active: speed === s }"
          :aria-pressed="speed === s"
          @click="setSpeed(s)"
        >
          {{ s }}×
        </button>
      </div>

      <button
        type="button"
        class="ctrl-btn"
        :class="{ active: skipInactive }"
        :aria-pressed="skipInactive"
        :aria-label="'toggle skip inactive'"
        @click="toggleSkipInactive"
      >
        <PhSkipForward :size="16" weight="regular" aria-hidden="true" />
      </button>
    </footer>
  </div>
</template>

<style>
/* replay-core Replayer 全局样式 */
.replayer-wrapper {
  transform-origin: top left;
  left: 50%;
  top: 50%;
  position: relative;
}
</style>

<style scoped>
.replay-viewer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--pc-color-bg-canvas);
}

.viewer-header {
  display: flex;
  align-items: center;
  gap: var(--pc-space-section);
  padding: var(--pc-space-component) var(--pc-space-section);
  background: var(--pc-color-bg-surface);
  border-bottom: 1px solid var(--pc-color-border-default);
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  text-decoration: none;
  color: var(--pc-color-accent-default);
  font-size: var(--pc-text-sm);
  font-weight: var(--pc-weight-medium);
  padding: 6px 10px;
  border-radius: var(--pc-radius-md);
  transition: background var(--pc-duration-fast) var(--pc-easing), color var(--pc-duration-fast) var(--pc-easing);
}
.back-link:hover { background: var(--pc-color-accent-subtle); color: var(--pc-color-accent-hover); }

.session-meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--pc-text-sm);
  color: var(--pc-color-text-secondary);
}

.session-meta .label { color: var(--pc-color-text-muted); font-size: var(--pc-text-xs); text-transform: uppercase; letter-spacing: 0.04em; }
.session-meta .value { color: var(--pc-color-text-primary); font-weight: var(--pc-weight-medium); font-variant-numeric: tabular-nums; }
.session-meta .sep { color: var(--pc-color-text-muted); opacity: 0.5; }

.session-id {
  background: var(--pc-color-bg-subtle);
  padding: 2px 6px;
  border-radius: var(--pc-radius-sm);
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-primary);
}

.loading-spin { color: var(--pc-color-accent-default); animation: spin 1.2s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .loading-spin { animation-duration: 3s; } }

.player-area {
  flex: 1;
  min-height: 0;
  position: relative;
  background: var(--pc-color-bg-subtle);
  display: flex;
  align-items: stretch;
  justify-content: stretch;
  overflow: hidden;
}

.player-container {
  flex: 1;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
}

.player-container :deep(iframe) {
  display: block !important;
  pointer-events: auto !important;
  border: 0;
}

.state {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--pc-space-component);
  color: var(--pc-color-text-muted);
  background: var(--pc-color-bg-subtle);
  font-size: var(--pc-text-sm);
}

.error-state { color: var(--pc-color-danger); background: var(--pc-color-danger-subtle); }

/* ===== Custom Calm Chrome 控制栏 ===== */
.controls {
  display: flex;
  align-items: center;
  gap: var(--pc-space-component);
  padding: var(--pc-space-component) var(--pc-space-section);
  background: var(--pc-color-bg-surface);
  border-top: 1px solid var(--pc-color-border-default);
}

.ctrl-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--pc-radius-md);
  color: var(--pc-color-text-secondary);
  background: transparent;
  transition: background var(--pc-duration-fast) var(--pc-easing), color var(--pc-duration-fast) var(--pc-easing);
}
.ctrl-btn:hover { background: var(--pc-color-bg-subtle); color: var(--pc-color-text-primary); }
.ctrl-btn.active { background: var(--pc-color-accent-subtle); color: var(--pc-color-accent-default); }

.play-btn {
  background: var(--pc-color-accent-default);
  color: var(--pc-color-accent-on);
  width: 36px;
  height: 36px;
  border-radius: var(--pc-radius-pill);
}
.play-btn:hover { background: var(--pc-color-accent-hover); color: var(--pc-color-accent-on); }

.time {
  font-family: var(--pc-font-mono);
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-secondary);
  font-variant-numeric: tabular-nums;
  min-width: 40px;
  text-align: center;
}

/* scrubber */
.scrubber {
  flex: 1;
  min-width: 100px;
  height: 4px;
  appearance: none;
  background: linear-gradient(
    to right,
    var(--pc-color-accent-default) 0%,
    var(--pc-color-accent-default) var(--progress, 0%),
    var(--pc-color-border-strong) var(--progress, 0%),
    var(--pc-color-border-strong) 100%
  );
  border-radius: var(--pc-radius-pill);
  cursor: pointer;
  outline: none;
}
.scrubber::-webkit-slider-thumb { appearance: none; width: 14px; height: 14px; border-radius: 50%; background: var(--pc-color-bg-surface); border: 2px solid var(--pc-color-accent-default); box-shadow: var(--pc-shadow-xs); transition: transform var(--pc-duration-fast) var(--pc-easing); }
.scrubber::-webkit-slider-thumb:hover { transform: scale(1.15); }
.scrubber::-moz-range-thumb { width: 14px; height: 14px; border-radius: 50%; background: var(--pc-color-bg-surface); border: 2px solid var(--pc-color-accent-default); box-shadow: var(--pc-shadow-xs); }
.scrubber:focus-visible { box-shadow: var(--pc-focus-ring); }

.speed-toggle {
  display: inline-flex;
  padding: 2px;
  background: var(--pc-color-bg-subtle);
  border-radius: var(--pc-radius-pill);
}
.speed-toggle .seg {
  padding: 3px 8px;
  font-size: var(--pc-text-xs);
  font-weight: var(--pc-weight-medium);
  color: var(--pc-color-text-muted);
  border-radius: var(--pc-radius-pill);
  transition: color var(--pc-duration-fast) var(--pc-easing), background var(--pc-duration-fast) var(--pc-easing);
  min-width: 28px;
}
.speed-toggle .seg:hover:not(.active) { color: var(--pc-color-text-secondary); }
.speed-toggle .seg.active {
  background: var(--pc-color-bg-surface);
  color: var(--pc-color-accent-default);
  box-shadow: var(--pc-shadow-xs);
}
</style>
