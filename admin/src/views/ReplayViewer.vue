<script setup lang="ts">
// 单会话历史回放页（切片 1d）
// 用 rrweb-player 原生控制器 + 分页拉取所有事件

import { ref, onMounted, watch, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { getSessionReplay, type RRWebEvent } from '../api/sessions';

const { t } = useI18n();
const route = useRoute();

const sessionId = ref<string>(String(route.params.session_id ?? ''));
const loading = ref(true);
const loadingMore = ref(false);
const error = ref<string | null>(null);
const total = ref(0);
const hasMore = ref(false);
const events = ref<RRWebEvent[]>([]);

const playerContainer = ref<HTMLDivElement | null>(null);
// 用 unknown 透传 rrweb-player alpha 类型
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let player: any = null;

async function loadInitial() {
  loading.value = true;
  error.value = null;
  try {
    const resp = await getSessionReplay(sessionId.value, 0, 10000);
    events.value = resp.events ?? [];
    total.value = resp.total ?? 0;
    hasMore.value = resp.has_more ?? false;
    if (resp.has_more) {
      loadMore();
    }
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

async function loadMore() {
  if (!hasMore.value || loadingMore.value) return;
  loadingMore.value = true;
  try {
    const resp = await getSessionReplay(sessionId.value, events.value.length, 10000);
    const moreEvents = resp.events ?? [];
    events.value.push(...moreEvents);
    total.value = resp.total ?? events.value.length;
    hasMore.value = resp.has_more ?? false;
    if (player?.append && moreEvents.length > 0) {
      player.append(moreEvents);
    }
    if (resp.has_more) {
      setTimeout(() => loadMore(), 50);
    }
  } catch (e) {
    console.warn('loadMore failed', e);
  } finally {
    loadingMore.value = false;
  }
}

async function initPlayer() {
  if (!playerContainer.value || events.value.length === 0) return;
  // 动态 import rrweb-player（与 ReplayPlayer.vue 同策略）
  try {
    const mod = await import('rrweb-player');
    const Player = mod.default;
    // 清空旧 player
    playerContainer.value.replaceChildren();
    player = new Player({
      target: playerContainer.value,
      props: {
        events: events.value,
        showDebug: false,
        autoPlay: false,
        skipInactive: true,
      },
    });
  } catch (e) {
    console.error('rrweb-player init failed', e);
    error.value = t('replay.play_failed');
  }
}

watch(events, (newEvents) => {
  if (newEvents.length > 0 && !player) {
    initPlayer();
  }
}, { deep: false });

onMounted(loadInitial);

onUnmounted(() => {
  if (playerContainer.value) {
    playerContainer.value.replaceChildren();
  }
  player = null;
});

// 监听 sessionId 变化
watch(() => route.params.session_id, (newId) => {
  if (newId && newId !== sessionId.value) {
    sessionId.value = String(newId);
    if (playerContainer.value) playerContainer.value.replaceChildren();
    player = null;
    loadInitial();
  }
});
</script>

<template>
  <div class="replay-viewer">
    <header class="header">
      <RouterLink to="/replay" class="back">{{ t('replay.back') }}</RouterLink>
      <span class="session-info">
        {{ t('replay.session_label') }} <code>{{ sessionId }}</code>
        | {{ t('replay.events_label') }} {{ total }}
        <span v-if="loadingMore" class="loading-more">{{ t('replay.loading_more') }}</span>
      </span>
    </header>

    <div v-if="loading" class="placeholder">{{ t('replay.loading') }}</div>
    <div v-else-if="error" class="placeholder error">{{ error }}</div>
    <div v-else-if="events.length === 0" class="placeholder">{{ t('replay.no_events') }}</div>

    <div ref="playerContainer" class="player-container"></div>
  </div>
</template>

<style scoped>
.replay-viewer {
  font-family: system-ui, sans-serif;
  display: flex;
  flex-direction: column;
  height: 100vh;
}
.header {
  padding: 0.6rem 1rem;
  border-bottom: 1px solid #ebeef5;
  background: #fff;
  display: flex;
  align-items: center;
  gap: 1rem;
}
.back {
  color: #409eff;
  text-decoration: none;
  font-size: 0.85rem;
}
.session-info {
  font-size: 0.8rem;
  color: #606266;
}
.session-info code {
  font-family: ui-monospace, monospace;
  background: #f5f7fa;
  padding: 0.1rem 0.3rem;
  border-radius: 2px;
}
.loading-more {
  color: #e6a23c;
}
.placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
}
.placeholder.error {
  color: #f56c6c;
}
.player-container {
  flex: 1;
  background: #f5f7fa;
  overflow: auto;
}
</style>
