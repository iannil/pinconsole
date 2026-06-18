// 切片 1aa:vue-router beforeEach 守卫测试
// 覆盖 ensureAuthInit lazy 触发 + 三种 redirect 分支(已认证→dashboard / 未认证→login / public 直通)。
//
// 注意:router 是单例,内部 ensureAuthInit 缓存了 fetchMe promise。
// 用 vi.resetModules + 动态 import 隔离每个 test 的 module state。
// 需 jsdom 环境(createWebHistory 内部依赖 window.history)。

// @vitest-environment jsdom

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

describe('router guards', () => {
  let router: typeof import('../src/router')['router'];
  let authStore: typeof import('../src/stores/auth')['useAuthStore'];
  let getMeMock: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    vi.resetModules();
    setActivePinia(createPinia());

    getMeMock = vi.fn();
    vi.doMock('../src/api/auth', () => ({
      postLogin: vi.fn(),
      postLogout: vi.fn(),
      getMe: getMeMock,
    }));

    const routerModule = await import('../src/router');
    const authModule = await import('../src/stores/auth');
    router = routerModule.router;
    authStore = authModule.useAuthStore;
  });

  afterEach(() => {
    vi.doUnmock('../src/api/auth');
    vi.clearAllMocks();
  });

  it('redirects unauthenticated user from protected route to /login with redirect query', async () => {
    getMeMock.mockResolvedValue(null); // fetchMe → user null

    await router.push('/dashboard');
    await router.isReady();

    expect(router.currentRoute.value.name).toBe('login');
    expect(router.currentRoute.value.query.redirect).toBe('/dashboard');
  });

  it('lets unauthenticated user through public /login route', async () => {
    getMeMock.mockResolvedValue(null);

    await router.push('/login');
    await router.isReady();

    expect(router.currentRoute.value.name).toBe('login');
  });

  it('redirects authenticated user from /login to /dashboard', async () => {
    getMeMock.mockResolvedValue({
      id: 'u1',
      email: 'a@b.c',
      display_name: 'A',
      role: 'admin',
    });

    await router.push('/login');
    await router.isReady();

    expect(router.currentRoute.value.name).toBe('dashboard');
  });

  it('lets authenticated user through protected route', async () => {
    getMeMock.mockResolvedValue({
      id: 'u1',
      email: 'a@b.c',
      display_name: 'A',
      role: 'admin',
    });

    await router.push('/dashboard');
    await router.isReady();

    expect(router.currentRoute.value.name).toBe('dashboard');
  });

  it('caches fetchMe promise across navigations (single network call)', async () => {
    getMeMock.mockResolvedValue({
      id: 'u1',
      email: 'a@b.c',
      display_name: 'A',
      role: 'admin',
    });

    await router.push('/dashboard');
    await router.isReady();
    await router.push('/replay');
    await router.isReady();
    await router.push('/privacy');
    await router.isReady();

    // ensureAuthInit 应只触发一次 fetchMe
    expect(getMeMock).toHaveBeenCalledTimes(1);
  });

  it('does not break router when fetchMe rejects (resilience for networks errors)', async () => {
    getMeMock.mockRejectedValue(new Error('NETWORK_DOWN'));

    await router.push('/dashboard');
    await router.isReady();

    // fetchMe 内部 catch 兜底:即使网络挂,router 也能 resolve
    // user 仍为 null → 跳 login
    expect(router.currentRoute.value.name).toBe('login');
  });
});
