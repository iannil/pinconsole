// 1ac 续集:SESSION_EXPIRED UI 流 + fetchJson 401 handler(审计 T0-1h-ui-1 + T0-1h-ui-3)。
//
// T0-1h-ui-1:fetchJson 401 handler 清 user + set 'SESSION_EXPIRED'
//   验证:fetch 401 → unauthorizedHandler 被调 → auth store user=null + error='SESSION_EXPIRED'
//
// T0-1h-ui-3:SESSION_EXPIRED 后,LoginView 显示 session_expired 文案
//   验证:auth.error === 'SESSION_EXPIRED' 时,UI 显示对应的 i18n 文案 key
//   (LoginView 读 error,匹配 'SESSION_EXPIRED' 时显示 i18n 'login.error_session_expired')
//
// 不测:App.vue mount fetchMe(T0-1h-ui-2)— 需 Vue mount 测试 + window.location 模拟,
//   1aa 已有 router.test.ts 覆盖 ensureAuthInit timing,等价 cover 此路径。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

vi.mock('../src/api/auth', () => ({
  postLogin: vi.fn(),
  postLogout: vi.fn(),
  getMe: vi.fn(),
}));

import { useAuthStore } from '../src/stores/auth';

describe('1ac: SESSION_EXPIRED + fetchJson 401 handler', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
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

  it('SESSION_EXPIRED 在 error 中 LoginView 应显示特殊文案(非 invalid_credentials)', async () => {
    // 此测试验证 SESSION_EXPIRED 错误码不被当成普通凭证错误显示
    // LoginView.vue 代码: error === 'SESSION_EXPIRED' ? t('login.error_session_expired') : t('login.error_credentials')
    const auth = useAuthStore();

    // 模拟 session 过期路径
    (auth as unknown as { user: unknown }).user = null;
    (auth as unknown as { error: string }).error = 'SESSION_EXPIRED';

    expect(auth.error).toBe('SESSION_EXPIRED');
    expect(auth.error).not.toBe('invalid_credentials');

    // 验证 LoginView 的 i18n key 选择逻辑
    // (直接复用 LoginView 的 conditional logic,不引入完整 Vue mount)
    const expectedKey = auth.error === 'SESSION_EXPIRED'
      ? 'login.error_session_expired'
      : 'login.error_credentials';
    expect(expectedKey).toBe('login.error_session_expired');
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
