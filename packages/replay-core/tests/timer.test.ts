/**
 * @vitest-environment jsdom
 *
 * Timer 类测试 — 驱动 replay 时间线的核心。
 * requestAnimationFrame / performance.now 需要 mock。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Timer, addDelay } from '../src/replay/timer';
import { EventType, IncrementalSource } from '../src/types';
import type { eventWithTime } from '../src/types';
import type { actionWithDelay } from '../src/rrweb-types';

// ── Helpers ──────────────────────────────────────────────

/** 创建一个简单的 actionWithDelay */
function makeAction(delay: number, label = ''): actionWithDelay {
  return { delay, doAction: vi.fn() };
}

/** 让 performance.now() 每次返回递增值（模拟时间推进）*/
let nowValue = 0;
function mockPerformanceNow(): number {
  return nowValue;
}

/** 模拟 RAF：直接同步执行回调，同时推进时间 */
let rafCb: FrameRequestCallback | null = null;
function mockRAF(cb: FrameRequestCallback): number {
  rafCb = cb;
  return 1;
}

/** 模拟 cancelAnimationFrame */
const mockCancelRAF = vi.fn();

beforeEach(() => {
  nowValue = 0;
  rafCb = null;
  vi.stubGlobal('requestAnimationFrame', mockRAF);
  vi.stubGlobal('cancelAnimationFrame', mockCancelRAF);
  vi.stubGlobal(
    'performance',
    { now: mockPerformanceNow },
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  mockCancelRAF.mockClear();
});

// ── Timer ────────────────────────────────────────────────

describe('Timer', () => {
  describe('constructor', () => {
    it('stores actions and config', () => {
      const actions = [makeAction(100)];
      const t = new Timer(actions, { speed: 2 });
      expect(t.speed).toBe(2);
      expect(t.isActive()).toBe(false);
    });

    it('defaults to empty actions array', () => {
      const t = new Timer(undefined, { speed: 1 });
      expect(t.isActive()).toBe(false);
    });
  });

  describe('addAction', () => {
    it('appends to empty list', () => {
      const t = new Timer(undefined, { speed: 1 });
      const a = makeAction(100);
      t.addAction(a);
      // no way to inspect private actions directly, but we test via start
      expect(t.isActive()).toBe(false);
    });

    it('appends in order (fast track)', () => {
      const t = new Timer([makeAction(100), makeAction(200)], { speed: 1 });
      t.addAction(makeAction(150));
      // After RAF not active, addAction doesn't restart RAF
      expect(t.isActive()).toBe(false);
    });

    it('restarts RAF if was active (raf===true)', () => {
      // Simulate raf=true state by creating a Timer, starting it,
      // then clearing it (which sets raf=true)
      const t = new Timer([makeAction(100)], { speed: 1 });
      t.start();
      // After start, rafCb is set. Run rafCb to exhaust actions and set raf=true.
      rafCb!(0);
      // Now raf === true
      t.addAction(makeAction(200));
      // Should have restarted RAF
      expect(t.isActive()).toBe(true);
    });
  });

  describe('start + rafCheck', () => {
    it('sets timeOffset to 0 and starts RAF', () => {
      const t = new Timer([makeAction(100)], { speed: 1 });
      t.start();
      expect(t.timeOffset).toBe(0);
      expect(t.isActive()).toBe(true);
    });

    it('sets raf=true (was active) when all actions exhausted', () => {
      const a1 = makeAction(0, 'a1');
      const a2 = makeAction(50, 'a2');
      const a3 = makeAction(100, 'a3');
      const t = new Timer([a1, a2, a3], { speed: 1 });
      t.start();

      // Advance time to 60ms, run rafCheck
      nowValue = 60;
      rafCb!(1000);
      // a1 (delay=0) and a2 (delay=50) should fire
      expect(a1.doAction).toHaveBeenCalledTimes(1);
      expect(a2.doAction).toHaveBeenCalledTimes(1);
      expect(a3.doAction).not.toHaveBeenCalled();
      // Still has a3 remaining → RAF still running
      expect(t.isActive()).toBe(true);

      // Advance to 120ms
      nowValue = 120;
      rafCb!(1000);
      expect(a3.doAction).toHaveBeenCalledTimes(1);
      // All actions exhausted → raf=true (was active, not null)
      expect(t.isActive()).toBe(true);
    });

    it('applies speed multiplier', () => {
      const a1 = makeAction(200, 'a1');
      const t = new Timer([a1], { speed: 2 }); // 2x speed
      t.start();

      // At 50ms real time, timeOffset = 50 * 2 = 100 (not yet 200)
      nowValue = 50;
      rafCb!(0);
      expect(a1.doAction).not.toHaveBeenCalled();

      // At 100ms real time, timeOffset = 100 * 2 = 200
      nowValue = 100;
      rafCb!(0);
      expect(a1.doAction).toHaveBeenCalledTimes(1);
    });
  });

  describe('clear', () => {
    it('cancels RAF and clears actions', () => {
      const t = new Timer([makeAction(100)], { speed: 1 });
      t.start();
      expect(t.isActive()).toBe(true);

      t.clear();
      expect(t.isActive()).toBe(false);
      expect(mockCancelRAF).toHaveBeenCalled();
    });

    it('handles raf===true state (already finished but was active)', () => {
      const t = new Timer([makeAction(0)], { speed: 1 });
      t.start();
      rafCb!(0); // exhaust action, raf becomes true
      expect(t.isActive()).toBe(true); // raf===true

      t.clear();
      expect(t.isActive()).toBe(false);
      // cancelAnimationFrame not called when raf===true (it's not a number)
      expect(mockCancelRAF).not.toHaveBeenCalled();
    });

    it('is safe to call when already cleared', () => {
      const t = new Timer(undefined, { speed: 1 });
      t.clear();
      t.clear(); // no throw
    });
  });

  describe('setSpeed', () => {
    it('updates speed', () => {
      const t = new Timer(undefined, { speed: 1 });
      expect(t.speed).toBe(1);
      t.setSpeed(4);
      expect(t.speed).toBe(4);
    });
  });

  describe('isActive', () => {
    it('returns false before start', () => {
      const t = new Timer(undefined, { speed: 1 });
      expect(t.isActive()).toBe(false);
    });

    it('returns true after start', () => {
      const t = new Timer([makeAction(1000)], { speed: 1 });
      t.start();
      expect(t.isActive()).toBe(true);
    });

    it('returns false after clear', () => {
      const t = new Timer([makeAction(1000)], { speed: 1 });
      t.start();
      t.clear();
      expect(t.isActive()).toBe(false);
    });
  });
});

// ── addDelay ─────────────────────────────────────────────

describe('addDelay', () => {
  it('calculates delay from timestamp and baselineTime', () => {
    const event = {
      type: EventType.FullSnapshot,
      timestamp: 2000,
      data: {},
    } as unknown as eventWithTime;
    addDelay(event, 1000);
    expect(event.delay).toBe(1000);
  });

  it('returns the delay value', () => {
    const event = {
      type: EventType.Meta,
      timestamp: 500,
      data: {},
    } as unknown as eventWithTime;
    const result = addDelay(event, 200);
    expect(result).toBe(300);
  });

  it('handles MouseMove with positions timeOffset', () => {
    const event = {
      type: EventType.IncrementalSnapshot,
      timestamp: 5000,
      data: {
        source: IncrementalSource.MouseMove,
        positions: [{ timeOffset: -100 }],
      },
    } as unknown as eventWithTime;
    addDelay(event, 2000);
    // firstOffset = -100, firstTimestamp = 5000 + (-100) = 4900
    // delay = 4900 - 2000 = 2900
    expect(event.delay).toBe(2900);
  });

  it('uses earliest position timeOffset for MouseMove', () => {
    const event = {
      type: EventType.IncrementalSnapshot,
      timestamp: 5000,
      data: {
        source: IncrementalSource.MouseMove,
        positions: [{ timeOffset: -500 }, { timeOffset: -200 }],
      },
    } as unknown as eventWithTime;
    addDelay(event, 1000);
    // firstOffset = -500, firstTimestamp = 5000 - 500 = 4500
    // delay = 4500 - 1000 = 3500
    expect(event.delay).toBe(3500);
  });

  it('handles non-MouseMove incremental events normally', () => {
    const event = {
      type: EventType.IncrementalSnapshot,
      timestamp: 3000,
      data: {
        source: IncrementalSource.Mutation,
      },
    } as unknown as eventWithTime;
    addDelay(event, 1000);
    expect(event.delay).toBe(2000);
  });
});
