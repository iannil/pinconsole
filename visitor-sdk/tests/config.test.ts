// 切片 1aa:config 单元测试
// 覆盖 dropUndefined + resolveConfig + readScriptData + readWindowConfig + parseBool。
// 注:config 依赖 document/window,SDK 默认 node env,需 stubGlobal mock。

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { resolveConfig } from '../src/config';

// dropUndefined 是 private helper,通过 resolveConfig 行为间接验证。
// 关键 case:显式 undefined 不能覆盖 DEFAULTS。

interface MockScriptLike {
  getAttribute: (k: string) => string | null;
}

function setupDomStub(opts: {
  scripts?: MockScriptLike[];
  windowConfig?: Record<string, unknown>;
}) {
  const docMock = {
    querySelectorAll: vi.fn(() => opts.scripts ?? []),
  };
  const winMock: Record<string, unknown> = {};
  if (opts.windowConfig !== undefined) {
    winMock.MM_CONFIG = opts.windowConfig;
  }
  vi.stubGlobal('document', docMock);
  vi.stubGlobal('window', winMock);
  return { docMock, winMock };
}

describe('config — resolveConfig', () => {
  beforeEach(() => {
    // 每个测试自己 setupDomStub
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  describe('defaults', () => {
    it('returns safe defaults when no script and no window config', () => {
      setupDomStub({});
      const cfg = resolveConfig();
      expect(cfg.enableRecording).toBe(false);
      expect(cfg.debug).toBe(false);
      expect(cfg.consentMode).toBe('opt-in');
      expect(cfg.showCoBrowseBanner).toBe(true);
    });
  });

  describe('readScriptData', () => {
    it('reads data-tenant-id and data-page-id', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) => {
              if (k === 'data-tenant-id') return 't1';
              if (k === 'data-page-id') return 'p1';
              return null;
            },
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.tenantId).toBe('t1');
      expect(cfg.pageId).toBe('p1');
    });

    it('reads data-endpoint', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) =>
              k === 'data-endpoint' ? 'wss://custom/ws' : null,
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.endpoint).toBe('wss://custom/ws');
    });

    it('parses data-enable-recording and data-debug as bool', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) => {
              if (k === 'data-enable-recording') return 'true';
              if (k === 'data-debug') return '1';
              return null;
            },
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.enableRecording).toBe(true);
      expect(cfg.debug).toBe(true);
    });

    it('treats missing data-* as undefined (does not override defaults)', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: () => null,
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.enableRecording).toBe(false);
      expect(cfg.debug).toBe(false);
    });

    it('uses the LAST script element when multiple match selector', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) =>
              k === 'data-tenant-id' ? 'first' : null,
          },
          {
            getAttribute: (k: string) =>
              k === 'data-tenant-id' ? 'second' : null,
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.tenantId).toBe('second');
    });

    it('validates data-consent-mode against enum', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) => {
              if (k === 'data-consent-mode') return 'opt-out';
              return null;
            },
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.consentMode).toBe('opt-out');
    });

    it('rejects invalid data-consent-mode (falls back to default)', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) => {
              if (k === 'data-consent-mode') return 'invalid-mode';
              return null;
            },
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.consentMode).toBe('opt-in');
    });
  });

  describe('readWindowConfig (window.MM_CONFIG)', () => {
    it('window config overrides script config', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) =>
              k === 'data-tenant-id' ? 'from-script' : null,
          },
        ],
        windowConfig: { tenantId: 'from-window' },
      });
      const cfg = resolveConfig();
      expect(cfg.tenantId).toBe('from-window');
    });

    it('window config undefined fields do not override defaults', () => {
      setupDomStub({
        windowConfig: { enableRecording: undefined },
      });
      const cfg = resolveConfig();
      // 显式 undefined 不应覆盖默认 false
      expect(cfg.enableRecording).toBe(false);
    });

    it('window config explicit values override defaults', () => {
      setupDomStub({
        windowConfig: { debug: true, showCoBrowseBanner: false },
      });
      const cfg = resolveConfig();
      expect(cfg.debug).toBe(true);
      expect(cfg.showCoBrowseBanner).toBe(false);
    });
  });

  describe('dropUndefined behavior (Partial<T> safety)', () => {
    it('regression: Partial<T> with explicit undefined must not override DEFAULTS', () => {
      // 这是 v1-followups 修复的核心 case:
      // mm.init({ apiBase: undefined }) 应保持默认 endpoint 推断,而非 undefined
      setupDomStub({
        scripts: [
          {
            getAttribute: () => null,
          },
        ],
        windowConfig: {
          consentMode: undefined,
          enableRecording: undefined,
          debug: undefined,
        },
      });
      const cfg = resolveConfig();
      expect(cfg.consentMode).toBe('opt-in');
      expect(cfg.enableRecording).toBe(false);
      expect(cfg.debug).toBe(false);
    });
  });

  describe('edge cases', () => {
    it('no script elements returns defaults', () => {
      setupDomStub({ scripts: [] });
      const cfg = resolveConfig();
      expect(cfg.tenantId).toBeUndefined();
      expect(cfg.pageId).toBeUndefined();
    });

    it('parseBool returns false for non-true values', () => {
      setupDomStub({
        scripts: [
          {
            getAttribute: (k: string) => {
              if (k === 'data-debug') return 'false';
              if (k === 'data-enable-recording') return '0';
              return null;
            },
          },
        ],
      });
      const cfg = resolveConfig();
      expect(cfg.debug).toBe(false);
      expect(cfg.enableRecording).toBe(false);
    });
  });
});
