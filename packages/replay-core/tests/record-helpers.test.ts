/**
 * @vitest-environment jsdom
 *
 * record 模块测试：error-handler + processed-node-manager + stylesheet-manager。
 * 这些是 record/ 中可独立测试的组件，不依赖真实 MutationObserver 运行。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  registerErrorHandler,
  unregisterErrorHandler,
  callbackWrapper,
} from '../src/record/error-handler';
import ProcessedNodeManager from '../src/record/processed-node-manager';
import { StylesheetManager } from '../src/record/stylesheet-manager';

// ── Helpers ──────────────────────────────────────────────

function mockBuffer(): any {
  return { foo: 'buffer_' + Math.random() };
}

// ── error-handler ────────────────────────────────────────

describe('error-handler', () => {
  afterEach(() => {
    unregisterErrorHandler();
  });

  describe('registerErrorHandler / unregisterErrorHandler', () => {
    it('registers and unregisters without throwing', () => {
      const handler = vi.fn();
      registerErrorHandler(handler);
      unregisterErrorHandler();
    });

    it('allows undefined handler', () => {
      registerErrorHandler(undefined);
      unregisterErrorHandler();
    });
  });

  describe('callbackWrapper', () => {
    it('returns same callback when no handler registered', () => {
      const cb = () => 42;
      const wrapped = callbackWrapper(cb);
      expect(wrapped).toBe(cb);
    });

    it('wraps callback with try-catch when handler registered', () => {
      const cb = () => 42;
      registerErrorHandler(vi.fn());
      const wrapped = callbackWrapper(cb);
      expect(wrapped).not.toBe(cb);
      expect(wrapped()).toBe(42);
    });

    it('passes error to handler and returns undefined when handler returns true', () => {
      const handler = vi.fn().mockReturnValue(true);
      registerErrorHandler(handler);
      const cb = () => { throw new Error('test error'); };
      const wrapped = callbackWrapper(cb);
      expect(wrapped()).toBeUndefined();
      expect(handler).toHaveBeenCalledWith(new Error('test error'));
    });

    it('re-throws error when handler returns non-true', () => {
      const handler = vi.fn().mockReturnValue(false);
      registerErrorHandler(handler);
      const cb = () => { throw new Error('test error'); };
      const wrapped = callbackWrapper(cb);
      expect(() => wrapped()).toThrow('test error');
    });

    it('passes arguments through', () => {
      registerErrorHandler(vi.fn());
      const cb = vi.fn();
      const wrapped = callbackWrapper(cb);
      wrapped(1, 2, 3);
      expect(cb).toHaveBeenCalledWith(1, 2, 3);
    });
  });
});

// ── ProcessedNodeManager ─────────────────────────────────

describe('ProcessedNodeManager', () => {
  let manager: ProcessedNodeManager;

  beforeEach(() => {
    manager = new ProcessedNodeManager();
  });

  it('inOtherBuffer returns undefined (falsy) for untracked node', () => {
    const node = document.createElement('div');
    // Returns undefined (falsy), not false, because WeakMap.get returns undefined
    // and the && short-circuit propagates it
    expect(manager.inOtherBuffer(node, mockBuffer())).toBeUndefined();
  });

  it('inOtherBuffer returns false for node in same buffer', () => {
    const node = document.createElement('div');
    const buf = mockBuffer();
    manager.add(node, buf);
    expect(manager.inOtherBuffer(node, buf)).toBe(false);
  });

  it('inOtherBuffer returns true for node in different buffer', () => {
    const node = document.createElement('div');
    manager.add(node, mockBuffer());
    expect(manager.inOtherBuffer(node, mockBuffer())).toBe(true);
  });

  it('destroy does not throw', () => {
    expect(() => manager.destroy()).not.toThrow();
  });
});

// ── StylesheetManager ────────────────────────────────────

describe('StylesheetManager', () => {
  let manager: StylesheetManager;
  let mutationCb: any;
  let adoptedCb: any;

  beforeEach(() => {
    mutationCb = vi.fn();
    adoptedCb = vi.fn();
    manager = new StylesheetManager({ mutationCb, adoptedStyleSheetCb: adoptedCb });
  });

  it('styleMirror is defined', () => {
    expect(manager.styleMirror).toBeDefined();
  });

  describe('reset', () => {
    it('does not throw', () => {
      expect(() => manager.reset()).not.toThrow();
    });
  });

  describe('attachLinkElement', () => {
    it('handles linkEl with _cssText attribute', () => {
      const link = document.createElement('link');
      const childSn = { id: 1, tagName: 'link', attributes: { _cssText: 'body { color: red; }' } } as any;
      manager.attachLinkElement(link, childSn);
      expect(mutationCb).toHaveBeenCalled();
    });

    it('handles linkEl without _cssText attribute', () => {
      const link = document.createElement('link');
      const childSn = { id: 1, tagName: 'link', attributes: {} } as any;
      manager.attachLinkElement(link, childSn);
      expect(mutationCb).not.toHaveBeenCalled();
    });
  });

  describe('trackLinkElement', () => {
    it('accepts link element', () => {
      const link = document.createElement('link');
      expect(() => manager.trackLinkElement(link)).not.toThrow();
    });

    it('handles repeated tracks', () => {
      const link = document.createElement('link');
      manager.trackLinkElement(link);
      manager.trackLinkElement(link); // no throw
    });
  });

  describe('adoptStyleSheets', () => {
    it('handles empty sheets array', () => {
      manager.adoptStyleSheets([], 1);
      expect(adoptedCb).not.toHaveBeenCalled();
    });
  });
});
