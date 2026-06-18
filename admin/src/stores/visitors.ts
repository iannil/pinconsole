// Pinia store：访客列表 + 选中访客 + 实时事件流

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { VisitorPresence } from '../composables/useWs';
import type { Envelope } from '@marketing-monitor/proto';
import type { EventPayload } from '@marketing-monitor/proto';

interface VisitorListItem {
  sessionId: string;
  visitorId: string;
  fingerprint: string;
  startedAt: number;
  lastEventAt: number | null;
  eventCount: number;
  // 1w P1-29:UI 消费 is_flagged 字段(后端 /api/sessions 已就绪)。
  // flagged session = BehaviorTracker 标记可疑,运营应警惕(可能是爬虫/刷量)。
  isFlagged?: boolean;
  flagReason?: string;
}

export const useVisitorsStore = defineStore('visitors', () => {
  const visitors = ref<Map<string, VisitorListItem>>(new Map());
  const selectedSessionId = ref<string | null>(null);
  const events = ref<Map<string, EventPayload[]>>(new Map());
  // 1f：navigated 事件的 old/new session ID，供 Dashboard 自动重订阅
  const navigatedFromId = ref<string | null>(null);
  const navigatedToId = ref<string | null>(null);

  const visitorList = computed<VisitorListItem[]>(() => {
    return Array.from(visitors.value.values()).sort(
      (a, b) => (b.lastEventAt ?? b.startedAt) - (a.lastEventAt ?? a.startedAt),
    );
  });

  const selectedVisitor = computed<VisitorListItem | null>(() => {
    if (!selectedSessionId.value) return null;
    return visitors.value.get(selectedSessionId.value) ?? null;
  });

  const selectedEvents = computed<EventPayload[]>(() => {
    if (!selectedSessionId.value) return [];
    return events.value.get(selectedSessionId.value) ?? [];
  });

  function setInitialList(items: VisitorListItem[]) {
    visitors.value = new Map(items.map((it) => [it.sessionId, it]));
  }

  function applyPresence(p: VisitorPresence) {
    const sessionId = p.sessionId;
    if (!sessionId) return;
    if (p.event === 'online') {
      // SDK reload(或 reconnect)会产生新 session,但 fingerprint 相同。
      // 检查同 fingerprint 的旧 session,删除 + 自动切换 selectedSessionId。
      // 否则 admin 选中的是旧 session(已离线),订阅了也收不到 events,
      // 用户体验是"订阅后 player 空"。
      let oldSameFp: string | null = null;
      for (const [sid, v] of visitors.value) {
        if (v.fingerprint === p.fingerprint && sid !== sessionId) {
          oldSameFp = sid;
          break;
        }
      }
      visitors.value.set(sessionId, {
        sessionId,
        visitorId: p.visitorId,
        fingerprint: p.fingerprint,
        startedAt: p.startedAt,
        lastEventAt: null,
        eventCount: 0,
      });
      if (oldSameFp) {
        visitors.value.delete(oldSameFp);
        // 清旧 session 的 events 避免陈旧数据混入
        events.value.delete(oldSameFp);
        events.value = new Map(events.value);
        // admin 选中的是同 fingerprint 旧 session → 自动切到新 session
        if (selectedSessionId.value === oldSameFp) {
          selectedSessionId.value = sessionId;
        }
      }
      visitors.value = new Map(visitors.value);
    } else if (p.event === 'offline') {
      // 从列表删除,但**不清 selectedSessionId**。
      // 原实现清 selectedSessionId 导致 admin 选中 visitor 后,SDK 重连短暂
      // 断线触发 offline → selectedSessionId=null → 整个 panel(含订阅按钮、
      // events area、player)消失,UX 极差。
      // 修复:selectedSessionId 保留,VisitorPanel 仍可见,只是该 session 不
      // 在 online 列表里(再次 online 时自动回来)。
      visitors.value.delete(sessionId);
    } else if (p.event === 'navigated') {
      // 1f：访客跳转到新页面。old session 下线，new session 上线。
      const oldId = p.oldSessionId;
      const newId = p.newSessionId;
      if (oldId) {
        visitors.value.delete(oldId);
      }
      if (newId) {
        visitors.value.set(newId, {
          sessionId: newId,
          visitorId: p.visitorId,
          fingerprint: p.fingerprint,
          startedAt: p.startedAt,
          lastEventAt: null,
          eventCount: 0,
        });
      }
      // 若当前选中的是 old session，自动切到 new session
      if (oldId && selectedSessionId.value === oldId && newId) {
        navigatedFromId.value = oldId;
        navigatedToId.value = newId;
        selectedSessionId.value = newId;
      }
    }
    visitors.value = new Map(visitors.value);
  }

  function select(sessionId: string | null) {
    selectedSessionId.value = sessionId;
  }

  function appendEvent(sessionId: string, env: Envelope) {
    const oldList = events.value.get(sessionId) ?? [];
    // 支持 single 与 batch（1c：SDK 批量发 array）
    const payloads: EventPayload[] = [];
    if (Array.isArray(env.payload)) {
      for (const p of env.payload as unknown[]) {
        if (p && typeof p === 'object' && 'type' in p) {
          payloads.push(p as EventPayload);
        }
      }
    } else if (env.payload && typeof env.payload === 'object' && 'type' in env.payload) {
      payloads.push(env.payload as EventPayload);
    }
    if (payloads.length === 0) return;

    // **创建新 array**(immutable update),让下游 computed/watch 能检测到引用变化。
    // 原实现 push 到旧 array + set 旧 reference,导致 ReplayPlayer 的 watch 不触发
    // (浅比较 array reference 不变)→ incremental events 永远不进 player。
    let newList = [...oldList, ...payloads];
    // 1c：rrweb 事件多，从 200 扩到 500
    if (newList.length > 500) newList = newList.slice(newList.length - 500);
    events.value.set(sessionId, newList);
    events.value = new Map(events.value);

    // 更新该 session 的元数据
    const item = visitors.value.get(sessionId);
    if (item) {
      visitors.value.set(sessionId, {
        ...item,
        lastEventAt: env.ts,
        eventCount: item.eventCount + payloads.length,
      });
      visitors.value = new Map(visitors.value);
    }
  }

  function clear() {
    visitors.value = new Map();
    events.value = new Map();
    selectedSessionId.value = null;
  }

  function clearNavigated() {
    navigatedFromId.value = null;
    navigatedToId.value = null;
  }

  return {
    visitors,
    visitorList,
    selectedSessionId,
    selectedVisitor,
    selectedEvents,
    events,
    navigatedFromId,
    navigatedToId,
    setInitialList,
    applyPresence,
    select,
    appendEvent,
    clear,
    clearNavigated,
  };
});
