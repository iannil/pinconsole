/**
 * @vitest-environment jsdom
 *
 * Replayer 核心方法测试 — 测试不依赖 iframe 渲染的公共 API。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Replayer } from '../src/replay/index';
import type { eventWithTime } from '../src/types';
import { EventType, IncrementalSource } from '../src/types';

function makeMeta(ts: number): eventWithTime {
  return { type: EventType.Meta, timestamp: ts, data: { width: 1024, height: 768, href: 'http://localhost' } } as eventWithTime;
}

function makeLiveEvents(): eventWithTime[] {
  return [
    { type: EventType.Meta, timestamp: 0, data: { width: 1024, height: 768, href: 'http://localhost' } } as eventWithTime,
    { type: EventType.IncrementalSnapshot, timestamp: 100, data: { source: IncrementalSource.Mutation } } as eventWithTime,
  ];
}

describe('Replayer', () => {
  let replayer: Replayer;

  beforeEach(() => {
    // Use liveMode + no fullSnapshot so constructor doesn't schedule rebuild
    replayer = new Replayer(makeLiveEvents(), { liveMode: true });
  });

  describe('constructor', () => {
    it('throws with < 2 events', () => {
      expect(() => new Replayer([])).toThrow('at least 2 events');
    });

    it('throws with single event in non-live mode', () => {
      expect(() => new Replayer([makeMeta(0)])).toThrow('at least 2 events');
    });

    it('creates Replayer with >= 2 events', () => {
      const r = new Replayer(makeLiveEvents());
      expect(r).toBeInstanceOf(Replayer);
      expect(r.config.speed).toBe(1);
      expect(r.wrapper).toBeDefined();
      expect(r.iframe).toBeDefined();
      r.destroy();
    });

    it('works with liveMode even with < 2 events', () => {
      const r = new Replayer([makeMeta(0)], { liveMode: true });
      expect(r).toBeInstanceOf(Replayer);
      r.destroy();
    });

    it('merges user config with defaults', () => {
      const r = new Replayer(makeLiveEvents(), { speed: 4, showDebug: true, liveMode: true });
      expect(r.config.speed).toBe(4);
      expect(r.config.showDebug).toBe(true);
      expect(r.config.maxSpeed).toBe(360); // default
      r.destroy();
    });
  });

  describe('setConfig', () => {
    it('updates config values', () => {
      replayer.setConfig({ speed: 2 });
      expect(replayer.config.speed).toBe(2);
    });

    it('does not affect unset config values', () => {
      replayer.setConfig({ showWarning: false });
      expect(replayer.config.speed).toBe(1); // unchanged
    });
  });

  describe('getMetaData', () => {
    it('returns metadata object with expected properties', () => {
      const meta = replayer.getMetaData();
      expect(meta).toHaveProperty('startTime');
      expect(meta).toHaveProperty('endTime');
      expect(meta).toHaveProperty('totalTime');
    });
  });

  describe('getCurrentTime', () => {
    it('returns 0 before play', () => {
      expect(replayer.getCurrentTime()).toBe(0);
    });
  });

  describe('getTimeOffset', () => {
    it('returns 0 by default', () => {
      expect(replayer.getTimeOffset()).toBe(0);
    });
  });

  describe('getMirror', () => {
    it('returns Mirror instance', () => {
      const mirror = replayer.getMirror();
      expect(mirror).toBeDefined();
    });
  });

  describe('enableInteract / disableInteract', () => {
    it('disableInteract does not throw', () => {
      expect(() => replayer.disableInteract()).not.toThrow();
    });

    it('enableInteract does not throw', () => {
      expect(() => replayer.enableInteract()).not.toThrow();
    });

    it('enableInteract after disableInteract does not throw', () => {
      replayer.disableInteract();
      expect(() => replayer.enableInteract()).not.toThrow();
    });
  });

  describe('resetCache', () => {
    it('does not throw', () => {
      expect(() => replayer.resetCache()).not.toThrow();
    });
  });

  describe('on / off', () => {
    it('on registers event handler', () => {
      const handler = vi.fn();
      replayer.on('test-event', handler);
      // Emit via the replayer's internal emitter
      (replayer as any).emitter.emit('test-event', 'arg');
      expect(handler).toHaveBeenCalledWith('arg');
    });

    it('off removes event handler', () => {
      const handler = vi.fn();
      replayer.on('test-event', handler);
      replayer.off('test-event', handler);
      (replayer as any).emitter.emit('test-event');
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('destroy', () => {
    it('does not throw on first call', () => {
      expect(() => replayer.destroy()).not.toThrow();
    });
  });
});
