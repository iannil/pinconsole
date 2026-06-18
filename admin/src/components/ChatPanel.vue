<script setup lang="ts">
// ChatPanel：嵌入 VisitorPanel 下方的聊天面板
//
// 1z 修 deep-audit P2-23 + 控制台 403 刷屏:
// session 未 claim 时 server 返 403 not_claimed,旧实现 catch ignore + 继续 2s 轮询,
// 控制台被 403 刷爆。新实现:遇 401/403 暂停轮询,UI 提示需 claim 才能聊天;
// sessionId 变化或 send 成功后恢复。
import { ref, watch, onMounted, nextTick } from 'vue';
import { useI18n } from 'vue-i18n';
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

import { onUnmounted } from 'vue';
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
    <div class="input-bar">
      <input
        v-model="input"
        type="text"
        :placeholder="t('chat.placeholder')"
        @keydown.enter="send"
        :disabled="!sessionId || !!pausedReason"
      />
      <button @click="send" :disabled="!sessionId || !input.trim() || !!pausedReason">
        {{ t('chat.send') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  height: 200px;
  border-top: 1px solid #ebeef5;
}
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
}
.msg {
  margin-bottom: 6px;
  font-size: 13px;
}
.msg.operator {
  text-align: right;
}
.sender {
  font-size: 11px;
  color: #909399;
  margin-right: 4px;
}
.content {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 6px;
  max-width: 75%;
  word-break: break-word;
}
.msg.operator .content {
  background: #409eff;
  color: #fff;
}
.msg.visitor .content {
  background: #f5f7fa;
  color: #303133;
}
.empty {
  text-align: center;
  color: #c0c4cc;
  font-size: 12px;
  padding: 1rem;
}
.input-bar {
  display: flex;
  gap: 4px;
  padding: 8px;
  border-top: 1px solid #ebeef5;
}
input {
  flex: 1;
  padding: 4px 8px;
  border: 1px solid #dcdfe6;
  border-radius: 3px;
  font-size: 13px;
  outline: none;
}
input:focus {
  border-color: #409eff;
}
button {
  padding: 4px 14px;
  background: #409eff;
  color: #fff;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  font-size: 13px;
}
button:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}
</style>
