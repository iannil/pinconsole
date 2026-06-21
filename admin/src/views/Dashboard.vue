<script setup lang="ts">
// Dashboard —— 3-Column Smart Expand(design-system.md §4.2)
//
// 默认(未 claim):VisitorList + LiveColumn(2 栏)
// claim 后:VisitorList + LiveColumn + EngagementPanel(3 栏展开)
//
// Script 逻辑保留(claim heartbeat / toggleCoBrowsing / useWs subscribe),
// dashboard_wiring.test.ts 做源码契约校验,不能改这些 pattern。
import { onMounted, onUnmounted, computed, ref, watch } from 'vue';
import { useVisitorsStore } from '../stores/visitors';
import { useWs } from '../composables/useWs';
import VisitorList from '../components/VisitorList.vue';
import LiveColumn from '../components/LiveColumn.vue';
import EngagementPanel from '../components/EngagementPanel.vue';
import { useI18n } from 'vue-i18n';
import { apiFetch } from '../api/client';
import { claimSession, releaseSession, refreshClaim } from '../api/claim';

const { t } = useI18n();
const store = useVisitorsStore();

// 1e：co-browsing 控制状态
const coBrowsingActive = ref(false);
// claim 是否成功(用于错误提示 + UI 状态)
const claimError = ref('');

// Phase 3:cobrowse hint 文本由 Dashboard 算好传 LiveColumn
const cobrowseHint = computed(() =>
  coBrowsingActive.value
    ? t('dashboard.cobrowse_hint_active')
    : t('dashboard.cobrowse_hint_idle'),
);

// P1-claim-TTL 修复:claim active 时每 60s 续 TTL,避免 5min 自然过期。
// 续 TTL 失败(403 = claim 已丢)时,自动转 Start 状态 + 提示运营。
let refreshTimer: ReturnType<typeof setInterval> | null = null;
const REFRESH_INTERVAL_MS = 60_000;

function startClaimHeartbeat(sessionId: string) {
  stopClaimHeartbeat();
  refreshTimer = setInterval(async () => {
    const ok = await refreshClaim(sessionId);
    if (!ok) {
      // claim 已丢(TTL 过期 / 被他人 release)
      stopClaimHeartbeat();
      coBrowsingActive.value = false;
      claimError.value = t('dashboard.claim_lost_hint');
    }
  }, REFRESH_INTERVAL_MS);
}

function stopClaimHeartbeat() {
  if (refreshTimer !== null) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
}

async function toggleCoBrowsing() {
  if (!store.selectedSessionId) return;
  if (!coBrowsingActive.value) {
    // Start:先 claim,成功后才激活 overlay
    try {
      await claimSession(store.selectedSessionId);
      coBrowsingActive.value = true;
      claimError.value = '';
      startClaimHeartbeat(store.selectedSessionId);
    } catch (e) {
      claimError.value = (e as Error).message;
    }
  } else {
    // Stop:释放 claim + 关 overlay
    stopClaimHeartbeat();
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
      stopClaimHeartbeat();
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

onUnmounted(() => {
  stopClaimHeartbeat();
  close();
});

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
// player 渲染。Phase 3:订阅按钮从 UI 移除(自动订阅足够),逻辑保留。
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
</script>

<template>
  <div class="dashboard">
    <main class="main">
      <VisitorList :status="status" />

      <LiveColumn
        :co-browsing-active="coBrowsingActive"
        :claim-error="claimError"
        :cobrowse-hint="cobrowseHint"
        :operator-name="t('chat.operator')"
        @toggle-cobrowse="toggleCoBrowsing"
      />

      <Transition name="engage">
        <EngagementPanel
          v-if="coBrowsingActive && store.selectedSessionId"
          :session-id="store.selectedSessionId"
        />
      </Transition>
    </main>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.main {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
  background: var(--pc-color-bg-canvas);
}

/* Engagement panel slide-in transition(Gentle & Restrained:240ms ease-out) */
.engage-enter-active,
.engage-leave-active {
  transition: transform var(--pc-duration-slow) var(--pc-easing),
    opacity var(--pc-duration-slow) var(--pc-easing);
}

.engage-enter-from,
.engage-leave-to {
  transform: translateX(320px);
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .engage-enter-active,
  .engage-leave-active {
    transition-duration: 1ms;
  }
}
</style>
