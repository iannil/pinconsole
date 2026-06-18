<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref, watch } from 'vue';
import { useVisitorsStore } from '../stores/visitors';
import { useWs, type WsStatus } from '../composables/useWs';
import VisitorList from '../components/VisitorList.vue';
import VisitorPanel from '../components/VisitorPanel.vue';
import CoBrowseOverlay from '../components/CoBrowseOverlay.vue';
import { useI18n } from 'vue-i18n';
import { apiFetch } from '../api/client';
import { claimSession, releaseSession } from '../api/claim';

const { t, locale, availableLocales } = useI18n();
const store = useVisitorsStore();

// 1j:语言切换按钮 - 在中英之间 toggle
function toggleLocale() {
  const currentIdx = availableLocales.indexOf(locale.value);
  const nextIdx = (currentIdx + 1) % availableLocales.length;
  locale.value = availableLocales[nextIdx];
}

// 1e：co-browsing 控制状态
const coBrowsingActive = ref(false);
// claim 是否成功(用于错误提示 + UI 状态)
const claimError = ref('');

async function toggleCoBrowsing() {
  if (!store.selectedSessionId) return;
  if (!coBrowsingActive.value) {
    // Start:先 claim,成功后才激活 overlay
    try {
      await claimSession(store.selectedSessionId);
      coBrowsingActive.value = true;
      claimError.value = '';
    } catch (e) {
      claimError.value = (e as Error).message;
    }
  } else {
    // Stop:释放 claim + 关 overlay
    try {
      await releaseSession(store.selectedSessionId);
    } finally {
      coBrowsingActive.value = false;
    }
  }
}

// 切换访客时,先释放当前 claim
watch(
  () => store.selectedSessionId,
  async (_newId, oldId) => {
    if (oldId && coBrowsingActive.value) {
      try { await releaseSession(oldId); } catch { /* 已结束/已 release */ }
      coBrowsingActive.value = false;
    }
  },
);

