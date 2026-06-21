<script setup lang="ts">
// EngagementPanel —— Dashboard 右列(claim 后展开)
// 来源:docs/design-system.md §4.2 (3-Column Smart Expand)
//
// 3 个 tab:
// - Chat(默认):接 ChatPanel
// - Popup:推送弹窗(4 输入 + 发送)—— 从旧 VisitorPanel 迁来
// - Forms:最近表单提交 —— 从旧 VisitorPanel 迁来
import { ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { PhChatsCircle, PhMegaphone, PhClipboardText } from '@phosphor-icons/vue';
import { useVisitorsStore } from '../stores/visitors';
import { sendCommand } from '../api/sessions';
import ChatPanel from './ChatPanel.vue';

type TabId = 'chat' | 'popup' | 'forms';

const { t } = useI18n();
const store = useVisitorsStore();

const props = defineProps<{
  /** 当前会话 ID(传给 ChatPanel + popup sendCommand) */
  sessionId: string | null;
}>();

const activeTab = ref<TabId>('chat');

const tabs = computed(() => [
  { id: 'chat' as const, label: t('engagement.tab_chat'), Icon: PhChatsCircle },
  { id: 'popup' as const, label: t('engagement.tab_popup'), Icon: PhMegaphone },
  { id: 'forms' as const, label: t('engagement.tab_forms'), Icon: PhClipboardText },
]);

// ===== Popup tab state =====
const popupTitle = ref('');
const popupBody = ref('');
const popupActionLabel = ref('');
const popupActionUrl = ref('');
const popupSending = ref(false);

async function sendPopup() {
  if (!props.sessionId || !popupTitle.value.trim()) return;
  popupSending.value = true;
  try {
    await sendCommand(props.sessionId, 'show_popup', {
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
  } finally {
    popupSending.value = false;
  }
}

// ===== Forms tab data =====
const selectedEvents = computed(() => store.selectedEvents);
const recentFormSubmits = computed(() =>
  selectedEvents.value.filter((e) => e.type === 'form_submit' && e.form_submit).slice(-10).reverse(),
);
</script>

<template>
  <aside class="engagement-panel" aria-label="engagement">
    <header class="panel-header">
      <span class="title">{{ t('engagement.title') }}</span>
    </header>

    <nav class="tabs" role="tablist">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        class="tab"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        <component :is="tab.Icon" :size="16" weight="regular" aria-hidden="true" />
        <span>{{ tab.label }}</span>
      </button>
    </nav>

    <div class="tab-body">
      <!-- Chat tab -->
      <div v-show="activeTab === 'chat'" class="tab-content chat-tab" role="tabpanel">
        <ChatPanel :session-id="sessionId" />
      </div>

      <!-- Popup tab -->
      <div v-show="activeTab === 'popup'" class="tab-content popup-tab" role="tabpanel">
        <label class="pc-label" for="popup-title">{{ t('visitor.popup_title_ph') }}</label>
        <input
          id="popup-title"
          v-model="popupTitle"
          class="pc-input"
          :placeholder="t('visitor.popup_title_ph')"
        />

        <label class="pc-label" for="popup-body">{{ t('visitor.popup_body_ph') }}</label>
        <input
          id="popup-body"
          v-model="popupBody"
          class="pc-input"
          :placeholder="t('visitor.popup_body_ph')"
        />

        <label class="pc-label" for="popup-action">{{ t('visitor.popup_action_ph') }}</label>
        <input
          id="popup-action"
          v-model="popupActionLabel"
          class="pc-input"
          :placeholder="t('visitor.popup_action_ph')"
        />

        <label class="pc-label" for="popup-url">{{ t('visitor.popup_url_ph') }}</label>
        <input
          id="popup-url"
          v-model="popupActionUrl"
          class="pc-input"
          :placeholder="t('visitor.popup_url_ph')"
        />

        <button
          type="button"
          class="pc-btn pc-btn--primary send-btn"
          :disabled="!popupTitle.trim() || popupSending"
          @click="sendPopup"
        >
          {{ t('visitor.send_popup') }}
        </button>
      </div>

      <!-- Forms tab -->
      <div v-show="activeTab === 'forms'" class="tab-content forms-tab" role="tabpanel">
        <p class="forms-count">
          {{ t('visitor.recent_form_submits', { count: recentFormSubmits.length }) }}
        </p>
        <ul v-if="recentFormSubmits.length > 0" class="form-list">
          <li v-for="(f, idx) in recentFormSubmits" :key="idx" class="form-item">
            <code class="pc-mono">{{ f.form_submit!.form_id || '(no id)' }}</code>
            <span class="field-count">
              {{ t('visitor.field_count', { count: Object.keys(f.form_submit!.fields).length }) }}
            </span>
          </li>
        </ul>
        <p v-else class="empty">{{ t('visitor.no_form_submits') }}</p>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.engagement-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 320px;
  flex-shrink: 0;
  background: var(--pc-color-bg-surface);
  border-left: 1px solid var(--pc-color-border-default);
}

.panel-header {
  padding: var(--pc-space-component) var(--pc-space-section);
  border-bottom: 1px solid var(--pc-color-border-default);
}

.title {
  font-size: var(--pc-text-sm);
  font-weight: var(--pc-weight-semibold);
  color: var(--pc-color-text-primary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.tabs {
  display: flex;
  border-bottom: 1px solid var(--pc-color-border-default);
  padding: 0 var(--pc-space-component);
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: var(--pc-space-component) 8px;
  font-size: var(--pc-text-sm);
  font-weight: var(--pc-weight-medium);
  color: var(--pc-color-text-muted);
  position: relative;
  transition: color var(--pc-duration-fast) var(--pc-easing);
}

.tab:hover {
  color: var(--pc-color-text-secondary);
}

.tab.active {
  color: var(--pc-color-accent-default);
}

.tab.active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 2px;
  background: var(--pc-color-accent-default);
  border-radius: var(--pc-radius-pill);
}

.tab-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tab-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Chat tab:ChatPanel 自身 flex column */
.chat-tab {
  padding: 0;
}

/* Popup tab:表单滚动 */
.popup-tab {
  padding: var(--pc-space-component) var(--pc-space-section);
  gap: 4px;
  overflow-y: auto;
}

.popup-tab .pc-label {
  margin-top: var(--pc-space-field);
  margin-bottom: 4px;
}

.popup-tab .pc-label:first-child {
  margin-top: 0;
}

.send-btn {
  margin-top: var(--pc-space-component);
  align-self: flex-start;
}

/* Forms tab */
.forms-tab {
  padding: var(--pc-space-component) var(--pc-space-section);
  overflow-y: auto;
}

.forms-count {
  margin: 0 0 var(--pc-space-component);
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.form-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.form-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--pc-space-component);
  padding: 6px 10px;
  background: var(--pc-color-bg-subtle);
  border-radius: var(--pc-radius-sm);
  font-size: var(--pc-text-xs);
}

.form-item code {
  color: var(--pc-color-text-primary);
  font-weight: var(--pc-weight-medium);
}

.field-count {
  color: var(--pc-color-text-muted);
}

.empty {
  margin: 0;
  padding: var(--pc-space-section);
  text-align: center;
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
  font-style: italic;
}
</style>
