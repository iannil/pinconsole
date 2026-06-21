// CV-2 切片补测:admin 包覆盖率从 85.67% → ≥90%。
//
// 补测目标:
// 1. claim.ts (4.76%) — claimSession/releaseSession/getClaimState 全测
// 2. privacy.ts (12.5%) — deleteVisitorByFingerprint
// 3. fetchJson.ts (96.55%) — 非 ok 时 catch 分支(line 39)
// 4. client.ts (94.59%) — statusText fallback (line 81-82)
// 5. App.vue — mount 一次
// 6. FloatingInput.vue — fill/cancel/Enter/Escape keydown
// 7. ReplayViewer.vue — loadInitial error + events.length=0
// 8. LoginView.vue — errorText 三种 + onSubmit + defaultHint
// 9. i18n/index.ts — 导入即触发 createI18n
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';

// === claim.ts 测试 ===

vi.mock('../src/api/client', () => ({
  apiJson: vi.fn(),
}));

import { apiJson } from '../src/api/client';
import {
  claimSession,
  releaseSession,
  getClaimState,
} from '../src/api/claim';
import { deleteVisitorByFingerprint } from '../src/api/privacy';
import { fetchJson, setUnauthorizedHandler } from '../src/utils/fetchJson';

describe('claim.ts', () => {
  beforeEach(() => vi.clearAllMocks());

  it('claimSession POST /api/sessions/:id/claim', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { claimed: true, claimed_by: 'op-1' },
    });
    const got = await claimSession('s1');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s1/claim',
      expect.objectContaining({ method: 'POST' }),
    );
    expect(got.claimed).toBe(true);
    expect(got.claimed_by).toBe('op-1');
  });

  it('claimSession 对特殊字符做 encodeURIComponent', async () => {
    (apiJson as any).mockResolvedValueOnce({ data: { claimed: false } });
    await claimSession('s with space');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s%20with%20space/claim',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('releaseSession POST /api/sessions/:id/release', async () => {
    (apiJson as any).mockResolvedValueOnce({ data: { ok: true } });
    const got = await releaseSession('s2');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/sessions/s2/release',
      expect.objectContaining({ method: 'POST' }),
    );
    expect(got.ok).toBe(true);
  });

  it('getClaimState GET /api/sessions/:id/claim', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { claimed: false },
    });
    const got = await getClaimState('s3');
    expect(apiJson).toHaveBeenCalledWith('/api/sessions/s3/claim');
    expect(got.claimed).toBe(false);
  });
});

// === privacy.ts 测试 ===

describe('privacy.ts', () => {
  beforeEach(() => vi.clearAllMocks());

  it('deleteVisitorByFingerprint DELETE /api/privacy/visitor/:fp', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: {
        ok: true,
        fingerprint: 'fp-1',
        deleted_sessions: 3,
        deleted_minio_objects: 5,
      },
    });
    const got = await deleteVisitorByFingerprint('fp-1');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/privacy/visitor/fp-1',
      expect.objectContaining({ method: 'DELETE' }),
    );
    expect(got.ok).toBe(true);
    expect(got.deleted_sessions).toBe(3);
  });

  it('deleteVisitorByFingerprint 对特殊字符做 encodeURIComponent', async () => {
    (apiJson as any).mockResolvedValueOnce({
      data: { ok: true, fingerprint: 'a/b' },
    });
    await deleteVisitorByFingerprint('a/b');
    expect(apiJson).toHaveBeenCalledWith(
      '/api/privacy/visitor/a%2Fb',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });
});

// === fetchJson.ts: 非 ok + body parse 失败 fallback ===

