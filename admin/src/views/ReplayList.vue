<script setup lang="ts">
// 历史会话列表页（切片 1d）
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { listEndedSessions, type EndedSession, type SinceRange } from '../api/sessions';
import { formatRelative } from '../utils/time';

const { t } = useI18n();
const router = useRouter();
const sessions = ref<EndedSession[]>([]);
const since = ref<SinceRange>('24h');
const loading = ref(false);
const error = ref<string | null>(null);

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

onMounted(refresh);
</script>

<template>
  <div class="replay-list">
    <header class="header">
      <h1>{{ t('replay.title') }}</h1>
      <div class="actions">
        <button
          v-for="r in (['24h', '7d', '30d'] as SinceRange[])"
          :key="r"
          :class="{ active: since === r }"
          @click="changeSince(r)"
        >
          {{ r }}
        </button>
        <button @click="refresh" :disabled="loading">{{ t('replay.refresh') }}</button>
        <RouterLink to="/dashboard" class="nav-link">{{ t('nav.live') }}</RouterLink>
      </div>
    </header>

    <div v-if="error" class="error">{{ error }}</div>

    <table v-else class="sessions-table">
      <thead>
        <tr>
          <th>访客</th>
          <th>开始</th>
          <th>时长</th>
          <th>事件数</th>
          <th>UA</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="s in sessions" :key="s.session_id" @click="openReplay(s.session_id)">
          <td class="fp">{{ s.fingerprint.slice(0, 12) }}</td>
          <td class="time">{{ formatRelative(s.started_at) }}</td>
          <td class="dur">{{ formatDuration(s.duration_ms) }}</td>
          <td class="count">{{ s.event_count }}</td>
          <td class="ua">{{ s.ua.slice(0, 40) }}</td>
          <td class="action">▶</td>
        </tr>
        <tr v-if="!loading && sessions.length === 0">
          <td colspan="6" class="empty">{{ t('replay.empty') }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts">
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3_600_000) return `${Math.round(ms / 60_000)}min`;
  return `${(ms / 3_600_000).toFixed(1)}h`;
}
</script>

<style scoped>
.replay-list {
  font-family: system-ui, sans-serif;
  padding: 1rem 1.5rem;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
.header h1 {
  font-size: 1.2rem;
  margin: 0;
}
.actions {
  display: flex;
  gap: 0.5rem;
}
button {
  padding: 0.3rem 0.7rem;
  border: 1px solid #dcdfe6;
  background: #fff;
  cursor: pointer;
  border-radius: 3px;
  font-size: 0.8rem;
}
button.active {
  background: #409eff;
  color: #fff;
  border-color: #409eff;
}
.nav-link {
  padding: 0.3rem 0.7rem;
  text-decoration: none;
  color: #606266;
  font-size: 0.8rem;
}
.error {
  padding: 1rem;
  background: #fef0f0;
  color: #f56c6c;
  border-radius: 4px;
  margin-bottom: 1rem;
}
.sessions-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}
.sessions-table th {
  text-align: left;
  padding: 0.6rem;
  background: #f5f7fa;
  color: #606266;
  font-weight: 500;
}
.sessions-table td {
  padding: 0.6rem;
  border-bottom: 1px solid #ebeef5;
}
.sessions-table tbody tr {
  cursor: pointer;
  transition: background 0.1s;
}
.sessions-table tbody tr:hover {
  background: #f5f7fa;
}
.fp {
  font-family: ui-monospace, monospace;
}
.time,
.dur {
  color: #606266;
}
.ua {
  color: #909399;
  font-size: 0.75rem;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.action {
  color: #409eff;
  font-size: 1rem;
}
.empty {
  text-align: center;
  color: #909399;
  padding: 2rem;
}
</style>