const wsEndpoint = computed(() => {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${location.host}/ws/operator`;
});

const {
  status,
  connect,
  close,
  subscribe,
  unsubscribe,
} = useWs({
  endpoint: wsEndpoint.value,
  onPresence: (p) => {
    store.applyPresence({
      event: p.event,
      sessionId: p.session_id,
      visitorId: p.visitor_id,
      fingerprint: p.fingerprint,
      startedAt: p.started_at,
    });
  },
  onEvent: (e) => {
    if (e.envelope.payload) {
      store.appendEvent(e.sessionId, e.envelope);
    }
  },
});

// 初始拉取访客列表（REST），随后 WS 推送增量
// 1z P1-1:改用 apiFetch 自动注入 X-Trace-Id(原来用裸 fetch 不带 trace_id)
async function fetchInitial() {
  try {
    const { response } = await apiFetch('/api/sessions');
    if (!response.ok) return;
    const data = await response.json();
    store.setInitialList(
      (data.sessions ?? []).map((s: Record<string, unknown>) => ({
        sessionId: String(s.session_id ?? ''),
        visitorId: String(s.visitor_id ?? ''),
        fingerprint: String(s.fingerprint ?? ''),
        startedAt: Number(s.started_at ?? Date.now()),
        lastEventAt:
          typeof s.last_event_at === 'number' ? Number(s.last_event_at) : null,
        eventCount: Number(s.event_count ?? 0),
        // 1w P1-29:消费后端 is_flagged 字段
        isFlagged: Boolean(s.is_flagged),
        flagReason: s.flag_reason ? String(s.flag_reason) : undefined,
      })),
    );
  } catch (e) {
    console.warn('fetch sessions failed', e);
  }
}

onMounted(async () => {
  await fetchInitial();
  connect();
});

onUnmounted(() => close());

// 1f：navigated 自动重订阅
watch(
  () => store.navigatedToId,
  (newId, _oldId) => {
    if (!newId) return;
    const oldId = store.navigatedFromId;
    if (oldId) {
      unsubscribe(oldId);
    }
    subscribe(newId);
    store.clearNavigated();
  },
);

// 选中 visitor 自动订阅(取消手动 click "订阅实时" 步骤)。
// 原 UX:user click visitor → 看到 panel → 找 "订阅实时" 按钮(底部可能滚动不到)
// → click → 等 events → player 渲染。步骤多,易踩坑。
// 改:user click visitor → 自动 unsubscribe 旧 + subscribe 新 → 等 events →
// player 渲染。subscribe-bar 按钮保留(可手动取消订阅)。
watch(
  () => store.selectedSessionId,
  (newId, oldId) => {
    if (oldId && oldId !== newId) {
      unsubscribe(oldId);
    }
    if (newId) {
      subscribe(newId);
    }
  },
);

function statusBadgeClass(s: WsStatus): string {
  return `status-badge status-${s}`;
}
</script>

<template>
  <div class="dashboard">
    <header class="top-bar">
      <span class="title">{{ t('app.title') }}</span>
      <span :class="statusBadgeClass(status)">{{ status }}</span>
      <RouterLink to="/replay" class="nav-link">{{ t('nav.replay') }}</RouterLink>
      <button class="lang-switch" @click="toggleLocale">{{ t('app.switch_lang') }}</button>
    </header>
    <main class="main">
      <VisitorList />
      <div class="panel-wrapper">
        <VisitorPanel />
        <!-- 1e：CoBrowseOverlay 覆盖在 VisitorPanel 上 -->
        <CoBrowseOverlay
          v-if="store.selectedSessionId"
          :session-id="store.selectedSessionId"
          :active="coBrowsingActive"
          :operator-name="t('chat.operator')"
        />
        <div v-if="store.selectedSessionId" class="subscribe-bar">
          <button @click="subscribe(store.selectedSessionId!)">{{ t('dashboard.subscribe') }}</button>
          <button @click="unsubscribe(store.selectedSessionId!)">{{ t('dashboard.unsubscribe') }}</button>
          <button
            v-if="store.selectedSessionId"
            class="cobrowse-btn"
            :class="{ active: coBrowsingActive }"
            :disabled="!store.selectedSessionId"
            @click="toggleCoBrowsing"
          >
            {{ coBrowsingActive ? t('dashboard.stop_cobrowse') : t('dashboard.start_cobrowse') }}
          </button>
          <span class="hint">
            {{ coBrowsingActive ? t('dashboard.cobrowse_hint_active') : t('dashboard.cobrowse_hint_idle') }}
          </span>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  height: 100vh;
}
.top-bar {
  padding: 0.6rem 1rem;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  gap: 1rem;
}
.title {
  font-weight: 600;
  font-size: 0.95rem;
}
.status-badge {
  padding: 0.15rem 0.6rem;
  border-radius: 10px;
  font-size: 0.75rem;
  font-family: ui-monospace, monospace;
  background: #f5f7fa;
  color: #606266;
}
.status-connected {
  background: #e1f3d8;
  color: #67c23a;
}
.status-connecting,
.status-reconnecting {
  background: #fdf6ec;
  color: #e6a23c;
}
.status-closed,
.status-idle {
  background: #fef0f0;
  color: #f56c6c;
}
.nav-link {
  margin-left: auto;
  padding: 0.3rem 0.7rem;
  text-decoration: none;
  color: #409eff;
  font-size: 0.8rem;
}
.cobrowse-btn.active {
  background: #f56c6c;
  border-color: #f56c6c;
  color: #fff;
}
.panel-wrapper {
  position: relative;
}
.main {
  flex: 1;
  display: flex;
  overflow: hidden;
}
.panel-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.subscribe-bar {
  padding: 0.5rem 1rem;
  background: #f5f7fa;
  border-top: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
button {
  padding: 0.3rem 0.8rem;
  border: 1px solid #409eff;
  background: #fff;
  color: #409eff;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8rem;
}
button:hover {
  background: #ecf5ff;
}
.hint {
  color: #909399;
  font-size: 0.75rem;
}
</style>
