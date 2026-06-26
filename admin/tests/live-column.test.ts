// LiveColumn.vue 基础测试：未选 visitor / 未 claim / 已 claim 三种状态
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { setActivePinia, createPinia } from 'pinia';
import enUS from '../src/i18n/en-US';
import LiveColumn from '../src/components/LiveColumn.vue';
import { useVisitorsStore } from '../src/stores/visitors';

// mock phosphor icons
vi.mock('@phosphor-icons/vue', () => ({
  PhHand: { template: '<span class="mock-icon" />' },
  PhX: { template: '<span class="mock-icon" />' },
  PhWarningCircle: { template: '<span class="mock-icon" />' },
}));

// mock ReplayPlayer (child)
vi.mock('../src/components/ReplayPlayer.vue', () => ({
  default: { template: '<div class="mock-replay" />' },
}));

// mock CoBrowseOverlay (child)
vi.mock('../src/components/CoBrowseOverlay.vue', () => ({
  default: { template: '<div class="mock-overlay" />' },
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS },
});

function createStoreWithVisitor() {
  const pinia = createPinia();
  setActivePinia(pinia);
  const store = useVisitorsStore();
  store.setInitialList([{
    sessionId: 'sess-1',
    visitorId: 'v1',
    fingerprint: 'abc123def4567890',
    startedAt: Date.now() - 60000,
    lastEventAt: null,
    eventCount: 0,
  }]);
  store.select('sess-1');
  return { pinia, store };
}

describe('LiveColumn.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('未选 visitor 显示 placeholder', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const w = mount(LiveColumn, {
      props: { coBrowsingActive: false, claimError: '', cobrowseHint: '', operatorName: 'Op' },
      global: { plugins: [i18n, pinia] },
    });
    expect(w.text()).toContain('Select a visitor');
    w.unmount();
  });

  it('已选 visitor + 未 claim 显示 Start Co-browsing 按钮', () => {
    const { pinia } = createStoreWithVisitor();
    const w = mount(LiveColumn, {
      props: { coBrowsingActive: false, claimError: '', cobrowseHint: '', operatorName: 'Op' },
      global: { plugins: [i18n, pinia] },
    });
    expect(w.text()).toContain('Start Co-browsing');
    expect(w.find('.mock-replay').exists()).toBe(true);
    w.unmount();
  });

  it('已 claim 显示 Stop Co-browsing 按钮 + overlay', () => {
    const { pinia } = createStoreWithVisitor();
    const w = mount(LiveColumn, {
      props: { coBrowsingActive: true, claimError: '', cobrowseHint: 'Control mode active', operatorName: 'Op' },
      global: { plugins: [i18n, pinia] },
    });
    expect(w.text()).toContain('Stop Co-browsing');
    expect(w.find('.mock-overlay').exists()).toBe(true);
    w.unmount();
  });

  it('点击 claim/release 按钮 emit toggle-cobrowse', () => {
    const { pinia } = createStoreWithVisitor();
    const w = mount(LiveColumn, {
      props: { coBrowsingActive: false, claimError: '', cobrowseHint: '', operatorName: 'Op' },
      global: { plugins: [i18n, pinia] },
    });
    const btn = w.find('.controls .pc-btn');
    expect(btn.exists()).toBe(true);
    btn.trigger('click');
    expect(w.emitted('toggle-cobrowse')).toHaveLength(1);
    w.unmount();
  });

  it('显示 claim 错误提示', () => {
    const { pinia } = createStoreWithVisitor();
    const w = mount(LiveColumn, {
      props: { coBrowsingActive: false, claimError: 'Session timed out', cobrowseHint: '', operatorName: 'Op' },
      global: { plugins: [i18n, pinia] },
    });
    expect(w.text()).toContain('timed out');
    w.unmount();
  });
});
