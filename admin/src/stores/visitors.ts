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
      visitors.value.set(sessionId, {
        sessionId,
        visitorId: p.visitorId,
        fingerprint: p.fingerprint,
        startedAt: p.startedAt,
        lastEventAt: null,
        eventCount: 0,
      });
    } else if (p.event === 'offline') {
      visitors.value.delete(sessionId);
      if (selectedSessionId.value === sessionId) {
        selectedSessionId.value = null;
      }
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
    const list = events.value.get(sessionId) ?? [];
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

    for (const p of payloads) list.push(p);
    // 1c：rrweb 事件多，从 200 扩到 500
    if (list.length > 500) list.splice(0, list.length - 500);
    events.value.set(sessionId, list);
    events.value = new Map(events.value);

    // 更新该 session 的元数据
    const item = visitors.value.get(sessionId);
    if (item) {
      item.lastEventAt = env.ts;
      item.eventCount += payloads.length;
      visitors.value.set(sessionId, { ...item });
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
