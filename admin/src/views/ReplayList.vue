<script setup lang="ts">
// 历史会话列表(Phase 4 Calm 升级)
// - 表格用 var(--pc-*) token + Phosphor 图标
// - since filter 用 pill toggle(代替旧 button.active)
// - 删 "← Live" nav-link(AppShell 顶栏已有 nav)
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { PhArrowsClockwise, PhPlayCircle, PhWarningCircle, PhFolderOpen } from '@phosphor-icons/vue';
import { listEndedSessions, type EndedSession, type SinceRange } from '../api/sessions';
import { formatRelative } from '../utils/time';

const { t } = useI18n();
const router = useRouter();
const sessions = ref<EndedSession[]>([]);
const since = ref<SinceRange>('24h');
const loading = ref(false);
const error = ref<string | null>(null);

const sinceOptions: SinceRange[] = ['24h', '7d', '30d'];

async function refresh() {
  loading.value = true;
  error.value = null;
  try {
    const resp = await listEndedSessions(since.value);
    sessions.value = resp.sessions;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

function openReplay(sessionId: string) {
  router.push(`/replay/${sessionId}`);
}

function changeSince(s: SinceRange) {
  since.value = s;
  refresh();
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3_600_000) return `${Math.round(ms / 60_000)}min`;
  return `${(ms / 3_600_000).toFixed(1)}h`;
}

onMounted(refresh);
</script>

<template>
  <div class="replay-list">
    <header class="header">
      <div class="title-block">
        <h1>{{ t('replay.title') }}</h1>
        <span class="count">{{ sessions.length }}</span>
      </div>
      <div class="actions">
        <div class="since-toggle" role="group">
          <button
            v-for="r in sinceOptions"
            :key="r"
            type="button"
            class="seg"
            :class="{ active: since === r }"
            :aria-pressed="since === r"
            @click="changeSince(r)"
          >
            {{ r }}
          </button>
        </div>
        <button
          type="button"
          class="pc-btn pc-btn--secondary refresh-btn"
          :disabled="loading"
          @click="refresh"
        >
          <PhArrowsClockwise :size="16" weight="regular" aria-hidden="true" />
          <span>{{ t('replay.refresh') }}</span>
        </button>
      </div>
    </header>

    <div v-if="error" class="state error-state" role="alert">
      <PhWarningCircle :size="20" weight="regular" aria-hidden="true" />
      <span>{{ error }}</span>
    </div>

    <div v-else-if="!loading && sessions.length === 0" class="state empty-state">
      <PhFolderOpen :size="32" weight="regular" aria-hidden="true" />
      <p>{{ t('replay.empty') }}</p>
    </div>

    <table v-else class="sessions-table">
      <thead>
        <tr>
          <th>{{ t('replay.th_visitor') }}</th>
          <th>{{ t('replay.th_started') }}</th>
          <th>{{ t('replay.th_duration') }}</th>
          <th>{{ t('replay.th_events') }}</th>
          <th>{{ t('replay.th_ua') }}</th>
          <th aria-label="open"></th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="s in sessions"
          :key="s.session_id"
          tabindex="0"
          @click="openReplay(s.session_id)"
          @keydown.enter="openReplay(s.session_id)"
        >
          <td class="fp pc-mono">{{ s.fingerprint.slice(0, 12) }}</td>
          <td class="time">{{ formatRelative(s.started_at, t) }}</td>
          <td class="dur">{{ formatDuration(s.duration_ms) }}</td>
          <td class="count">{{ s.event_count }}</td>
          <td class="ua" :title="s.ua">{{ s.ua.slice(0, 40) }}</td>
          <td class="action" aria-hidden="true">
            <PhPlayCircle :size="18" weight="regular" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.replay-list {
  padding: var(--pc-space-section) var(--pc-space-page);
  height: 100%;
  overflow-y: auto;
  background: var(--pc-color-bg-canvas);
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--pc-space-section);
  margin-bottom: var(--pc-space-section);
}

.title-block {
  display: inline-flex;
  align-items: baseline;
  gap: var(--pc-space-component);
}

.title-block h1 {
  margin: 0;
  font-size: var(--pc-text-xl);
  font-weight: var(--pc-weight-semibold);
}

.count {
  font-size: var(--pc-text-sm);
  color: var(--pc-color-text-muted);
  font-variant-numeric: tabular-nums;
}

.actions {
  display: inline-flex;
  align-items: center;
  gap: var(--pc-space-component);
}

.since-toggle {
  display: inline-flex;
  padding: 2px;
  background: var(--pc-color-bg-subtle);
  border-radius: var(--pc-radius-pill);
}

.since-toggle .seg {
  padding: 4px 12px;
  font-size: var(--pc-text-xs);
  font-weight: var(--pc-weight-medium);
  color: var(--pc-color-text-muted);
  border-radius: var(--pc-radius-pill);
  transition: color var(--pc-duration-fast) var(--pc-easing),
    background var(--pc-duration-fast) var(--pc-easing);
}

.since-toggle .seg:hover:not(.active) {
  color: var(--pc-color-text-secondary);
}

.since-toggle .seg.active {
  background: var(--pc-color-bg-surface);
  color: var(--pc-color-text-primary);
  box-shadow: var(--pc-shadow-xs);
}

.refresh-btn {
  min-height: 32px;
  padding: 4px 12px;
  font-size: var(--pc-text-sm);
}

/* state placeholders(error / empty) */
.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--pc-space-component);
  padding: var(--pc-space-page);
  margin-top: var(--pc-space-section);
  border-radius: var(--pc-radius-lg);
  text-align: center;
}

.error-state {
  color: var(--pc-color-danger);
  background: var(--pc-color-danger-subtle);
}

.empty-state {
  color: var(--pc-color-text-muted);
  background: var(--pc-color-bg-surface);
  border: 1px dashed var(--pc-color-border-default);
}

.empty-state p {
  margin: 0;
  font-size: var(--pc-text-sm);
}

/* sessions table */
.sessions-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--pc-text-sm);
  background: var(--pc-color-bg-surface);
  border: 1px solid var(--pc-color-border-default);
  border-radius: var(--pc-radius-lg);
  overflow: hidden;
}

.sessions-table th {
  text-align: left;
  padding: var(--pc-space-component) var(--pc-space-card);
  background: var(--pc-color-bg-subtle);
  color: var(--pc-color-text-secondary);
  font-weight: var(--pc-weight-medium);
  font-size: var(--pc-text-xs);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid var(--pc-color-border-default);
}

.sessions-table td {
  padding: var(--pc-space-component) var(--pc-space-card);
  border-bottom: 1px solid var(--pc-color-border-default);
  color: var(--pc-color-text-primary);
}

.sessions-table tbody tr {
  cursor: pointer;
  transition: background var(--pc-duration-fast) var(--pc-easing);
}

.sessions-table tbody tr:hover {
  background: var(--pc-color-bg-subtle);
}

.sessions-table tbody tr:last-child td {
  border-bottom: none;
}

.sessions-table tbody tr:focus-visible {
  outline: none;
  box-shadow: var(--pc-focus-ring);
  position: relative;
}

.fp {
  font-weight: var(--pc-weight-medium);
  color: var(--pc-color-text-primary);
}

.time,
.dur {
  color: var(--pc-color-text-secondary);
  font-variant-numeric: tabular-nums;
}

.count {
  color: var(--pc-color-text-secondary);
  font-variant-numeric: tabular-nums;
}

.ua {
  color: var(--pc-color-text-muted);
  font-size: var(--pc-text-xs);
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.action {
  color: var(--pc-color-accent-default);
  text-align: right;
  width: 32px;
}
</style>
