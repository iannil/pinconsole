<script setup lang="ts">
// Phase 3:🚩 emoji → Phosphor PhFlag(fill,danger 色)。
// .flag-icon 类保留(tests + 旧 selector 仍工作)。
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { PhFlag } from '@phosphor-icons/vue';
import { useVisitorsStore } from '../stores/visitors';
import { formatRelative } from '../utils/time';
import type { WsStatus } from '../composables/useWs';
import StatusBadge from './StatusBadge.vue';

const props = defineProps<{
  /** WS 连接状态(来自 Dashboard useWs)。不传则不显示状态点。 */
  status?: WsStatus;
}>();

const { t } = useI18n();
const store = useVisitorsStore();

const list = computed(() => store.visitorList);

// WS status → StatusBadge variant + pulse
const statusVariant = computed<'success' | 'warning' | 'danger' | 'neutral'>(() => {
  switch (props.status) {
    case 'connected':
      return 'success';
    case 'connecting':
    case 'reconnecting':
      return 'warning';
    case 'closed':
      return 'danger';
    default:
      return 'neutral';
  }
});

const statusPulse = computed(() => props.status === 'connected');

const statusLabel = computed(() => {
  if (!props.status) return '';
  return t(`status.${props.status}`);
});

function onClick(sessionId: string) {
  store.select(sessionId);
}

// 1w P1-29:flagged session tooltip 文案(运营 hover 🚩 看到原因)
function flagTitle(reason?: string): string {
  if (reason) {
    return t('visitor.flagged_tooltip_with_reason', { reason });
  }
  return t('visitor.flagged_tooltip');
}
</script>

<template>
  <div class="visitor-list">
    <div class="header">
      <span class="count">{{ t('dashboard.online_count', { count: list.length }) }}</span>
      <StatusBadge
        v-if="status && statusLabel"
        :variant="statusVariant"
        :dot="true"
        :pulse="statusPulse"
      >
        {{ statusLabel }}
      </StatusBadge>
    </div>
    <ul>
      <li
        v-for="v in list"
        :key="v.sessionId"
        :class="{ selected: store.selectedSessionId === v.sessionId, flagged: v.isFlagged }"
        @click="onClick(v.sessionId)"
      >
        <div class="fingerprint" :title="v.fingerprint">
          <span class="flag-icon" v-if="v.isFlagged" :title="flagTitle(v.flagReason)">
            <PhFlag :size="12" weight="fill" aria-hidden="true" />
          </span>
          {{ v.fingerprint.slice(0, 12) }}
        </div>
        <div class="meta">
          <span class="events">{{ v.eventCount }} events</span>
          <span class="time">{{ formatRelative(v.lastEventAt ?? v.startedAt, t) }}</span>
        </div>
      </li>
      <li v-if="list.length === 0" class="empty">{{ t('dashboard.waiting') }}</li>
    </ul>
  </div>
</template>

<style scoped>
.visitor-list {
  width: 280px;
  border-right: 1px solid var(--pc-color-border-default);
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--pc-color-bg-surface);
  font-family: var(--pc-font-sans);
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--pc-space-field);
  padding: var(--pc-space-component) var(--pc-space-card);
  background: var(--pc-color-bg-subtle);
  border-bottom: 1px solid var(--pc-color-border-default);
  font-size: var(--pc-text-sm);
  color: var(--pc-color-text-secondary);
}

.count {
  font-weight: var(--pc-weight-medium);
}

ul {
  list-style: none;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  flex: 1;
}

li {
  padding: var(--pc-space-component) var(--pc-space-card);
  cursor: pointer;
  border-bottom: 1px solid var(--pc-color-border-default);
  transition: background var(--pc-duration-fast) var(--pc-easing);
}

li:hover {
  background: var(--pc-color-bg-subtle);
}

li.selected {
  background: var(--pc-color-accent-subtle);
  border-left: 3px solid var(--pc-color-accent-default);
  padding-left: calc(var(--pc-space-card) - 3px);
}

/* 1w P1-29:flagged session 高亮提示运营警惕 */
li.flagged {
  background: var(--pc-color-danger-subtle);
  border-left: 3px solid var(--pc-color-danger);
}

li.flagged:hover {
  background: var(--pc-color-danger-subtle);
  filter: brightness(0.97);
}

li.flagged.selected {
  background: var(--pc-color-danger-subtle);
  border-left: 3px solid var(--pc-color-danger);
}

.flag-icon {
  display: inline-flex;
  align-items: center;
  margin-right: 4px;
  color: var(--pc-color-danger);
  cursor: help;
}

.fingerprint {
  font-family: var(--pc-font-mono);
  font-size: var(--pc-text-sm);
  color: var(--pc-color-text-primary);
  margin-bottom: 4px;
  letter-spacing: -0.01em;
}

.meta {
  display: flex;
  justify-content: space-between;
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
}

.empty {
  padding: var(--pc-space-section) var(--pc-space-card);
  text-align: center;
  color: var(--pc-color-text-muted);
  cursor: default;
}
</style>
