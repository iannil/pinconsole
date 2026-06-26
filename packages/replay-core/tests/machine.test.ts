/**
 * @vitest-environment jsdom
 *
 * 状态机测试 — createPlayerService / createSpeedService / discardPriorSnapshots。
 * xstate/fsm interpret() 生成可订阅状态机，测试状态转换和 action 调用。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  discardPriorSnapshots,
  createPlayerService,
  createSpeedService,
  type PlayerContext,
  type SpeedContext,
} from '../src/replay/machine';
import { EventType, IncrementalSource, ReplayerEvents } from '../src/types';
import { Timer } from '../src/replay/timer';
import type { eventWithTime, Emitter } from '../src/types';

// ── Helpers ──────────────────────────────────────────────

function makeMeta(ts: number): eventWithTime {
  return { type: EventType.Meta, timestamp: ts, data: { width: 1024, height: 768, href: 'http://localhost' } } as eventWithTime;
}

function makeFullSnapshot(ts: number): eventWithTime {
  return { type: EventType.FullSnapshot, timestamp: ts, data: {} } as eventWithTime;
}

function makeIncremental(ts: number): eventWithTime {
  return { type: EventType.IncrementalSnapshot, timestamp: ts, data: { source: IncrementalSource.Mutation } } as eventWithTime;
}

/** Mock emitter that captures emitted events */
function mockEmitter(): Emitter {
  const handlers = new Map<string, Array<(...args: any[]) => void>>();
  return {
    on(event: string, handler: (...args: any[]) => void) {
      if (!handlers.has(event)) handlers.set(event, []);
      handlers.get(event)!.push(handler);
    },
    emit(event: string, ...args: any[]) {
      handlers.get(event)?.forEach((h) => h(...args));
    },
    off(event: string, handler: (...args: any[]) => void) {
      const arr = handlers.get(event);
      if (arr) {
        const idx = arr.indexOf(handler);
        if (idx >= 0) arr.splice(idx, 1);
      }
    },
  };
}

function makePlayerAssets() {
  return {
    emitter: mockEmitter(),
    applyEventsSynchronously: vi.fn(),
    getCastFn: vi.fn(() => vi.fn()),
  };
}

// ── discardPriorSnapshots ────────────────────────────────

describe('discardPriorSnapshots', () => {
  it('returns all events when no Meta event', () => {
    const events = [makeIncremental(100), makeIncremental(200)];
    const result = discardPriorSnapshots(events, 0);
    expect(result).toBe(events);
  });

  it('returns events from last Meta <= baselineTime', () => {
    const events = [makeMeta(100), makeIncremental(150), makeMeta(200), makeIncremental(250), makeIncremental(300)];
    const result = discardPriorSnapshots(events, 250);
    // last Meta with timestamp <= 250 is at index 2 (ts=200)
    expect(result).toHaveLength(3);
    expect(result[0].timestamp).toBe(200);
    expect((result[0] as eventWithTime).type).toBe(EventType.Meta);
  });

  it('returns from first Meta when baselineTime is before all', () => {
    const events = [makeMeta(500), makeIncremental(600)];
    const result = discardPriorSnapshots(events, 1000);
    expect(result).toHaveLength(2);
  });

  it('returns empty slice when events empty', () => {
    expect(discardPriorSnapshots([], 100)).toEqual([]);
  });

  it('handles single Meta event before baseline', () => {
    const events = [makeMeta(100), makeIncremental(200)];
    const result = discardPriorSnapshots(events, 150);
    expect(result).toHaveLength(2);
  });
});

// ── createPlayerService ──────────────────────────────────

