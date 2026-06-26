// ReplayList.vue 基础测试：列表渲染 / loading / error / since filter
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { createRouter, createWebHistory } from 'vue-router';
import enUS from '../src/i18n/en-US';
import ReplayList from '../src/views/ReplayList.vue';

// mock api/sessions
const listEndedSessionsMock = vi.fn();
vi.mock('../src/api/sessions', () => ({
  listEndedSessions: (...args: unknown[]) => listEndedSessionsMock(...args),
}));

// mock phosphor icons
vi.mock('@phosphor-icons/vue', () => ({
  PhArrowsClockwise: { template: '<span class="mock-icon" />' },
  PhPlayCircle: { template: '<span class="mock-icon" />' },
  PhWarningCircle: { template: '<span class="mock-icon" />' },
  PhFolderOpen: { template: '<span class="mock-icon" />' },
}));

// mock time util
vi.mock('../src/utils/time', () => ({
  formatRelative: () => 'just now',
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS },
});

const router = createRouter({
  history: createWebHistory(),
  routes: [{ path: '/replay/:id', name: 'replay-viewer', component: { template: '<div/>' } }],
});

function makeSession(id: string, overrides?: Record<string, unknown>) {
  return {
    session_id: id,
    visitor_id: 'v1',
    fingerprint: 'abc123def456',
    started_at: Date.now() - 60000,
    ended_at: Date.now(),
    duration_ms: 60000,
    event_count: 50,
    ua: 'Mozilla/5.0',
    ...overrides,
  };
}

describe('ReplayList.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('加载中显示 refresh 按钮禁用', async () => {
    // 不 resolve mock，让 loading 保持 true
    listEndedSessionsMock.mockReturnValue(new Promise(() => {}));
    const w = mount(ReplayList, {
      global: { plugins: [i18n, router] },
    });
    await flushPromises();
    expect(w.find('.refresh-btn').attributes('disabled')).toBe('');
    w.unmount();
  });

  it('加载成功后显示 sessions（table rows）', async () => {
    listEndedSessionsMock.mockResolvedValue({
      sessions: [makeSession('s1'), makeSession('s2')],
      total: 2,
    });
    const w = mount(ReplayList, {
      global: { plugins: [i18n, router] },
    });
    await flushPromises();
    const rows = w.findAll('tbody tr');
    expect(rows).toHaveLength(2);
    // 每行显示 fingerprint 前 12 字符
    expect(rows[0].text()).toContain('abc123def456'.slice(0, 12));
    w.unmount();
  });

  it('空列表显示 No sessions found', async () => {
    listEndedSessionsMock.mockResolvedValue({ sessions: [], total: 0 });
    const w = mount(ReplayList, {
      global: { plugins: [i18n, router] },
    });
    await flushPromises();
    expect(w.text()).toContain('No sessions found');
    w.unmount();
  });

  it('加载失败显示 error 消息', async () => {
    listEndedSessionsMock.mockRejectedValue(new Error('Network error'));
    const w = mount(ReplayList, {
      global: { plugins: [i18n, router] },
    });
    await flushPromises();
    expect(w.text()).toContain('Network error');
    w.unmount();
  });

  it('切换 since filter 重新加载', async () => {
    listEndedSessionsMock.mockResolvedValue({ sessions: [makeSession('s1')], total: 1 });
    const w = mount(ReplayList, {
      global: { plugins: [i18n, router] },
    });
    await flushPromises();
    vi.clearAllMocks();

    listEndedSessionsMock.mockResolvedValue({ sessions: [makeSession('s7d')], total: 1 });
    // 点 7d
    await w.findAll('.seg')[1].trigger('click');
    await flushPromises();
    expect(listEndedSessionsMock).toHaveBeenCalledWith('7d');
    // 新数据渲染
    expect(w.findAll('tbody tr')).toHaveLength(1);
    w.unmount();
  });

  it('点击 session 行跳转到 replay', async () => {
    listEndedSessionsMock.mockResolvedValue({
      sessions: [makeSession('click-me')],
      total: 1,
    });
    const pushSpy = vi.spyOn(router, 'push');
    const w = mount(ReplayList, {
      global: { plugins: [i18n, router] },
    });
    await flushPromises();
    await w.find('tbody tr').trigger('click');
    expect(pushSpy).toHaveBeenCalledWith('/replay/click-me');
    w.unmount();
  });

  it('formatDuration 格式化时长', async () => {
    listEndedSessionsMock.mockResolvedValue({
      sessions: [
        makeSession('d1', { duration_ms: 500 }),
        makeSession('d2', { duration_ms: 5500 }),
        makeSession('d3', { duration_ms: 125000 }),
        makeSession('d4', { duration_ms: 7200000 }),
      ],
      total: 4,
    });
    const w = mount(ReplayList, {
      global: { plugins: [i18n, router] },
    });
    await flushPromises();
    const rows = w.findAll('tbody tr');
    expect(rows).toHaveLength(4);
    expect(rows[0].text()).toContain('500ms');
    expect(rows[1].text()).toContain('5.5s');
    expect(rows[2].text()).toContain('2min');
    expect(rows[3].text()).toContain('2.0h');
    w.unmount();
  });
});

