// 切片 1aa:api/auth 单元测试
// 覆盖 postLogin / postLogout / getMe + 401 特殊处理 + trace_id 自动注入。

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { postLogin, postLogout, getMe } from '../src/api/auth';

describe('api/auth', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  describe('postLogin', () => {
    it('posts credentials and returns UserInfo on 2xx', async () => {
      const mockUser = {
        id: 'u1',
        email: 'a@b.c',
        display_name: 'A',
        role: 'admin',
      };
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response(JSON.stringify(mockUser), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      );

      const result = await postLogin({ email: 'a@b.c', password: 'p' });

      expect(result).toEqual(mockUser);
      const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock
        .calls[0] as [string, RequestInit];
      expect(url).toBe('/api/auth/login');
      expect(init.method).toBe('POST');
      expect(init.credentials).toBe('include');
      const headers = init.headers as Headers;
      expect(headers.get('Content-Type')).toBe('application/json');
      expect(headers.get('X-Trace-Id')).toMatch(/^[0-9a-f]{32}$/);
      expect(JSON.parse(init.body as string)).toEqual({
        email: 'a@b.c',
        password: 'p',
      });
    });

    it('throws on non-2xx with trace_id in error', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response('{"error":"invalid_credentials"}', {
          status: 401,
          headers: {
            'Content-Type': 'application/json',
            'X-Trace-Id': 'trace-abc',
          },
        }),
      );

      await expect(
        postLogin({ email: 'x', password: 'y' }),
      ).rejects.toThrow(/trace_id=trace-abc/);
    });
  });

  describe('postLogout', () => {
    it('posts logout and resolves void on any response (no throw)', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response('', { status: 500 }),
      );

      await expect(postLogout()).resolves.toBeUndefined();

      const [url, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock
        .calls[0] as [string, RequestInit];
      expect(url).toBe('/api/auth/logout');
      expect(init.method).toBe('POST');
    });

    it('does not inject body (no Content-Type header)', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response(null, { status: 204 }),
      );

      await postLogout();

      const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock
        .calls[0] as [string, RequestInit];
      // postLogout 不传 body,apiFetch 也不会加 Content-Type
      const headers = init.headers as Headers;
      expect(headers.get('Content-Type')).toBeNull();
    });
  });

  describe('getMe', () => {
    it('returns null on 401 (not authenticated is a valid state)', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response('{"error":"not_authenticated"}', { status: 401 }),
      );

      const result = await getMe();
      expect(result).toBeNull();
    });

    it('returns parsed user on 2xx', async () => {
      const mockUser = {
        id: 'u1',
        email: 'a@b.c',
        display_name: 'A',
        role: 'admin',
      };
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response(JSON.stringify(mockUser), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      );

      const result = await getMe();
      expect(result).toEqual(mockUser);
    });

    it('throws on non-401 non-2xx status', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response('server error', { status: 500 }),
      );

      await expect(getMe()).rejects.toThrow(/HTTP 500/);
    });

    it('injects X-Trace-Id header via apiFetch', async () => {
      (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
        new Response('{}', { status: 200 }),
      );

      await getMe();

      const [, init] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock
        .calls[0] as [string, RequestInit];
      const headers = init.headers as Headers;
      expect(headers.get('X-Trace-Id')).toMatch(/^[0-9a-f]{32}$/);
    });
  });
});