describe('fetchJson.ts error fallback', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    setUnauthorizedHandler(() => {});
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('non-2xx with non-json body → catch branch uses statusText', async () => {
    // 返回非 JSON body,触发 .json() 抛错 → 走 catch 分支
    const mockResp = new Response('not json', {
      status: 500,
      statusText: 'Internal Server Error',
    });
    (globalThis.fetch as any).mockResolvedValue(mockResp);

    await expect(fetchJson('/api/x')).rejects.toThrow(/HTTP 500/);
  });

  it('non-2xx with error field uses error message', async () => {
    const mockResp = new Response('{"error":"specific error"}', {
      status: 400,
    });
    (globalThis.fetch as any).mockResolvedValue(mockResp);
    await expect(fetchJson('/api/x')).rejects.toThrow('specific error');
  });

  it('non-2xx with detail field uses detail message', async () => {
    const mockResp = new Response('{"detail":"detailed"}', { status: 422 });
    (globalThis.fetch as any).mockResolvedValue(mockResp);
    await expect(fetchJson('/api/x')).rejects.toThrow('detailed');
  });

  it('204 No Content 返回 undefined', async () => {
    const mockResp = new Response(null, { status: 204 });
    (globalThis.fetch as any).mockResolvedValue(mockResp);
    const got = await fetchJson('/api/x');
    expect(got).toBeUndefined();
  });

  it('POST 带 body 自动加 Content-Type', async () => {
    const mockResp = new Response('{}', { status: 200 });
    (globalThis.fetch as any).mockResolvedValue(mockResp);
    await fetchJson('/api/x', { method: 'POST', body: '{}' });
    const [, init] = (globalThis.fetch as any).mock.calls[0];
    expect((init.headers as any)['Content-Type']).toBe('application/json');
  });

  it('401 触发 unauthorizedHandler 并 throw UNAUTHORIZED', async () => {
    const handler = vi.fn();
    setUnauthorizedHandler(handler);
    const mockResp = new Response('', { status: 401 });
    (globalThis.fetch as any).mockResolvedValue(mockResp);
    await expect(fetchJson('/api/x')).rejects.toThrow('UNAUTHORIZED');
    expect(handler).toHaveBeenCalledTimes(1);
  });
});

// === client.ts: statusText fallback ===
// 注意:文件顶部 vi.mock('../src/api/client') 会覆盖 apiJson,
// 这里用 vi.importActual 拿真实 apiJson。

describe('client.ts apiJson statusText fallback', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('non-2xx with non-json body uses statusText', async () => {
    const actual = await vi.importActual<typeof import('../src/api/client')>('../src/api/client');
    // clone().json() 失败 → fallback statusText
    const mockResp = new Response('not json', {
      status: 502,
      statusText: 'Bad Gateway',
    });
    (globalThis.fetch as any).mockResolvedValue(mockResp);
    await expect(actual.apiJson('/api/x')).rejects.toThrow(/Bad Gateway/);
  });
});

// === App.vue mount ===

describe('App.vue', () => {
  it('mount 含 router-view', async () => {
    const { default: App } = await import('../src/App.vue');
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div>test-content</div>' } }],
    });
    await router.push('/');
    await router.isReady();
    const wrapper = await mount(App, { global: { plugins: [router] } });
    expect(wrapper.html()).toContain('test-content');
  });
});

// === FloatingInput.vue keydown handlers ===

describe('FloatingInput.vue', () => {
  let pinia: ReturnType<typeof createPinia>;
  let i18n: ReturnType<typeof createI18n>;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      messages: { 'zh-CN': {} },
    });
  });

  async function mountInput(props: Record<string, unknown> = {}) {
    const { default: FloatingInput } = await import('../src/components/FloatingInput.vue');
    return mount(FloatingInput, {
      props: { x: 100, y: 100, nodeId: 5, ...props },
      global: { plugins: [pinia, i18n] },
      attachTo: document.body,
    });
  }

  it('渲染基础结构 + label v-if fieldName', async () => {
    const w = await mountInput({ fieldName: 'Email' });
    expect(w.find('label').text()).toBe('Email');
    expect(w.find('input').exists()).toBe(true);
    w.unmount();
  });

  it('未传 fieldName 时不渲染 label,placeholder 用 default', async () => {
    const w = await mountInput();
    expect(w.find('label').exists()).toBe(false);
    w.unmount();
  });

  it('Enter 触发 fill + cancel emit', async () => {
    const w = await mountInput({ fieldName: 'X' });
    const input = w.find('input');
    await input.setValue('hello');
    await input.trigger('keydown', { key: 'Enter' });
    const emitted = w.emitted();
    expect(emitted.fill).toBeTruthy();
    expect(emitted.cancel).toBeTruthy();
    expect(emitted.fill[0]).toEqual([5, 'hello']);
    w.unmount();
  });

  it('Escape 触发 cancel emit (无 fill)', async () => {
    const w = await mountInput();
    const emittedBefore = w.emitted().cancel?.length ?? 0;
    await w.find('input').trigger('keydown', { key: 'Escape' });
    expect(w.emitted().cancel.length).toBe(emittedBefore + 1);
    expect(w.emitted().fill).toBeUndefined();
    w.unmount();
  });

  it('Enter 空 value 只 cancel,不 fill', async () => {
    const w = await mountInput({ nodeId: 0 });
    await w.find('input').trigger('keydown', { key: 'Enter' });
    expect(w.emitted().fill).toBeUndefined();
    expect(w.emitted().cancel).toBeTruthy();
    w.unmount();
  });

  it('blur 触发 cancel', async () => {
    const w = await mountInput();
    await w.find('input').trigger('blur');
    expect(w.emitted().cancel).toBeTruthy();
    w.unmount();
  });

  it('blur 有 value 触发 fill + cancel', async () => {
    const w = await mountInput({ nodeId: 7 });
    await w.find('input').setValue('xyz');
    await w.find('input').trigger('blur');
    expect(w.emitted().fill[0]).toEqual([7, 'xyz']);
    expect(w.emitted().cancel).toBeTruthy();
    w.unmount();
  });
});

