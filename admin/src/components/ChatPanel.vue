<script setup lang="ts">
// ChatPanel：嵌入 VisitorPanel 下方的聊天面板
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

async function refresh() {
  if (!props.sessionId) return;
  try {
    const resp = await listMessages(props.sessionId, lastId);
    if (resp.messages.length > 0) {
      messages.value.push(...resp.messages);
      lastId = resp.messages[resp.messages.length - 1].id;
      await nextTick();
      if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight;
    }
  } catch {
    // ignore
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
  } catch {
    // ignore
  }
}

watch(
  () => props.sessionId,
  (sid) => {
    messages.value = [];
    lastId = 0;
    if (sid) refresh();
  },
);

onMounted(() => {
  if (props.sessionId) refresh();
  pollTimer = setInterval(() => {
    if (props.sessionId) refresh();
  }, 2000);
});

import { onUnmounted } from 'vue';
onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer);
});
</script>

<template>
  <div class="chat-panel">
    <div ref="listEl" class="message-list">
      <div v-for="m in messages" :key="m.id" :class="['msg', m.sender]">
        <span class="sender">{{ m.sender === 'operator' ? t('chat.operator') : t('chat.visitor') }}</span>
        <span class="content">{{ m.content }}</span>
      </div>
      <div v-if="messages.length === 0" class="empty">{{ t('chat.empty') }}</div>
    </div>
    <div class="input-bar">
      <input
        v-model="input"
        type="text"
        :placeholder="t('chat.placeholder')"
        @keydown.enter="send"
        :disabled="!sessionId"
      />
      <button @click="send" :disabled="!sessionId || !input.trim()">{{ t('chat.send') }}</button>
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
