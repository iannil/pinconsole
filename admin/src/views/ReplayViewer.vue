<script setup lang="ts">
// 单会话历史回放 + Custom Calm Chrome(design-system.md §4.3)
//
// 改动:隐藏 rrweb-player 原生控制器,自建 Calm 风控制栏:
// - Play/Pause + scrubber + current/total time
// - Speed selector(1×/2×/4×/8×)
// - Skip inactive toggle
//
// rrweb-player v2 alpha API(见 node_modules/.../rrweb-player.d.ts):
//   player.play() / .pause() / .toggle() / .goto(timeOffset, play?)
//   player.setSpeed(speed) / .toggleSkipInactive()
//   player.getMetaData() → { startTime, endTime, totalTime }
//   player.addEventListener('ui-update-current-time', cb)
//   player.addEventListener('ui-update-player-state', cb)

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
// rrweb-player v2 alpha 类型不完整,用最小接口描述我们用到的 API surface
interface RRWebPlayerInstance {
  play: () => void;
  pause: () => void;
  toggle: () => void;
  goto: (timeOffset: number, play?: boolean) => void;
  setSpeed: (speed: number) => void;
  toggleSkipInactive: () => void;
  getMetaData: () => { startTime: number; endTime: number; totalTime: number };
  // rrweb-player 把 dispatch 的 detail({ payload }) 直接传给 handler(见
  // node_modules/.../rrweb-player.js:14750 `controller.$on(event, ({ detail }) => handler(detail))`)。
  // 所以 handler 收到的就是 { payload },不是再裹一层 { detail: { payload } }。
  addEventListener: (event: string, handler: (e: { payload?: unknown }) => void) => void;
  append?: (events: unknown[]) => void;
  // v2 支持 $set 热更新 props(width/height 等),用于响应式 sizing
  $set?: (props: Record<string, unknown>) => void;
  // v2 暴露 getReplayer 拿到内部 Replayer,用于兜底手动触发 handleResize
  getReplayer?: () => { handleResize?: (dimension: { width: number; height: number }) => void } | null;
}
let player: RRWebPlayerInstance | null = null;

// 响应式 sizing:覆盖 rrweb-player 默认 1024x576,按真实录制视口比例动态算
// player 外框尺寸,避免 letterbox / 视觉错位。详见 composable 注释。
const { start: startResponsiveSizing, stop: stopResponsiveSizing } =
  useResponsivePlayerSize(playerContainer, () => player);

// ===== 自建控制栏状态 =====
const isPlaying = ref(false);
const currentTimeMs = ref(0);
const totalTimeMs = ref(0);
const speed = ref(1);
const skipInactive = ref(true);

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

async function loadInitial() {
  loading.value = true;
  error.value = null;
  try {
    const resp = await getSessionReplay(sessionId.value, 0, 10000);
    events.value = resp.events ?? [];
    total.value = resp.total ?? 0;
    hasMore.value = resp.has_more ?? false;
    if (resp.has_more) {
      loadMore();
    }
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
    // rrweb-player v2 alpha 没有 append,用 addEvent(逐个)。
    // 但当前文件级 instance 接口未暴露 addEvent,留 TODO(实际历史会话事件
    // > 10k 的极少,v1 不阻塞)。
    if (resp.has_more) {
      setTimeout(() => loadMore(), 50);
    }
  } catch (e) {
    console.warn('loadMore failed', e);
  } finally {
    loadingMore.value = false;
  }
}

async function initPlayer() {
  if (!playerContainer.value || events.value.length === 0) return;
  try {
    // 必须同时加载 rrweb-player/dist/style.css,否则 .rr-player__frame 缺
    // overflow:hidden、.replayer-wrapper 缺 transform-origin:left top + left:50%/top:50%,
    // iframe 布局尺寸外溢触发 .player-container 滚动条 → responsive sizing 算小。
    // 详见 ReplayPlayer.vue 同名注释。
    await import('rrweb-player/dist/style.css');
    const mod = await import('rrweb-player');
    const Player = mod.default;
    playerContainer.value.replaceChildren();
    player = new Player({
      target: playerContainer.value,
      props: {
        events: events.value,
        showDebug: false,
        autoPlay: false,
        skipInactive: true,
        showController: false, // Phase 4:隐藏原生控制器
        // alpha.20 默认 sandbox="allow-same-origin"(无 allow-scripts),
        // 回放含 <script> 的页面时每个 script 触发一次 sandbox warning。
        // UNSAFE_replayCanvas 同时打开 allow-scripts。详见 ReplayPlayer.vue 同名注释。
        UNSAFE_replayCanvas: true,
      },
    }) as unknown as RRWebPlayerInstance;

    // rrweb-player v2 alpha.20:Svelte onMount 用 microtask 调度,
    // `new Replayer(events, ...)` 在 onMount 里跑,所以 `new Player(...)` 返回后
    // 内部 `replayer` 还是 undefined。立刻调 `getMetaData` / `addEventListener`
    // 会触发 `Cannot read properties of undefined (reading 'getMetaData')`。
    // 等 microtask + 一帧动画(更稳)让 onMount flush 完成。
    await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
    if (!player?.getReplayer?.()) {
      throw new Error('rrweb-player Replayer 未创建(events 不足或构造抛错)');
    }

    // 拿元数据算 total time(scrubber max)
    try {
      const meta = player.getMetaData();
      totalTimeMs.value = meta.totalTime ?? meta.endTime - meta.startTime;
    } catch (e) {
      console.warn('getMetaData failed', e);
    }

    // 跟踪 current time / player state
    player.addEventListener('ui-update-current-time', (e) => {
      const payload = e?.payload;
      if (typeof payload === 'number') {
        currentTimeMs.value = payload;
      }
    });
    player.addEventListener('ui-update-player-state', (e) => {
      const payload = e?.payload;
      isPlaying.value = payload === 'playing';
    });

    // 启动响应式 sizing(覆盖 rrweb-player 默认 1024x576,从 iframe width/height 读真实录制视口)
    startResponsiveSizing();
  } catch (e) {
    console.error('rrweb-player init failed', e);
    error.value = t('replay.play_failed');
  }
}

