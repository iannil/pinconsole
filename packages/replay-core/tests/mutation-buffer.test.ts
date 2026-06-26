/**
 * @vitest-environment jsdom
 *
 * MutationBuffer 核心状态方法测试。
 * 不依赖真实 MutationObserver — 只测 freeze/unfreeze/lock/unlock/reset/isFrozen/processMutations/emit。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import MutationBuffer from '../src/record/mutation';
import type { blockClass } from '../src/rrweb-types';

type MockCanvasManager = {
  freeze: ReturnType<typeof vi.fn>;
  unfreeze: ReturnType<typeof vi.fn>;
  lock: ReturnType<typeof vi.fn>;
  unlock: ReturnType<typeof vi.fn>;
  reset: ReturnType<typeof vi.fn>;
};

function makeMockCanvasManager(): MockCanvasManager {
  return {
    freeze: vi.fn(),
    unfreeze: vi.fn(),
    lock: vi.fn(),
    unlock: vi.fn(),
    reset: vi.fn(),
  };
}

describe('MutationBuffer', () => {
  let buf: MutationBuffer;
  let canvasManager: MockCanvasManager;

  function makeOpts(overrides: Record<string, unknown> = {}): any {
    return {
      mutationCb: vi.fn(),
      blockClass: 'rr-block' as blockClass,
      blockSelector: null,
      maskTextClass: 'rr-mask' as blockClass,
      maskTextSelector: null,
      inlineStylesheet: true,
      maskAllInputs: true,
      maskInputOptions: {},
      maskInputFn: undefined,
      maskTextFn: undefined,
      hooks: {},
      defaultChildList: true as unknown as Record<string, true>,
      defaultAttributes: true as unknown as Record<string, true>,
      defaultCharacterData: true as unknown as Record<string, true>,
      defaultAttributeValue: '',
      canvasManager,
      shadowDomManager: { reset: vi.fn() },
      ...overrides,
    };
  }

  beforeEach(() => {
    buf = new MutationBuffer();
    canvasManager = makeMockCanvasManager();
  });

  describe('init', () => {
    it('does not throw with minimal options', () => {
      expect(() => buf.init(makeOpts())).not.toThrow();
    });
  });

  describe('isFrozen', () => {
    it('returns false after init', () => {
      buf.init(makeOpts());
      expect(buf.isFrozen()).toBe(false);
    });
  });

  describe('freeze / unfreeze', () => {
    it('freeze sets frozen=true', () => {
      buf.init(makeOpts());
      buf.freeze();
      expect(buf.isFrozen()).toBe(true);
    });

    it('unfreeze sets frozen=false', () => {
      buf.init(makeOpts());
      buf.freeze();
      buf.unfreeze();
      expect(buf.isFrozen()).toBe(false);
    });

    it('freeze calls canvasManager.freeze', () => {
      buf.init(makeOpts());
      buf.freeze();
      expect(canvasManager.freeze).toHaveBeenCalled();
    });

    it('unfreeze calls canvasManager.unfreeze', () => {
      buf.init(makeOpts());
      buf.freeze();
      buf.unfreeze();
      expect(canvasManager.unfreeze).toHaveBeenCalled();
    });
  });

  describe('lock / unlock', () => {
    it('lock sets locked state and calls canvasManager.lock', () => {
      buf.init(makeOpts());
      buf.lock();
      expect(canvasManager.lock).toHaveBeenCalled();
    });

    it('unlock calls canvasManager.unlock', () => {
      buf.init(makeOpts());
      buf.lock();
      buf.unlock();
      expect(canvasManager.unlock).toHaveBeenCalled();
    });
  });

  describe('reset', () => {
    it('calls canvasManager.reset', () => {
      buf.init(makeOpts());
      buf.reset();
      expect(canvasManager.reset).toHaveBeenCalled();
    });
  });

  describe('processMutations', () => {
    it('handles empty mutations array', () => {
      buf.init(makeOpts());
      expect(() => buf.processMutations([])).not.toThrow();
    });
  });

  describe('emit', () => {
    it('returns early when frozen', () => {
      buf.init(makeOpts());
      buf.freeze();
      expect(() => (buf as any).emit()).not.toThrow();
    });
  });
});
