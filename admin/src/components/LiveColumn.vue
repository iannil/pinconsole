<script setup lang="ts">
// LiveColumn —— Dashboard 中间列(Header + Replay + Controls)
// 来源:docs/design-system.md §4.2 (3-Column Smart Expand)
//
// 三种状态:
// 1. 未选 visitor:placeholder
// 2. 选了但未 claim:Header + Replay + [Claim] 按钮(底部)
// 3. 已 claim:Header + Replay + CoBrowseOverlay + [Release] 按钮 + claim lost 提示
//
// events 直接从 visitors store 读(LiveColumn 是 Dashboard 专属,不需 prop drilling)
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { PhHand, PhX, PhWarningCircle } from '@phosphor-icons/vue';
import { useVisitorsStore } from '../stores/visitors';
import ReplayPlayer from './ReplayPlayer.vue';
import CoBrowseOverlay from './CoBrowseOverlay.vue';

const props = defineProps<{
  /** claim 状态 */
  coBrowsingActive: boolean;
  /** claim 失败/丢失的提示文案(空则不显示) */
  claimError: string;
  /** cobrowse hint 当前状态文本(由 Dashboard 算好传入) */
  cobrowseHint: string;
  /** 运营名(CoBrowseOverlay 显示在光标处) */
  operatorName: string;
}>();

const emit = defineEmits<{
  (e: 'toggle-cobrowse'): void;
}>();

const { t } = useI18n();
const store = useVisitorsStore();

const visitor = computed(() => store.selectedVisitor);
const sessionId = computed(() => store.selectedSessionId);
const events = computed(() => store.selectedEvents);

const claimButtonLabel = computed(() =>
  props.coBrowsingActive ? t('dashboard.stop_cobrowse') : t('dashboard.start_cobrowse'),
);
</script>

<template>
  <section class="live-column" :class="{ engaging: coBrowsingActive }">
    <!-- Header:访客 fingerprint + session ID -->
    <header v-if="visitor" class="header">
      <div class="identity">
        <h2 class="fingerprint">{{ visitor.fingerprint.slice(0, 16) }}</h2>
        <span class="sid pc-mono">{{ sessionId }}</span>
      </div>
    </header>

    <!-- Replay 区:flex 1,含 rrweb-player + 透明 CoBrowseOverlay -->
    <div class="replay-area">
      <div v-if="!visitor" class="placeholder">
        <p>{{ t('dashboard.select_visitor') }}</p>
      </div>
      <template v-else>
        <ReplayPlayer
          :key="sessionId ?? ''"
          :events="events"
          :session-id="sessionId"
        />
        <CoBrowseOverlay
          :session-id="sessionId"
          :active="coBrowsingActive"
          :operator-name="operatorName"
        />
      </template>
    </div>

    <!-- Controls:Claim/Release + 提示 + claim 错误 -->
    <footer v-if="visitor" class="controls">
      <button
        type="button"
        class="pc-btn"
        :class="coBrowsingActive ? 'pc-btn--danger' : 'pc-btn--primary'"
        @click="emit('toggle-cobrowse')"
      >
        <component
          :is="coBrowsingActive ? PhX : PhHand"
          :size="16"
          weight="regular"
          aria-hidden="true"
        />
        <span>{{ claimButtonLabel }}</span>
      </button>

      <span class="hint">
        {{ cobrowseHint }}
      </span>

      <p v-if="claimError" class="claim-error" role="alert">
        <PhWarningCircle :size="14" weight="regular" aria-hidden="true" />
        <span>{{ claimError }}</span>
      </p>
    </footer>
  </section>
</template>

<style scoped>
.live-column {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--pc-color-bg-surface);
  min-width: 0;
  flex: 1;
}

/* claim 状态下左/右边界微调以留位置给 engagement panel */
.live-column.engaging {
  border-right: 1px solid var(--pc-color-border-default);
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--pc-space-component) var(--pc-space-section);
  border-bottom: 1px solid var(--pc-color-border-default);
  background: var(--pc-color-bg-surface);
}

.identity {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.fingerprint {
  margin: 0;
  font-family: var(--pc-font-mono);
  font-size: var(--pc-text-md);
  font-weight: var(--pc-weight-semibold);
  color: var(--pc-color-text-primary);
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sid {
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
}

.replay-area {
  flex: 1;
  position: relative;
  min-height: 0;
  background: var(--pc-color-bg-canvas);
  display: flex;
  align-items: stretch;
  justify-content: stretch;
}

.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  color: var(--pc-color-text-muted);
  font-size: var(--pc-text-sm);
}

.controls {
  display: flex;
  align-items: center;
  gap: var(--pc-space-component);
  padding: var(--pc-space-component) var(--pc-space-section);
  border-top: 1px solid var(--pc-color-border-default);
  background: var(--pc-color-bg-subtle);
  flex-wrap: wrap;
}

.hint {
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
}

.claim-error {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 4px 10px;
  font-size: var(--pc-text-xs);
  color: var(--pc-color-danger);
  background: var(--pc-color-danger-subtle);
  border-radius: var(--pc-radius-md);
}
</style>
