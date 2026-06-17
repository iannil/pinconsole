<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useVisitorsStore } from '../stores/visitors';
import ReplayPlayer from './ReplayPlayer.vue';
import ChatPanel from './ChatPanel.vue';
import { sendCommand } from '../api/sessions';

const { t } = useI18n();

const store = useVisitorsStore();

const selectedEvents = computed(() => store.selectedEvents);

const recentFormSubmits = computed(() =>
  selectedEvents.value.filter((e) => e.type === 'form_submit' && e.form_submit).slice(-3).reverse(),
);

const incrementalCount = computed(
  () => selectedEvents.value.filter((e) => e.type === 'rrweb' && e.rrweb?.type === 3).length,
);

// 1g：弹窗发送
const popupTitle = ref('');
const popupBody = ref('');
const popupActionLabel = ref('');
const popupActionUrl = ref('');

async function sendPopup() {
  if (!store.selectedSessionId || !popupTitle.value.trim()) return;
  try {
    await sendCommand(store.selectedSessionId, 'show_popup', {
      title: popupTitle.value,
      body: popupBody.value,
      action_label: popupActionLabel.value || undefined,
      action_url: popupActionUrl.value || undefined,
      dismissible: true,
    });
    popupTitle.value = '';
    popupBody.value = '';
    popupActionLabel.value = '';
    popupActionUrl.value = '';
  } catch (e) {
    console.warn('popup send failed', e);
  }
}
</script>

<template>
  <div class="visitor-panel">
    <div v-if="!store.selectedVisitor" class="placeholder">
      <p>{{ t('dashboard.select_visitor') }}</p>
    </div>
    <template v-else>
      <div class="header">
        <h2>{{ store.selectedVisitor.fingerprint.slice(0, 16) }}</h2>
        <span class="sid">{{ store.selectedSessionId }}</span>
      </div>

      <!-- 1c：用 ReplayPlayer 替换 1b 的 SVG 鼠标圈 -->
      <div class="replay-area">
        <ReplayPlayer
          :key="store.selectedSessionId ?? ''"
          :events="selectedEvents"
          :session-id="store.selectedSessionId"
        />
      </div>

      <div class="events-area">
        <section>
          <h3>{{ t('visitor.events_total', { count: selectedEvents.length }) }}</h3>
          <p class="hint">{{ t('visitor.incremental_count', { count: incrementalCount }) }}</p>
        </section>
        <section>
          <h3>{{ t('visitor.recent_form_submits', { count: recentFormSubmits.length }) }}</h3>
          <ul>
            <li v-for="(f, idx) in recentFormSubmits" :key="idx">
              <code>{{ f.form_submit!.form_id || '(no id)' }}</code>
              <span>{{ t('visitor.field_count', { count: Object.keys(f.form_submit!.fields).length }) }}</span>
            </li>
            <li v-if="recentFormSubmits.length === 0" class="empty">{{ t('visitor.no_form_submits') }}</li>
          </ul>
        </section>
      </div>

      <!-- 1g：弹窗推送 -->
      <div class="popup-sender">
        <h3>{{ t('visitor.popup_title') }}</h3>
        <input v-model="popupTitle" :placeholder="t('visitor.popup_title_ph')" class="popup-input" />
        <input v-model="popupBody" :placeholder="t('visitor.popup_body_ph')" class="popup-input" />
        <input v-model="popupActionLabel" :placeholder="t('visitor.popup_action_ph')" class="popup-input" />
        <input v-model="popupActionUrl" :placeholder="t('visitor.popup_url_ph')" class="popup-input" />
        <button @click="sendPopup" :disabled="!popupTitle.trim()">{{ t('visitor.send_popup') }}</button>
      </div>

      <!-- 1g：聊天面板 -->
      <ChatPanel :session-id="store.selectedSessionId" />
    </template>
  </div>
</template>

<style scoped>
.visitor-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100vh;
  font-family: system-ui, sans-serif;
  background: #fff;
}
.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #909399;
}
.header {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #ebeef5;
}
.header h2 {
  margin: 0 0 0.25rem 0;
  font-size: 1rem;
  font-family: ui-monospace, monospace;
}
.sid {
  font-size: 0.75rem;
  color: #909399;
}
.replay-area {
  flex: 1;
  border-bottom: 1px solid #ebeef5;
  min-height: 300px;
}
.events-area {
  padding: 1rem 1.5rem;
  max-height: 30%;
  overflow-y: auto;
}
.popup-sender {
  padding: 0.5rem 1.5rem;
  border-top: 1px solid #ebeef5;
}
.popup-sender h3 {
  font-size: 0.8rem;
  color: #606266;
  margin: 0.5rem 0;
}
.popup-input {
  display: block;
  width: 100%;
  padding: 4px 8px;
  border: 1px solid #dcdfe6;
  border-radius: 3px;
  font-size: 0.8rem;
  margin-bottom: 4px;
  outline: none;
}
.popup-sender button {
  padding: 4px 14px;
  background: #409eff;
  color: #fff;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8rem;
  margin-top: 4px;
}
.popup-sender button:disabled {
  background: #c0c4cc;
}
section {
  margin-bottom: 1rem;
}
section h3 {
  margin: 0 0 0.5rem 0;
  font-size: 0.85rem;
  color: #606266;
}
.hint {
  margin: 0.25rem 0;
  font-size: 0.75rem;
  color: #909399;
}
ul {
  list-style: none;
  padding: 0;
  margin: 0;
}
li {
  padding: 0.4rem 0.6rem;
  background: #f5f7fa;
  border-radius: 3px;
  margin-bottom: 0.25rem;
  font-size: 0.8rem;
  display: flex;
  gap: 0.5rem;
}
li.empty {
  background: transparent;
  color: #c0c4cc;
  font-style: italic;
}
code {
  font-family: ui-monospace, monospace;
  background: #fff;
  padding: 0.1rem 0.3rem;
  border-radius: 2px;
}
</style>
