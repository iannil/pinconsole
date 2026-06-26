// ReplayPlayer.vue 基础测试：loading/empty/error 状态 + events/sessionId 响应
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import enUS from '../src/i18n/en-US';
import ReplayPlayer from '../src/components/ReplayPlayer.vue';

// mock replay-core 模块（factory 内部创建 vi.fn，避免 hoist 捕获问题）
vi.mock('@pinconsole/replay-core', () => {
  const mockReplayer = vi.fn();
  return {
    Replayer: mockReplayer.mockImplementation((_events: unknown[], config: any) => {
      const wrapper = document.createElement('div');
      wrapper.className = 'replayer-wrapper';
      if (config.root) config.root.appendChild(wrapper);
      const iframe = document.createElement('iframe');
      return {
        wrapper,
        iframe,
        config,
        on: vi.fn(),
        destroy: vi.fn(),
        startLive: vi.fn(),
        addEvent: vi.fn(),
        handleResize: vi.fn(),
        getMetaData: () => ({ totalTime: 10000, startTime: 0, endTime: 10000 }),
        getCurrentTime: () => 0,
      };
    }),
  };
});

// mock useResponsivePlayerSize
vi.mock('../src/composables/useResponsivePlayerSize', () => ({
  useResponsivePlayerSize: () => ({
    start: vi.fn(),
    stop: vi.fn(),
  }),
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS },
});

/** 构造 payload 数组供 ReplayPlayer props.events 使用 */
function makePayload(rrwebEvents: unknown[]) {
  return rrwebEvents.map((e) => ({
    v: 1,
    type: 'rrweb' as const,
    ts: Date.now(),
    rrweb: e,
  }));
}

describe('ReplayPlayer.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('初始渲染显示 loading', () => {
    const w = mount(ReplayPlayer, {
      props: { events: [], sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    expect(w.text()).toContain('Loading');
    w.unmount();
  });

  it('空 events 在 async 完成后显示 waiting_events', async () => {
    const w = mount(ReplayPlayer, {
      props: { events: [], sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    expect(w.text()).toContain('Subscribed');
    w.unmount();
  });

  it('≥2 个 rrweb 事件不崩溃', async () => {
    const events = makePayload([
      { type: 4, data: { width: 1024, height: 768, href: 'about:blank' }, timestamp: 1 },
      { type: 2, data: { node: { type: 0, childNodes: [] }, initialOffset: { top: 0, left: 0 } }, timestamp: 2 },
    ]);
    const w = mount(ReplayPlayer, {
      props: { events, sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    expect(w.find('.player-container').exists()).toBe(true);
    w.unmount();
  });
});
