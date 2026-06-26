// 切片 1aa:visitors store 单元测试
// 覆盖 setInitialList / applyPresence / select / appendEvent / clear + navigated 锚点逻辑。

import { describe, it, expect, beforeEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useVisitorsStore } from '../src/stores/visitors';
import type { VisitorPresence } from '../src/composables/useWs';
import type { Envelope } from '@pinconsole/proto';

const basePresence = (over: Partial<VisitorPresence>): VisitorPresence => ({
  event: 'online',
  sessionId: 's1',
  visitorId: 'v1',
  fingerprint: 'fp1',
  startedAt: 1000,
  ...over,
});

function envelope(ts: number, payload: unknown): Envelope {
  return {
    v: 1,
    type: 'event',
    ts,
    session_id: 's1',
    payload,
  } as Envelope;
}

describe('visitors store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  describe('setInitialList', () => {
    it('populates visitor map', () => {
      const store = useVisitorsStore();
      store.setInitialList([
        {
          sessionId: 's1',
          visitorId: 'v1',
          fingerprint: 'fp1',
          startedAt: 1,
          lastEventAt: null,
          eventCount: 0,
        },
      ]);

      expect(store.visitorList.length).toBe(1);
      expect(store.visitorList[0]!.sessionId).toBe('s1');
    });

    it('replaces previous list on second call', () => {
      const store = useVisitorsStore();
      store.setInitialList([
        {
          sessionId: 's1',
          visitorId: 'v1',
          fingerprint: 'fp1',
          startedAt: 1,
          lastEventAt: null,
          eventCount: 0,
        },
      ]);
      store.setInitialList([
        {
          sessionId: 's2',
          visitorId: 'v2',
          fingerprint: 'fp2',
          startedAt: 2,
          lastEventAt: null,
          eventCount: 0,
        },
      ]);

      expect(store.visitorList.length).toBe(1);
      expect(store.visitorList[0]!.sessionId).toBe('s2');
    });
  });

  describe('applyPresence online', () => {
    it('adds new visitor to list', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({}));

      expect(store.visitorList.length).toBe(1);
      expect(store.visitorList[0]!.visitorId).toBe('v1');
    });

    it('removes old session with same fingerprint (SDK reload case)', () => {
      const store = useVisitorsStore();
      // 旧 session
      store.applyPresence(
        basePresence({ sessionId: 'old-s', fingerprint: 'fp-shared' }),
      );
      // 新 session 同 fp
      store.applyPresence(
        basePresence({ sessionId: 'new-s', fingerprint: 'fp-shared' }),
      );

      const ids = store.visitorList.map((v) => v.sessionId);
      expect(ids).toEqual(['new-s']);
    });

    it('auto-switches selectedSessionId to new session if fingerprint selected', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1', fingerprint: 'fp1' }));
      store.select('s1');
      expect(store.selectedSessionId).toBe('s1');

      // SDK reload 产生新 session 同 fp
      store.applyPresence(basePresence({ sessionId: 's2', fingerprint: 'fp1' }));

      // selectedFingerprint 内部锚点应驱动 selectedSessionId 切到新 session
      expect(store.selectedSessionId).toBe('s2');
    });
  });

  describe('applyPresence offline', () => {
    it('removes visitor from list but preserves selectedSessionId (v1-followups fix)', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1' }));
      store.select('s1');

      store.applyPresence(basePresence({ event: 'offline', sessionId: 's1' }));

      expect(store.visitorList.length).toBe(0);
      // 关键:不清 selectedSessionId,避免 panel 消失
      expect(store.selectedSessionId).toBe('s1');
    });

    it('no-op if session not in list', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ event: 'offline', sessionId: 'unknown' }));
      expect(store.visitorList.length).toBe(0);
    });
  });

  describe('applyPresence navigated', () => {
    it('removes old session and adds new session', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 'old-s' }));

      store.applyPresence(
        basePresence({
          event: 'navigated',
          sessionId: 'new-s',
          oldSessionId: 'old-s',
          newSessionId: 'new-s',
        }),
      );

      const ids = store.visitorList.map((v) => v.sessionId);
      expect(ids).toEqual(['new-s']);
    });

    it('auto-switches selectedSessionId from old to new', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 'old-s' }));
      store.select('old-s');

      store.applyPresence(
        basePresence({
          event: 'navigated',
          sessionId: 'new-s',
          oldSessionId: 'old-s',
          newSessionId: 'new-s',
        }),
      );

      expect(store.selectedSessionId).toBe('new-s');
      expect(store.navigatedFromId).toBe('old-s');
      expect(store.navigatedToId).toBe('new-s');
    });

    it('clearNavigated resets navigated state', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 'old-s' }));
      store.select('old-s');
      store.applyPresence(
        basePresence({
          event: 'navigated',
          sessionId: 'new-s',
          oldSessionId: 'old-s',
          newSessionId: 'new-s',
        }),
      );

      store.clearNavigated();

      expect(store.navigatedFromId).toBeNull();
      expect(store.navigatedToId).toBeNull();
    });
  });

  describe('select', () => {
    it('sets selectedSessionId and computes selectedVisitor', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1', fingerprint: 'fp1' }));

      store.select('s1');

      expect(store.selectedSessionId).toBe('s1');
      expect(store.selectedVisitor?.sessionId).toBe('s1');
      expect(store.selectedVisitor?.fingerprint).toBe('fp1');
    });

    it('select(null) clears selection', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1', fingerprint: 'fp1' }));
      store.select('s1');

      store.select(null);

      expect(store.selectedSessionId).toBeNull();
      expect(store.selectedVisitor).toBeNull();
    });

    it('selectedVisitor returns null for unknown session', () => {
      const store = useVisitorsStore();
      store.select('nonexistent');
      expect(store.selectedVisitor).toBeNull();
    });
  });

  describe('appendEvent', () => {
    it('appends single event payload', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1' }));

      store.appendEvent(
        's1',
        envelope(2000, { type: 2, timestamp: 2000, data: { x: 1 } }),
      );

      expect(store.selectedEvents).toHaveLength(0); // s1 not selected
      // 直接访问 events map
      const evs = store.events.get('s1');
      expect(evs).toHaveLength(1);
    });

    it('appends batch payload (array)', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1' }));

      store.appendEvent(
        's1',
        envelope(2000, [
          { type: 2, timestamp: 1, data: {} },
          { type: 2, timestamp: 2, data: {} },
          { type: 2, timestamp: 3, data: {} },
        ]),
      );

      expect(store.events.get('s1')).toHaveLength(3);
    });

    it('skips payload without type field', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1' }));

      store.appendEvent('s1', envelope(2000, { notype: true }));

      expect(store.events.get('s1') ?? []).toHaveLength(0);
    });

    it('updates lastEventAt and eventCount on visitor metadata', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1' }));

      store.appendEvent(
        's1',
        envelope(2000, { type: 2, timestamp: 2000, data: {} }),
      );

      const v = store.visitorList[0]!;
      expect(v.lastEventAt).toBe(2000);
      expect(v.eventCount).toBe(1);
    });

    it('keeps events within cap (cap=5000, 502 < cap so all retained)', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1' }));

      // 推 502 个事件
      for (let i = 0; i < 502; i++) {
        store.appendEvent(
          's1',
          envelope(i, { type: 2, timestamp: i, data: { i } }),
        );
      }

      const evs = store.events.get('s1')!;
      expect(evs).toHaveLength(502);
      // 保留全部 502 个 (cap=5000, i=0..501 全部保留)
      expect(evs[0]!.data).toEqual({ i: 0 });
      expect(evs[499]!.data).toEqual({ i: 499 });
    });
  });

  describe('clear', () => {
    it('empties all state', () => {
      const store = useVisitorsStore();
      store.applyPresence(basePresence({ sessionId: 's1' }));
      store.select('s1');
      store.appendEvent(
        's1',
        envelope(2000, { type: 2, timestamp: 2000, data: {} }),
      );

      store.clear();

      expect(store.visitorList).toHaveLength(0);
      expect(store.selectedSessionId).toBeNull();
      expect(store.events.size).toBe(0);
    });
  });

  describe('visitorList computed sorting', () => {
    it('sorts by lastEventAt desc (nulls use startedAt)', () => {
      const store = useVisitorsStore();
      store.applyPresence(
        basePresence({
          sessionId: 'a',
          fingerprint: 'fpa',
          startedAt: 1000,
        }),
      );
      store.applyPresence(
        basePresence({
          sessionId: 'b',
          fingerprint: 'fpb',
          startedAt: 2000,
        }),
      );

      // a 有更新事件
      store.appendEvent(
        'a',
        envelope(3000, { type: 2, timestamp: 3000, data: {} }),
      );

      const ids = store.visitorList.map((v) => v.sessionId);
      // a lastEventAt=3000 > b startedAt=2000
      expect(ids).toEqual(['a', 'b']);
    });
  });
});