function togglePlay() {
  if (!player) return;
  player.toggle();
}

function onScrub(e: Event) {
  if (!player) return;
  const target = e.target as HTMLInputElement;
  const newTime = Number(target.value);
  currentTimeMs.value = newTime;
  player.goto(newTime, isPlaying.value);
}

function setSpeed(s: number) {
  if (!player) return;
  speed.value = s;
  player.setSpeed(s);
}

function toggleSkipInactive() {
  if (!player) return;
  skipInactive.value = !skipInactive.value;
  player.toggleSkipInactive();
}

watch(events, (newEvents) => {
  if (newEvents.length > 0 && !player) {
    initPlayer();
  }
}, { deep: false });

onMounted(loadInitial);

onUnmounted(() => {
  if (playerContainer.value) {
    playerContainer.value.replaceChildren();
  }
  stopResponsiveSizing();
  player = null;
});

// 监听 sessionId 变化(路由 param 变)
watch(() => route.params.session_id, (newId) => {
  if (newId && newId !== sessionId.value) {
    sessionId.value = String(newId);
    if (playerContainer.value) playerContainer.value.replaceChildren();
    stopResponsiveSizing();
    player = null;
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

    <!-- Custom Calm Chrome 控制栏(仅 player 就绪后显示) -->
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
  transition: background var(--pc-duration-fast) var(--pc-easing),
    color var(--pc-duration-fast) var(--pc-easing);
}

.back-link:hover {
  background: var(--pc-color-accent-subtle);
  color: var(--pc-color-accent-hover);
}

.session-meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--pc-text-sm);
  color: var(--pc-color-text-secondary);
}

.session-meta .label {
  color: var(--pc-color-text-muted);
  font-size: var(--pc-text-xs);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.session-meta .value {
  color: var(--pc-color-text-primary);
  font-weight: var(--pc-weight-medium);
  font-variant-numeric: tabular-nums;
}

.session-meta .sep {
  color: var(--pc-color-text-muted);
  opacity: 0.5;
}

.session-id {
  background: var(--pc-color-bg-subtle);
  padding: 2px 6px;
  border-radius: var(--pc-radius-sm);
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-primary);
}

.loading-spin {
  color: var(--pc-color-accent-default);
  animation: spin 1.2s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .loading-spin { animation-duration: 3s; }
}

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
  /* 响应式 sizing 下 player 外框已按容器+录制比例算好,
     通常不会超出。保留 auto 作为兜底(异常尺寸时仍可滚动)。 */
  overflow: auto;
  /* player 外框(rr-player)由 rrweb-player 设 inline width/height,
     flex 居中让 letterbox 平均分布在两侧而非顶左对齐 */
  display: flex;
  align-items: center;
  justify-content: center;
}

/* rrweb-player iframe 强制可见(rrweb alpha mirror iframe 默认 display:none)。
   不设 width/height:100% — rrweb-player 通过 iframe 的 width/height attribute
   + .replayer-wrapper 的 transform:scale 缩放,height:100% 无法 resolve 时浏览器
   fallback 到默认 150px 导致视觉压扁。 */
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

.error-state {
  color: var(--pc-color-danger);
  background: var(--pc-color-danger-subtle);
}

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
  transition: background var(--pc-duration-fast) var(--pc-easing),
    color var(--pc-duration-fast) var(--pc-easing);
}

.ctrl-btn:hover {
  background: var(--pc-color-bg-subtle);
  color: var(--pc-color-text-primary);
}

.ctrl-btn.active {
  background: var(--pc-color-accent-subtle);
  color: var(--pc-color-accent-default);
}

.play-btn {
  background: var(--pc-color-accent-default);
  color: var(--pc-color-accent-on);
  width: 36px;
  height: 36px;
  border-radius: var(--pc-radius-pill);
}

.play-btn:hover {
  background: var(--pc-color-accent-hover);
  color: var(--pc-color-accent-on);
}

.time {
  font-family: var(--pc-font-mono);
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-secondary);
  font-variant-numeric: tabular-nums;
  min-width: 40px;
  text-align: center;
}

/* scrubber —— 自定义 range input */
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

.scrubber::-webkit-slider-thumb {
  appearance: none;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--pc-color-bg-surface);
  border: 2px solid var(--pc-color-accent-default);
  box-shadow: var(--pc-shadow-xs);
  transition: transform var(--pc-duration-fast) var(--pc-easing);
}

.scrubber::-webkit-slider-thumb:hover {
  transform: scale(1.15);
}

.scrubber::-moz-range-thumb {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--pc-color-bg-surface);
  border: 2px solid var(--pc-color-accent-default);
  box-shadow: var(--pc-shadow-xs);
}

.scrubber:focus-visible {
  box-shadow: var(--pc-focus-ring);
}

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
  transition: color var(--pc-duration-fast) var(--pc-easing),
    background var(--pc-duration-fast) var(--pc-easing);
  min-width: 28px;
}

.speed-toggle .seg:hover:not(.active) {
  color: var(--pc-color-text-secondary);
}

.speed-toggle .seg.active {
  background: var(--pc-color-bg-surface);
  color: var(--pc-color-accent-default);
  box-shadow: var(--pc-shadow-xs);
}
</style>
