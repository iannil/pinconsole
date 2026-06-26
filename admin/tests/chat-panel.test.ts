// ChatPanel.vue 基础测试：空状态 / paused 状态 / 消息渲染 / 输入框禁用
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import enUS from '../src/i18n/en-US';
import ChatPanel from '../src/components/ChatPanel.vue';

// mock api/sessions
const listMessagesMock = vi.fn();
const sendMessageMock = vi.fn();
vi.mock('../src/api/sessions', () => ({
  listMessages: (...args: unknown[]) => listMessagesMock(...args),
  sendMessage: (...args: unknown[]) => sendMessageMock(...args),
}));

// mock phosphor icons (SFC 组件)
vi.mock('@phosphor-icons/vue', () => ({
  PhPaperPlaneTilt: { template: '<span class="mock-icon" />' },
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS },
});

describe('ChatPanel.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listMessagesMock.mockResolvedValue({ messages: [] });
    sendMessageMock.mockResolvedValue({ id: 1, sender: 'operator', content: 'hi', created_at: Date.now() });
  });

  it('empty state 显示 No messages yet', async () => {
    const w = mount(ChatPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    expect(w.text()).toContain('No messages yet');
    w.unmount();
  });

  it('无 sessionId 时不调 API 且 input 禁用', () => {
    const w = mount(ChatPanel, {
      props: { sessionId: null },
      global: { plugins: [i18n] },
    });
    expect(listMessagesMock).not.toHaveBeenCalled();
    expect(w.find('input').attributes('disabled')).toBeDefined();
    expect(w.find('button').attributes('disabled')).toBeDefined();
    w.unmount();
  });

  it('sessionId 变化后调 refresh', async () => {
    const w = mount(ChatPanel, {
      props: { sessionId: null },
      global: { plugins: [i18n] },
    });
    await w.setProps({ sessionId: 's2' });
    await flushPromises();
    expect(listMessagesMock).toHaveBeenCalledWith('s2', 0);
    w.unmount();
  });

  it('listMessages 返回 401 → paused auth_required', async () => {
    listMessagesMock.mockRejectedValueOnce(new Error('HTTP 401'));
    const w = mount(ChatPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    expect(w.text()).toContain('expired');
    expect(w.find('input').attributes('disabled')).toBeDefined();
    w.unmount();
  });

  it('listMessages 返回 403 → paused not_claimed', async () => {
    listMessagesMock.mockRejectedValueOnce(new Error('HTTP 403'));
    const w = mount(ChatPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    expect(w.text()).toContain('Claim');
    expect(w.find('input').attributes('disabled')).toBeDefined();
    w.unmount();
  });

  it('渲染消息列表', async () => {
    listMessagesMock.mockResolvedValueOnce({
      messages: [
        { id: 1, sender: 'visitor', content: 'Hello', created_at: Date.now() },
        { id: 2, sender: 'operator', content: 'Hi there', created_at: Date.now() },
      ],
    });
    const w = mount(ChatPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    expect(w.text()).toContain('Hello');
    expect(w.text()).toContain('Hi there');
    expect(w.findAll('.msg')).toHaveLength(2);
    w.unmount();
  });

  it('send 后消息追加', async () => {
    listMessagesMock.mockResolvedValue({ messages: [] });
    sendMessageMock.mockResolvedValue({ id: 10, sender: 'operator', content: 'sent msg', created_at: Date.now() });
    const w = mount(ChatPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    await w.find('input').setValue('sent msg');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(sendMessageMock).toHaveBeenCalledWith('s1', 'sent msg');
    expect(w.text()).toContain('sent msg');
    w.unmount();
  });

  it('unmount 清理定时器', async () => {
    const w = mount(ChatPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n] },
    });
    await flushPromises();
    expect(listMessagesMock).toHaveBeenCalled();
    w.unmount();
    // unmount 后不再有新的 refresh 调
    const callCount = listMessagesMock.mock.calls.length;
    await new Promise(r => setTimeout(r, 50));
    expect(listMessagesMock.mock.calls.length).toBe(callCount);
  });
});