describe('createPlayerService', () => {
  let mockAssets: ReturnType<typeof makePlayerAssets>;

  function makeCtx(overrides: Partial<PlayerContext> = {}): PlayerContext {
    return {
      events: [makeMeta(0), makeFullSnapshot(100), makeIncremental(200), makeIncremental(300)],
      timer: new Timer(undefined, { speed: 1 }),
      timeOffset: 0,
      baselineTime: 0,
      lastPlayedEvent: null,
      ...overrides,
    };
  }

  beforeEach(() => {
    mockAssets = makePlayerAssets();
  });

  it('starts in paused state', () => {
    const service = createPlayerService(makeCtx(), mockAssets);
    service.start();
    expect(service.state.value).toBe('paused');
  });

  it('transitions to playing on PLAY', () => {
    const ctx = makeCtx();
    const service = createPlayerService(ctx, mockAssets);
    service.start();
    service.send({ type: 'PLAY', payload: { timeOffset: 0 } });
    expect(service.state.value).toBe('playing');
  });

  it('emits Flush + starts timer on PLAY', () => {
    const ctx = makeCtx();
    const emitSpy = vi.spyOn(mockAssets.emitter, 'emit');
    const timerStartSpy = vi.spyOn(ctx.timer, 'start');
    const service = createPlayerService(ctx, mockAssets);
    service.start();
    service.send({ type: 'PLAY', payload: { timeOffset: 0 } });

    expect(emitSpy).toHaveBeenCalledWith(ReplayerEvents.Flush);
    expect(timerStartSpy).toHaveBeenCalled();
  });

  it('transitions to paused on PAUSE', () => {
    const service = createPlayerService(makeCtx(), mockAssets);
    service.start();
    service.send({ type: 'PLAY', payload: { timeOffset: 0 } });
    service.send({ type: 'PAUSE' });
    expect(service.state.value).toBe('paused');
  });

  it('clears timer on PAUSE', () => {
    const ctx = makeCtx();
    const clearSpy = vi.spyOn(ctx.timer, 'clear');
    const service = createPlayerService(ctx, mockAssets);
    service.start();
    service.send({ type: 'PLAY', payload: { timeOffset: 0 } });
    service.send({ type: 'PAUSE' });
    expect(clearSpy).toHaveBeenCalled();
  });

  it('handles END transition: resets lastPlayedEvent and pauses', () => {
    const ctx = makeCtx({ lastPlayedEvent: makeIncremental(200) });
    const clearSpy = vi.spyOn(ctx.timer, 'clear');
    const service = createPlayerService(ctx, mockAssets);
    service.start();
    service.send({ type: 'PLAY', payload: { timeOffset: 0 } });
    service.send({ type: 'END' });

    expect(service.state.value).toBe('paused');
    expect(service.state.context.lastPlayedEvent).toBeNull();
    expect(clearSpy).toHaveBeenCalled();
  });

  it('transitions to live on TO_LIVE', () => {
    const service = createPlayerService(makeCtx(), mockAssets);
    service.start();
    service.send({ type: 'TO_LIVE', payload: {} });
    expect(service.state.value).toBe('live');
  });

  it('stays in live on ADD_EVENT', () => {
    const service = createPlayerService(makeCtx(), mockAssets);
    service.start();
    service.send({ type: 'TO_LIVE', payload: {} });
    service.send({ type: 'ADD_EVENT', payload: { event: makeIncremental(400) } });
    expect(service.state.value).toBe('live');
  });

  it('adds event to context events array', () => {
    const ctx = makeCtx({ events: [makeMeta(0)] });
    const service = createPlayerService(ctx, mockAssets);
    service.start();
    const newEvent = makeIncremental(100);
    service.send({ type: 'ADD_EVENT', payload: { event: newEvent } });
    expect(service.state.context.events).toHaveLength(2);
    expect(service.state.context.events[1].timestamp).toBe(100);
  });
});

// ── createSpeedService ───────────────────────────────────

describe('createSpeedService', () => {
  function makeSpeedCtx(): SpeedContext {
    return {
      normalSpeed: 1,
      timer: new Timer(undefined, { speed: 1 }),
    };
  }

  it('starts in normal state', () => {
    const service = createSpeedService(makeSpeedCtx());
    service.start();
    expect(service.state.value).toBe('normal');
  });

  it('transitions to skipping on FAST_FORWARD', () => {
    const service = createSpeedService(makeSpeedCtx());
    service.start();
    service.send({ type: 'FAST_FORWARD', payload: { speed: 4 } });
    expect(service.state.value).toBe('skipping');
  });

  it('records normal speed on FAST_FORWARD', () => {
    const ctx = makeSpeedCtx();
    const service = createSpeedService(ctx);
    service.start();
    service.send({ type: 'FAST_FORWARD', payload: { speed: 4 } });
    expect(service.state.context.normalSpeed).toBe(1);
  });

  it('sets timer speed on FAST_FORWARD', () => {
    const ctx = makeSpeedCtx();
    vi.spyOn(ctx.timer, 'setSpeed');
    const service = createSpeedService(ctx);
    service.start();
    service.send({ type: 'FAST_FORWARD', payload: { speed: 4 } });
    expect(ctx.timer.setSpeed).toHaveBeenCalledWith(4);
  });

  it('restores normal speed on BACK_TO_NORMAL', () => {
    const ctx = makeSpeedCtx();
    vi.spyOn(ctx.timer, 'setSpeed');
    const service = createSpeedService(ctx);
    service.start();
    service.send({ type: 'FAST_FORWARD', payload: { speed: 4 } });
    service.send({ type: 'BACK_TO_NORMAL' });
    expect(service.state.value).toBe('normal');
    expect(ctx.timer.setSpeed).toHaveBeenLastCalledWith(1);
  });

  it('handles SET_SPEED in normal state', () => {
    const ctx = makeSpeedCtx();
    vi.spyOn(ctx.timer, 'setSpeed');
    const service = createSpeedService(ctx);
    service.start();
    service.send({ type: 'SET_SPEED', payload: { speed: 2 } });
    expect(service.state.value).toBe('normal');
    expect(ctx.timer.setSpeed).toHaveBeenCalledWith(2);
  });

  it('handles SET_SPEED in skipping state', () => {
    const ctx = makeSpeedCtx();
    vi.spyOn(ctx.timer, 'setSpeed');
    const service = createSpeedService(ctx);
    service.start();
    service.send({ type: 'FAST_FORWARD', payload: { speed: 4 } });
    service.send({ type: 'SET_SPEED', payload: { speed: 8 } });
    expect(service.state.value).toBe('normal');
    expect(ctx.timer.setSpeed).toHaveBeenLastCalledWith(8);
  });
});
