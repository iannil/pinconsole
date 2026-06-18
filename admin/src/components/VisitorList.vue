<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useVisitorsStore } from '../stores/visitors';
import { formatRelative } from '../utils/time';

const { t } = useI18n();
const store = useVisitorsStore();

const list = computed(() => store.visitorList);

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
      <span>{{ t('dashboard.online_count', { count: list.length }) }}</span>
    </div>
    <ul>
      <li
        v-for="v in list"
        :key="v.sessionId"
        :class="{ selected: store.selectedSessionId === v.sessionId, flagged: v.isFlagged }"
        @click="onClick(v.sessionId)"
      >
        <div class="fingerprint" :title="v.fingerprint">
          <span class="flag-icon" v-if="v.isFlagged" :title="flagTitle(v.flagReason)">🚩</span>
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
  border-right: 1px solid #ebeef5;
  display: flex;
  flex-direction: column;
  height: 100vh;
  font-family: system-ui, sans-serif;
}
.header {
  padding: 1rem;
  background: #f5f7fa;
  font-size: 0.9rem;
  color: #606266;
  border-bottom: 1px solid #ebeef5;
}
ul {
  list-style: none;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  flex: 1;
}
li {
  padding: 0.75rem 1rem;
  cursor: pointer;
  border-bottom: 1px solid #f5f7fa;
  transition: background 0.1s;
}
li:hover {
  background: #f5f7fa;
}
li.selected {
  background: #ecf5ff;
  border-left: 3px solid #409eff;
  padding-left: calc(1rem - 3px);
}
/* 1w P1-29:flagged session 高亮(淡红底)提示运营警惕 */
li.flagged {
  background: #fef0f0;
  border-left: 3px solid #f56c6c;
}
li.flagged:hover {
  background: #fde2e2;
}
li.flagged.selected {
  background: #fde2e2;
  border-left: 3px solid #f56c6c;
}
.flag-icon {
  margin-right: 4px;
  cursor: help;
}
.fingerprint {
  font-family: ui-monospace, monospace;
  font-size: 0.85rem;
  color: #303133;
  margin-bottom: 0.25rem;
}
.meta {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: #909399;
}
.empty {
  padding: 2rem 1rem;
  text-align: center;
  color: #909399;
  cursor: default;
}
</style>
