// 切片 1aa:session 单元测试
// 覆盖 getOrCreateVisitorId(持久化)+ initSession(网络 + 持久化)+ getCachedSessionId + clearCachedSessionId。

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

describe('session', () => {
  let store: Record<string, string>;
  let originalLocalStorage: Storage;
  let originalCrypto: Crypto;

  beforeEach(() => {
    store = {};
    originalLocalStorage = globalThis.localStorage;
    originalCrypto = globalThis.crypto;

    // localStorage mock
    const lsMock: Storage = {
      getItem: vi.fn((k: string) => store[k] ?? null),
      setItem: vi.fn((k: string, v: string) => {
        store[k] = v;
      }),
      removeItem: vi.fn((k: string) => {
        delete store[k];
      }),
      clear: vi.fn(() => {
        Object.keys(store).forEach((k) => delete store[k]);
      }),
      key: vi.fn((i: number) => Object.keys(store)[i] ?? null),
      get length() {
        return Object.keys(store).length;
      },
    };
    vi.stubGlobal('localStorage', lsMock);

    // crypto.randomUUID mock(稳定值便于断言)
    const cryptoMock = {
      randomUUID: vi.fn(() => 'mock-uuid-1234'),
    };
    vi.stubGlobal('crypto', cryptoMock);

    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  describe('getOrCreateVisitorId', () => {
    it('generates and persists visitor_id on first call', async () => {
      const mod = await import('../src/session');
      const id = mod.getOrCreateVisitorId();

      expect(id).toBe('mock-uuid-1234');
      expect(localStorage.getItem('mm:visitor_id')).toBe('mock-uuid-1234');
    });

    it('returns existing visitor_id on subsequent calls', async () => {
      store['mm:visitor_id'] = 'existing-id';
      const mod = await import('../src/session');

      const id = mod.getOrCreateVisitorId();
      expect(id).toBe('existing-id');
    });
  });

  describe('initSession', () => {
    it('POSTs and persists session_id on success', async () => {
      const mod = await import('../src/session');
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response(
          JSON.stringify({
            visitor_id: 'v1',
            session_id: 's1',
            tenant_id: 't1',
          }),
          { status: 200 },
        ),
      );

      const info = await mod.initSession('v1', 'http://api', 'UA');

      expect(info).toEqual({
        visitorId: 'v1',
        sessionId: 's1',
        tenantId: 't1',
      });
      expect(localStorage.getItem('mm:session_id')).toBe('s1');

      const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock
        .calls[0] as [string, RequestInit];
      expect(url).toBe('http://api/api/session/init');
      expect(init.method).toBe('POST');
      expect(init.credentials).toBe('include');
      expect(JSON.parse(init.body as string)).toEqual({
        visitor_id: 'v1',
        ua: 'UA',
      });
    });

    it('throws on non-2xx response', async () => {
      const mod = await import('../src/session');
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response('error', { status: 500 }),
      );

      await expect(mod.initSession('v1', 'http://api', 'UA')).rejects.toThrow(
        /session init failed: HTTP 500/,
      );
    });

    it('does not persist session_id when server returns empty', async () => {
      const mod = await import('../src/session');
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response(
          JSON.stringify({
            visitor_id: 'v1',
            session_id: '',
            tenant_id: 't1',
          }),
          { status: 200 },
        ),
      );

      await mod.initSession('v1', 'http://api', 'UA');

      expect(localStorage.getItem('mm:session_id')).toBeNull();
    });
  });

  describe('getCachedSessionId', () => {
    it('returns null when no cached session', async () => {
      const mod = await import('../src/session');
      expect(mod.getCachedSessionId()).toBeNull();
    });

    it('returns cached session_id', async () => {
      store['mm:session_id'] = 'cached-s';
      const mod = await import('../src/session');
      expect(mod.getCachedSessionId()).toBe('cached-s');
    });
  });

  describe('clearCachedSessionId', () => {
    it('removes cached session_id', async () => {
      store['mm:session_id'] = 'cached-s';
      const mod = await import('../src/session');
      mod.clearCachedSessionId();
      expect(store['mm:session_id']).toBeUndefined();
    });
  });
});