// === ReplayViewer.vue: loadInitial error + empty events ===

describe('ReplayViewer.vue', () => {
  let pinia: ReturnType<typeof createPinia>;
  let i18n: ReturnType<typeof createI18n>;
  let router: ReturnType<typeof createRouter>;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      messages: {
        'zh-CN': {
          replay: {
            back: '返回',
            session_label: '会话',
            events_label: '事件',
            loading_more: '加载中',
            loading: '加载中',
            no_events: '无事件',
            play_failed: '播放失败',
          },
        },
      },
    });
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div></div>' } },
        { path: '/replay/:session_id', name: 'replay-viewer', component: { template: '<div></div>' } },
      ],
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.resetModules();
  });

  it('loadInitial 失败显示 error', async () => {
    const mod = await import('../src/api/sessions');
    vi.spyOn(mod, 'getSessionReplay').mockRejectedValueOnce(new Error('network down'));

    await router.push('/replay/sess-err');
    await router.isReady();

    const { default: ReplayViewer } = await import('../src/views/ReplayViewer.vue');
    const w = await mount(ReplayViewer, {
      global: {
        plugins: [pinia, i18n, router],
      },
    });
    await flushPromises();

    expect(w.html()).toMatch(/network down/);
    w.unmount();
  });

  it('loadInitial 返回空 events 显示 no_events', async () => {
    const mod = await import('../src/api/sessions');
    vi.spyOn(mod, 'getSessionReplay').mockResolvedValueOnce({
      events: [],
      total: 0,
      has_more: false,
    } as any);

    await router.push('/replay/sess-empty');
    await router.isReady();

    const { default: ReplayViewer } = await import('../src/views/ReplayViewer.vue');
    const w = await mount(ReplayViewer, {
      global: { plugins: [pinia, i18n, router] },
    });
    await flushPromises();

    expect(w.html()).toMatch(/无事件/);
    w.unmount();
  });

  it('loadInitial 返回 events 触发 player 初始化(mock rrweb-player)', async () => {
    vi.doMock('rrweb-player', () => ({
      default: class {
        constructor(opts: any) {
          (opts.target as HTMLElement).appendChild(document.createElement('div'));
        }
      },
    }));

    const mod = await import('../src/api/sessions');
    vi.spyOn(mod, 'getSessionReplay').mockResolvedValueOnce({
      events: [{ type: 2, data: {} } as any],
      total: 1,
      has_more: false,
    } as any);

    await router.push('/replay/sess-ok');
    await router.isReady();

    const { default: ReplayViewer } = await import('../src/views/ReplayViewer.vue');
    const w = await mount(ReplayViewer, {
      global: { plugins: [pinia, i18n, router] },
    });
    await flushPromises();
    await flushPromises(); // double flush for player init async

    expect(w.find('.player-container').exists()).toBe(true);
    w.unmount();
  });
});

// === LoginView.vue: errorText + onSubmit + defaultHint ===

