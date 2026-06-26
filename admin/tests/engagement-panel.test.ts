// EngagementPanel.vue 基础测试：tab 切换 / popup 发送 / forms tab
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import enUS from '../src/i18n/en-US';
import EngagementPanel from '../src/components/EngagementPanel.vue';

// mock phosphor icons
vi.mock('@phosphor-icons/vue', () => ({
  PhChatsCircle: { template: '<span class="mock-icon" />' },
  PhMegaphone: { template: '<span class="mock-icon" />' },
  PhClipboardText: { template: '<span class="mock-icon" />' },
  PhPaperPlaneTilt: { template: '<span class="mock-icon" />' },
}));

// mock sendCommand + listMessages + sendMessage (ChatPanel child 依赖)
const sendCommandMock = vi.fn();
const listMessagesMock = vi.fn();
const sendMessageMock = vi.fn();
vi.mock('../src/api/sessions', () => ({
  sendCommand: (...args: unknown[]) => sendCommandMock(...args),
  listMessages: (...args: unknown[]) => listMessagesMock(...args),
  sendMessage: (...args: unknown[]) => sendMessageMock(...args),
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS },
});

describe('EngagementPanel.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listMessagesMock.mockResolvedValue({ messages: [] });
    sendMessageMock.mockResolvedValue({ id: 0, sender: 'operator', content: '', created_at: Date.now() });
    setActivePinia(createPinia());
  });

  it('默认显示 chat tab 含 ChatPanel', () => {
    const w = mount(EngagementPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n, createPinia()] },
    });
    expect(w.text()).toContain('Chat');
    expect(w.find('.chat-tab').isVisible()).toBe(true);
    w.unmount();
  });

  it('点击 Popup tab 切换', async () => {
    const w = mount(EngagementPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n, createPinia()] },
    });
    const popupTab = w.findAll('.tab')[1];
    await popupTab.trigger('click');
    expect(w.find('.popup-tab').isVisible()).toBe(true);
    expect(w.find('.chat-tab').isVisible()).toBe(false);
    w.unmount();
  });

  it('点击 Forms tab 显示无表单提交', async () => {
    const w = mount(EngagementPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n, createPinia()] },
    });
    const formsTab = w.findAll('.tab')[2];
    await formsTab.trigger('click');
    expect(w.find('.forms-tab').isVisible()).toBe(true);
    expect(w.text()).toContain('None');
    w.unmount();
  });

  it('popup 发送成功后清空输入', async () => {
    sendCommandMock.mockResolvedValue(undefined);
    const w = mount(EngagementPanel, {
      props: { sessionId: 's1' },
      global: { plugins: [i18n, createPinia()] },
    });
    // 切到 popup tab
    await w.findAll('.tab')[1].trigger('click');

    // 填写 title
    await w.find('#popup-title').setValue('Test Title');
    await w.find('#popup-body').setValue('Test Body');
    await flushPromises();

    // popup tab 内的发送按钮
    const popupBtn = w.find('.popup-tab .send-btn');
    expect(popupBtn.exists()).toBe(true);
    expect(popupBtn.attributes('disabled')).toBeUndefined();
    await popupBtn.trigger('click');
    await flushPromises();

    expect(sendCommandMock).toHaveBeenCalledWith('s1', 'show_popup', {
      title: 'Test Title',
      body: 'Test Body',
      action_label: undefined,
      action_url: undefined,
      dismissible: true,
    });
    // 发送后清空
    expect((w.find('#popup-title').element as HTMLInputElement).value).toBe('');
    w.unmount();
  });

  it('无 sessionId 时不发送 popup', async () => {
    const w = mount(EngagementPanel, {
      props: { sessionId: null },
      global: { plugins: [i18n, createPinia()] },
    });
    await w.findAll('.tab')[1].trigger('click');
    await w.find('#popup-title').setValue('T');
    await w.find('.send-btn').trigger('click');
    await flushPromises();
    expect(sendCommandMock).not.toHaveBeenCalled();
    w.unmount();
  });
});
