<script setup lang="ts">
// ChatPanel —— 嵌入 EngagementPanel Chat tab 的聊天面板
//
// 1z 修 deep-audit P2-23 + 控制台 403 刷屏:
// session 未 claim 时 server 返 403 not_claimed,旧实现 catch ignore + 继续 2s 轮询,
// 控制台被 403 刷爆。新实现:遇 401/403 暂停轮询,UI 提示需 claim 才能聊天;
// sessionId 变化或 send 成功后恢复。
//
// Phase 3.5a:CSS 全量切 Calm token,气泡 operator=accent / visitor=subtle,
// input + button 用 .pc-input / .pc-btn 原子类。
import { ref, watch, onMounted, nextTick, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { PhPaperPlaneTilt } from '@phosphor-icons/vue';
import { listMessages, sendMessage, type ChatMessageItem } from '../api/sessions';

const { t } = useI18n();
const props = defineProps<{ sessionId: string | null }>();

const messages = ref<ChatMessageItem[]>([]);
const input = ref('');
const listEl = ref<HTMLDivElement | null>(null);
let pollTimer: ReturnType<typeof setInterval> | null = null;
let lastId = 0;
// pausedReason: '' / 'not_claimed' / 'auth_required' / 'unknown_error'
const pausedReason = ref<string>('');

async function refresh() {
  if (!props.sessionId) return;
  try {
    const resp = await listMessages(props.sessionId, lastId);
    // 成功一次,清掉 paused 状态(可能 claim 已恢复)
    if (pausedReason.value) pausedReason.value = '';
    if (resp.messages.length > 0) {
      messages.value.push(...resp.messages);
      lastId = resp.messages[resp.messages.length - 1].id;
      await nextTick();
      if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight;
    }
  } catch (err) {
    // 1z:401/403 → 暂停轮询(避免控制台 403 刷屏 + server 负载)
    // sessionId 变化或 send 成功后才会重新尝试
    const errObj = err as { message?: string };
    const msg = String(errObj?.message ?? err);
    if (msg.includes('HTTP 401')) {
      pausedReason.value = 'auth_required';
      stopPolling();
    } else if (msg.includes('HTTP 403')) {
      pausedReason.value = 'not_claimed';
      stopPolling();
    }
    // 其他错误静默(network blip),下个 tick 重试
  }
}

async function send() {
  if (!props.sessionId || !input.value.trim()) return;
  const content = input.value.trim();
  input.value = '';
  try {
    const msg = await sendMessage(props.sessionId, content);
    messages.value.push(msg);
    lastId = msg.id;
    await nextTick();
    if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight;
    // send 成功 → claim 已恢复 → 重新启动轮询
    if (pausedReason.value) {
      pausedReason.value = '';
      startPolling();
    }
  } catch {
    // ignore(send 时的错误不阻塞 UI)
  }
}

function startPolling() {
  if (pollTimer) return;
  pollTimer = setInterval(() => {
    if (props.sessionId) refresh();
  }, 2000);
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

watch(
  () => props.sessionId,
  (sid) => {
    messages.value = [];
    lastId = 0;
    pausedReason.value = '';
    stopPolling();
    if (sid) {
      refresh();
      startPolling();
    }
  },
);

onMounted(() => {
  if (props.sessionId) {
    refresh();
    startPolling();
  }
});

onUnmounted(() => {
  stopPolling();
});
</script>

<template>
  <div class="chat-panel">
    <div ref="listEl" class="message-list">
      <div v-for="m in messages" :key="m.id" :class="['msg', m.sender]">
        <span class="sender">{{ m.sender === 'operator' ? t('chat.operator') : t('chat.visitor') }}</span>
        <span class="content">{{ m.content }}</span>
      </div>
      <div v-if="messages.length === 0 && !pausedReason" class="empty">{{ t('chat.empty') }}</div>
      <div v-if="pausedReason === 'not_claimed'" class="empty paused">
        {{ t('chat.paused_claim_required') }}
      </div>
      <div v-else-if="pausedReason === 'auth_required'" class="empty paused">
        {{ t('chat.paused_auth_required') }}
      </div>
    </div>
    <form class="input-bar" @submit.prevent="send">
      <input
        v-model="input"
        type="text"
        class="pc-input chat-input"
        :placeholder="t('chat.placeholder')"
        :disabled="!sessionId || !!pausedReason"
      />
      <button
        type="submit"
        class="pc-btn pc-btn--primary send-btn"
        :disabled="!sessionId || !input.trim() || !!pausedReason"
        :aria-label="t('chat.send')"
      >
        <PhPaperPlaneTilt :size="16" weight="regular" aria-hidden="true" />
      </button>
    </form>
  </div>
</template>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--pc-color-bg-surface);
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--pc-space-component) var(--pc-space-section);
  display: flex;
  flex-direction: column;
  gap: var(--pc-space-field);
}

.msg {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: var(--pc-text-sm);
  max-width: 85%;
}

.msg.operator {
  align-self: flex-end;
  align-items: flex-end;
}

.msg.visitor {
  align-self: flex-start;
  align-items: flex-start;
}

.sender {
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
  padding: 0 4px;
}

.content {
  display: inline-block;
  padding: 6px 12px;
  border-radius: var(--pc-radius-lg);
  word-break: break-word;
  line-height: 1.4;
}

.msg.operator .content {
  background: var(--pc-color-accent-default);
  color: var(--pc-color-accent-on);
  border-bottom-right-radius: var(--pc-radius-sm);
}

.msg.visitor .content {
  background: var(--pc-color-bg-subtle);
  color: var(--pc-color-text-primary);
  border-bottom-left-radius: var(--pc-radius-sm);
}

.empty {
  text-align: center;
  color: var(--pc-color-text-muted);
  font-size: var(--pc-text-xs);
  padding: var(--pc-space-section);
  font-style: italic;
}

.empty.paused {
  color: var(--pc-color-warning);
  background: var(--pc-color-warning-subtle);
  border-radius: var(--pc-radius-md);
  font-style: normal;
}

.input-bar {
  display: flex;
  gap: var(--pc-space-field);
  padding: var(--pc-space-component) var(--pc-space-section);
  border-top: 1px solid var(--pc-color-border-default);
  background: var(--pc-color-bg-surface);
}

.chat-input {
  flex: 1;
  min-height: 36px;
}

.send-btn {
  flex-shrink: 0;
  padding: 0 12px;
  min-height: 36px;
}
</style>