describe('LoginView.vue', () => {
  let pinia: ReturnType<typeof createPinia>;
  let i18n: ReturnType<typeof createI18n>;
  let router: ReturnType<typeof createRouter>;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    i18n = createI18n({
      legacy: false,
      locale: 'zh-CN',
      messages: {
        'zh-CN': {
          login: {
            title: '登录',
            subtitle: '副标题',
            email: '邮箱',
            password: '密码',
            password_placeholder: '密码',
            signing_in: '登录中',
            sign_in: '登录',
            default_email_hint: '默认: {email}',
            error_credentials: '凭据错误',
            error_session_expired: '会话过期',
          },
        },
      },
    });
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div></div>' } },
        { path: '/dashboard', component: { template: '<div>dashboard</div>' } },
        { path: '/login', component: { template: '<div></div>' } },
      ],
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.resetModules();
  });

  async function mountLogin() {
    const { default: LoginView } = await import('../src/views/LoginView.vue');
    return mount(LoginView, { global: { plugins: [pinia, i18n, router] } });
  }

  it('默认邮箱显示 defaultHint', async () => {
    const w = await mountLogin();
    expect(w.find('.default-hint').text()).toMatch(/admin@pinconsole.local/);
    w.unmount();
  });

  it('非默认邮箱不显示 defaultHint', async () => {
    const w = await mountLogin();
    await w.find('input[type=email]').setValue('custom@example.com');
    expect(w.find('.default-hint').exists()).toBe(false);
    w.unmount();
  });

  it('auth.error = invalid_credentials 显示凭据错误', async () => {
    const { useAuthStore } = await import('../src/stores/auth');
    const auth = useAuthStore();
    auth.error = 'invalid_credentials';

    const w = await mountLogin();
    expect(w.find('.error').text()).toMatch(/凭据错误/);
    w.unmount();
  });

  it('auth.error = SESSION_EXPIRED 显示会话过期', async () => {
    const { useAuthStore } = await import('../src/stores/auth');
    const auth = useAuthStore();
    auth.error = 'SESSION_EXPIRED';

    const w = await mountLogin();
    expect(w.find('.error').text()).toMatch(/会话过期/);
    w.unmount();
  });

  it('auth.error = 其他 显示原文', async () => {
    const { useAuthStore } = await import('../src/stores/auth');
    const auth = useAuthStore();
    auth.error = 'some_other_error';

    const w = await mountLogin();
    expect(w.find('.error').text()).toMatch(/some_other_error/);
    w.unmount();
  });

  it('onSubmit 成功 → router.push(redirect)', async () => {
    const { useAuthStore } = await import('../src/stores/auth');
    const auth = useAuthStore();
    const loginSpy = vi.spyOn(auth, 'login').mockResolvedValueOnce();
    const pushSpy = vi.spyOn(router, 'push');

    await router.push('/login?redirect=/dashboard');
    await router.isReady();

    const w = await mountLogin();
    await w.find('input[type=password]').setValue('pw');
    await w.find('form').trigger('submit.prevent');
    await flushPromises();

    expect(loginSpy).toHaveBeenCalled();
    expect(pushSpy).toHaveBeenCalledWith('/dashboard');
    w.unmount();
  });

  it('onSubmit 失败 不跳转(吞 error)', async () => {
    const { useAuthStore } = await import('../src/stores/auth');
    const auth = useAuthStore();
    vi.spyOn(auth, 'login').mockRejectedValueOnce(new Error('bad'));

    await router.push('/login');
    await router.isReady();

    const pushSpy = vi.spyOn(router, 'push');

    const w = await mountLogin();
    await w.find('input[type=password]').setValue('pw');
    await w.find('form').trigger('submit.prevent');
    await flushPromises();
    // 等待 microtasks 排空
    await new Promise(resolve => setTimeout(resolve, 10));

    // login reject → onSubmit catch 吞掉,push 不应被 onSubmit 调用
    // pushSpy 可能被 router.setup 调用,过滤只看非 /login/dashboard 的
    const pushCallsForSubmit = pushSpy.mock.calls.filter(
      c => c[0] !== '/login' && c[0] !== '/dashboard',
    );
    expect(pushCallsForSubmit.length).toBe(0);
    w.unmount();
  });
});

// === i18n/index.ts: 导入即触发 createI18n ===

describe('i18n/index.ts', () => {
  it('i18n 实例可用,locale 默认 zh-CN', async () => {
    const { i18n } = await import('../src/i18n');
    expect(i18n).toBeTruthy();
    expect(i18n.global.locale.value).toBe('zh-CN');
  });
});
