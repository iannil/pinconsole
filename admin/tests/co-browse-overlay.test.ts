// CoBrowseOverlay.vue 基础测试：显示/隐藏状态 / 事件处理
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import enUS from '../src/i18n/en-US';
import CoBrowseOverlay from '../src/components/CoBrowseOverlay.vue';

const sendCommandMock = vi.fn();
vi.mock('../src/api/sessions', () => ({
  sendCommand: (...args: unknown[]) => sendCommandMock(...args),
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS },
});

describe('CoBrowseOverlay.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('active=false 不渲染', () => {
    const w = mount(CoBrowseOverlay, {
      props: { sessionId: 's1', active: false },
      global: { plugins: [i18n] },
    });
    expect(w.find('.co-browse-overlay').exists()).toBe(false);
    w.unmount();
  });

  it('active=true 渲染 overlay', () => {
    const w = mount(CoBrowseOverlay, {
      props: { sessionId: 's1', active: true },
      global: { plugins: [i18n] },
    });
    expect(w.find('.co-browse-overlay').exists()).toBe(true);
    w.unmount();
  });

  it('active=true + 无 sessionId 渲染但事件不发送', async () => {
    sendCommandMock.mockResolvedValue(undefined);
    const w = mount(CoBrowseOverlay, {
      props: { sessionId: null, active: true },
      global: { plugins: [i18n] },
    });
    expect(w.find('.co-browse-overlay').exists()).toBe(true);
    // 点击不会发送命令（sessionId 为 null）
    await w.find('.co-browse-overlay').trigger('click');
    await new Promise(r => setTimeout(r, 50));
    expect(sendCommandMock).not.toHaveBeenCalled();
    w.unmount();
  });

  it('显示命令计数 badge', () => {
    const w = mount(CoBrowseOverlay, {
      props: { sessionId: 's1', active: true },
      global: { plugins: [i18n] },
    });
    expect(w.find('.badge').exists()).toBe(true);
    w.unmount();
  });

  it('点击触发 command-sent 事件', async () => {
    sendCommandMock.mockResolvedValue(undefined);
    const w = mount(CoBrowseOverlay, {
      props: { sessionId: 's1', active: true },
      global: { plugins: [i18n] },
    });
    await w.find('.co-browse-overlay').trigger('click');
    // click 会调 requestNodeIdAt → sendClick → sendCommand
    // 由于没有 iframe，requestNodeIdAt 返回 0，sendCommand 仍然被调
    await new Promise(r => setTimeout(r, 50));
    expect(sendCommandMock).toHaveBeenCalled();
    w.unmount();
  });
});
