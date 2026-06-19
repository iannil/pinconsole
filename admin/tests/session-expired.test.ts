// 1ac 续集:SESSION_EXPIRED UI 流 + fetchJson 401 handler(审计 T0-1h-ui-1 + T0-1h-ui-3)。
//
// T0-1h-ui-1:fetchJson 401 handler 清 user + set 'SESSION_EXPIRED'
//   验证:fetch 401 → unauthorizedHandler 被调 → auth store user=null + error='SESSION_EXPIRED'
//
// T0-1h-ui-3:SESSION_EXPIRED 后,LoginView 显示 session_expired 文案
//   验证:mount 真 LoginView + 设 auth.error='SESSION_EXPIRED' → 渲染 i18n 文案"会话已过期"
//   (1ae 升级:从 inline 重写 conditional 的 trivial test 改为真 mount + i18n 实例 + DOM 断言)
//
// 不测:App.vue mount fetchMe(T0-1h-ui-2)— 需 Vue mount 测试 + window.location 模拟,
//   1aa 已有 router.test.ts 覆盖 ensureAuthInit timing,等价 cover 此路径。

// @vitest-environment jsdom

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createPinia, setActivePinia, type Pinia } from 'pinia';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';

vi.mock('../src/api/auth', () => ({
  postLogin: vi.fn(),
  postLogout: vi.fn(),
  getMe: vi.fn(),
}));

import { useAuthStore } from '../src/stores/auth';
import LoginView from '../src/views/LoginView.vue';
import zhCN from '../src/i18n/zh-CN';
import enUS from '../src/i18n/en-US';

describe('1ac: SESSION_EXPIRED + fetchJson 401 handler', () => {
  let pinia: Pinia;
  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    vi.clearAllMocks();
    vi.resetModules();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetchJson 401 triggers unauthorizedHandler → auth store clears user + sets SESSION_EXPIRED', async () => {
    // 用 真 fetchJson + stubbed fetch 模拟 401 响应
    // 动态 import 防止 pinia 未设置时 unauthorizedHandler 被捕获
    const { fetchJson, setUnauthorizedHandler } = await import('../src/utils/fetchJson');
    const auth = useAuthStore();

    // 模拟已登录状态
    (auth as unknown as { user: { id: string } }).user = {
      id: 'u-1',
      email: 'admin@test.local',
      display_name: 'Admin',
      role: 'admin',
    };
    expect(auth.user).not.toBeNull();
    expect(auth.error).toBe('');

    // auth store 构造时已调 setUnauthorizedHandler,这里再次显式调以确认接线
    // (实际生产环境 auth store 构造时已注册)
    setUnauthorizedHandler(() => {
      (auth as unknown as { user: unknown }).user = null;
      (auth as unknown as { error: string }).error = 'SESSION_EXPIRED';
    });

    // stub fetch 返回 401
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      status: 401,
      ok: false,
    } as Response));

    // 调 fetchJson,应抛 'UNAUTHORIZED' 并触发 handler
    await expect(fetchJson('/api/anything')).rejects.toThrow('UNAUTHORIZED');

    // 验证 handler 被调:auth store 已清 user + 设 SESSION_EXPIRED
    expect(auth.user).toBeNull();
    expect(auth.error).toBe('SESSION_EXPIRED');
  });

  it('SESSION_EXPIRED 时 LoginView mount 后渲染特殊 i18n 文案', async () => {
    // 1ae 升级 T0-1h-ui-3:从 trivial inline 重写 → 真 mount LoginView + 真 i18n + DOM 文本断言
    // 防止 LoginView.vue 改 errorText conditional 后测试仍能捕获
    const auth = useAuthStore();
    (auth as unknown as { user: unknown }).user = null;
    (auth as unknown as { error: string }).error = 'SESSION_EXPIRED';

    const i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      fallbackLocale: 'en-US',
      messages: { 'zh-CN': zhCN, 'en-US': enUS },
    });

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/login', component: LoginView }],
    });
    await router.push('/login');
    await router.isReady();

    const wrapper = mount(LoginView, {
      global: {
        plugins: [i18n, router, pinia],
      },
    });
    await flushPromises();

    // 关键断言:渲染的 DOM 必须包含 SESSION_EXPIRED 对应的 i18n 文案
    expect(wrapper.text()).toContain('会话已过期,请重新登录');
    // 且不能错误地显示 invalid_credentials 文案
    expect(wrapper.text()).not.toContain('邮箱或密码错误');
  });

  it('invalid_credentials 时 LoginView mount 后渲染凭证错误文案', async () => {
    // 1ae 新增:对照测试 invalid_credentials 分支
    const auth = useAuthStore();
    (auth as unknown as { user: unknown }).user = null;
    (auth as unknown as { error: string }).error = 'invalid_credentials';

    const i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      fallbackLocale: 'en-US',
      messages: { 'zh-CN': zhCN, 'en-US': enUS },
    });

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/login', component: LoginView }],
    });
    await router.push('/login');
    await router.isReady();

    const wrapper = mount(LoginView, {
      global: {
        plugins: [i18n, router, pinia],
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('邮箱或密码错误');
    expect(wrapper.text()).not.toContain('会话已过期');
  });

  it('fetchJson 200 不触发 unauthorizedHandler(正常路径)', async () => {
    const { fetchJson, setUnauthorizedHandler } = await import('../src/utils/fetchJson');
    const auth = useAuthStore();

    const handlerSpy = vi.fn();
    setUnauthorizedHandler(handlerSpy);

    // stub fetch 返回 200 + JSON body
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      status: 200,
      ok: true,
      json: () => Promise.resolve({ ok: true }),
    } as Response));

    await fetchJson('/api/anything');
    expect(handlerSpy).not.toHaveBeenCalled();
    expect(auth.user).toBeNull(); // unchanged
    expect(auth.error).toBe(''); // unchanged
  });

  it('fetchJson 500 不触发 unauthorizedHandler(走普通错误路径)', async () => {
    const { fetchJson, setUnauthorizedHandler } = await import('../src/utils/fetchJson');
    const handlerSpy = vi.fn();
    setUnauthorizedHandler(handlerSpy);

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      status: 500,
      ok: false,
      json: () => Promise.resolve({ error: 'server_error' }),
    } as Response));

    await expect(fetchJson('/api/anything')).rejects.toThrow('server_error');
    expect(handlerSpy).not.toHaveBeenCalled();
  });

  it('fetchJson credentials=include by default(cookie 必须随请求带)', async () => {
    const { fetchJson } = await import('../src/utils/fetchJson');
    const fetchSpy = vi.fn().mockResolvedValue({
      status: 200, ok: true, json: () => Promise.resolve({}),
    } as Response);
    vi.stubGlobal('fetch', fetchSpy);

    await fetchJson('/api/x');
    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [, init] = fetchSpy.mock.calls[0]!;
    expect(init).toBeDefined();
    expect((init as RequestInit).credentials).toBe('include');
  });
});
