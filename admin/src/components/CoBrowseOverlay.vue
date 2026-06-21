<script setup lang="ts">
// CoBrowseOverlay：覆盖在 rrweb-player 上的透明 div
// 运营"进入控制模式"时启用，捕获鼠标事件并转换为命令下发
// 详见 docs/progress/2026-06-17-slice-1e-spec.md §Overlay 实现

import { ref, onUnmounted, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { sendCommand, type CommandType } from '../api/sessions';

const { t } = useI18n();

const props = defineProps<{
  /** 当前订阅的 session ID */
  sessionId: string | null;
  /** 是否激活（Start 后 true，Stop 后 false） */
  active: boolean;
  /** 运营员名字（光标显示） */
  operatorName?: string;
}>();

const emit = defineEmits<{
  (e: 'command-sent', type: CommandType, count: number): void;
  (e: 'error', message: string): void;
}>();

const lastClickAt = ref(0);
const lastFillAt = ref(0);
const commandCount = ref(0);

// rAF + 30fps 节流（cursor_highlight）
let lastCursorSendAt = 0;
let rafScheduled = false;
let pendingX = 0;
let pendingY = 0;

const CURSOR_MIN_INTERVAL_MS = 1000 / 30; // 30fps
const CLICK_DEBOUNCE_MS = 200;
const FILL_DEBOUNCE_MS = 500;

function onMouseMove(e: MouseEvent) {
  if (!props.active) return;
  pendingX = e.clientX;
  pendingY = e.clientY;
  if (rafScheduled) return;
  rafScheduled = true;
  requestAnimationFrame(flushCursor);
}

function flushCursor() {
  rafScheduled = false;
  if (!props.active || !props.sessionId) return;
  const now = Date.now();
  if (now - lastCursorSendAt < CURSOR_MIN_INTERVAL_MS) {
    // 太快，下一帧再试
    rafScheduled = true;
    requestAnimationFrame(flushCursor);
    return;
  }
  lastCursorSendAt = now;
  const x = pendingX;
  const y = pendingY;
  void sendCursorHighlight(x, y);
}

async function sendCursorHighlight(x: number, y: number) {
  if (!props.sessionId) return;
  try {
    await sendCommand(props.sessionId, 'cursor_highlight', {
      x,
      y,
      name: props.operatorName ?? t('chat.operator'),
    });
    commandCount.value++;
    emit('command-sent', 'cursor_highlight', commandCount.value);
  } catch (e) {
    emit('error', (e as Error).message);
  }
}

function onClick(e: MouseEvent) {
  if (!props.active || !props.sessionId) return;
  e.preventDefault();
  e.stopPropagation();
  const now = Date.now();
  if (now - lastClickAt.value < CLICK_DEBOUNCE_MS) return;
  lastClickAt.value = now;
  // 1f：通过 postMessage 向 rrweb-player iframe 请求 nodeID
  void requestNodeIdAt(e.clientX, e.clientY).then((nodeID) => {
    void sendClick(nodeID, e.clientX, e.clientY);
  });
}

/**
 * 1f：向 rrweb-player iframe 发 postMessage 请求坐标处的 nodeID。
 * iframe 内部收到后用 document.elementFromPoint + 读 data-rr-node-id 回复。
 *
 * 注：当前 rrweb-player v2 alpha 未原生支持该协议；
 * 1f MVP 用坐标 fallback（nodeID=0）+ 服务端按坐标点击。
 * 后续可通过 monkey-patch rrweb-player 添加 message listener 实现。
 */
async function requestNodeIdAt(_x: number, _y: number): Promise<number> {
  // 1f MVP：保持坐标 fallback
  return 0;
}

async function sendClick(nodeID: number, x: number, y: number) {
  if (!props.sessionId) return;
  try {
    await sendCommand(props.sessionId, 'click', { node_id: nodeID, x, y });
    commandCount.value++;
    emit('command-sent', 'click', commandCount.value);
  } catch (e) {
    emit('error', (e as Error).message);
  }
}

function onWheel(e: WheelEvent) {
  if (!props.active || !props.sessionId) return;
  e.preventDefault();
  void sendScroll(window.scrollX + e.deltaX, window.scrollY + e.deltaY);
}

async function sendScroll(x: number, y: number) {
  if (!props.sessionId) return;
  try {
    await sendCommand(props.sessionId, 'scroll', { x: Math.round(x), y: Math.round(y) });
    commandCount.value++;
    emit('command-sent', 'scroll', commandCount.value);
  } catch (e) {
    emit('error', (e as Error).message);
  }
}

// 监听 active：Stop 时清状态
watch(
  () => props.active,
  (active) => {
    if (!active) {
      rafScheduled = false;
      commandCount.value = 0;
    }
  },
);

onUnmounted(() => {
  rafScheduled = false;
});

// 暴露给父：填表单（运营从专用面板触发，非直接捕获）
async function fillInput(nodeID: number, value: string) {
  if (!props.sessionId) return;
  const now = Date.now();
  if (now - lastFillAt.value < FILL_DEBOUNCE_MS) return;
  lastFillAt.value = now;
  try {
    await sendCommand(props.sessionId, 'fill_input', { node_id: nodeID, value });
    commandCount.value++;
    emit('command-sent', 'fill_input', commandCount.value);
  } catch (e) {
    emit('error', (e as Error).message);
  }
}

async function releaseControl() {
  if (!props.sessionId) return;
  try {
    await sendCommand(props.sessionId, 'release_control', {});
    emit('command-sent', 'release_control', commandCount.value);
  } catch (e) {
    emit('error', (e as Error).message);
  }
}

defineExpose({ fillInput, releaseControl });
</script>

<template>
  <div
    v-if="active"
    class="co-browse-overlay"
    @mousemove="onMouseMove"
    @click="onClick"
    @wheel="onWheel"
  >
    <div class="badge">
      <span class="dot" aria-hidden="true" />
      {{ t('cobrowse.active_badge', { count: commandCount }) }}
    </div>
  </div>
</template>

<style scoped>
.co-browse-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  /* Calm Crafted:用 accent 半透明 + 柔和 inset 边框(代替旧 dashed #409eff) */
  background: color-mix(in srgb, var(--pc-color-accent-default) 6%, transparent);
  box-shadow: inset 0 0 0 2px
    color-mix(in srgb, var(--pc-color-accent-default) 50%, transparent);
  z-index: 10;
  cursor: crosshair;
}

.badge {
  position: absolute;
  top: var(--pc-space-component);
  left: var(--pc-space-component);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: var(--pc-color-accent-default);
  color: var(--pc-color-accent-on);
  font-size: var(--pc-text-xs);
  font-weight: var(--pc-weight-medium);
  font-family: var(--pc-font-sans);
  border-radius: var(--pc-radius-pill);
  box-shadow: var(--pc-shadow-sm);
  pointer-events: none;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: var(--pc-radius-pill);
  background: var(--pc-color-accent-on);
  animation: pulse 1.4s var(--pc-easing) infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

@media (prefers-reduced-motion: reduce) {
  .dot {
    animation: none;
  }
}
</style>
