// TS-4 切片补测:VisitorList.vue 组件测试。
// 测试策略:mount 真组件 + mock store + 验证 DOM 渲染 + 交互。
// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia, type Pinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import VisitorList from '../src/components/VisitorList.vue';
import { useVisitorsStore } from '../src/stores/visitors';

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      dashboard: { online_count: '在线 {count}' },
      visitor: {
        flagged_tooltip: '已标记',
        flagged_tooltip_with_reason: '已标记:{reason}',
      },
    },
  },
});

describe('VisitorList.vue', () => {
  let pinia: Pinia;

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
  });

  it('空列表渲染 0 count', () => {
    const wrapper = mount(VisitorList, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    expect(wrapper.text()).toContain('在线 0');
  });

  it('渲染 visitor 列表 + 点击触发 select', async () => {
    const store = useVisitorsStore();
    // 用 setInitialList 填充(不用直接赋值 computed)
    store.setInitialList([
      {
        sessionId: 's1',
        fingerprint: 'fp-aaaaaaaaaaaaaa',
        isFlagged: false,
      },
      {
        sessionId: 's2',
        fingerprint: 'fp-bbbbbbbbbbbb',
        isFlagged: true,
        flagReason: 'no_mouse',
      },
    ] as any);

    const wrapper = mount(VisitorList, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    // 应渲染 2 个 li
    const items = wrapper.findAll('li');
    expect(items.length).toBe(2);

    // 第一个 li 含 fingerprint 前 12 字符
    expect(wrapper.text()).toContain('fp-aaaaaaaa');

    // 点击第一个 li 触发 store.select
    await items[0].trigger('click');
    expect(store.selectedSessionId).toBe('s1');
  });

  it('flagged visitor 渲染 flag icon', () => {
    const store = useVisitorsStore();
    store.setInitialList([
      {
        sessionId: 's1',
        fingerprint: 'fp-aaaaaaaaaaaa',
        isFlagged: true,
        flagReason: 'repetitive_clicks',
      },
    ] as any);

    const wrapper = mount(VisitorList, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    // Phase 3:🚩 emoji 已换 Phosphor PhFlag(SVG)。验证 .flag-icon span + 内含 svg
    const flagIcon = wrapper.find('.flag-icon');
    expect(flagIcon.exists()).toBe(true);
    expect(flagIcon.find('svg').exists()).toBe(true);
  });

  it('flagTitle 无 reason 返回默认 tooltip', async () => {
    const store = useVisitorsStore();
    store.setInitialList([
      {
        sessionId: 's1',
        fingerprint: 'fp-aaaaaaaaaaaa',
        isFlagged: true,
        // 无 flagReason
      },
    ] as any);

    const wrapper = mount(VisitorList, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    // 🚩 icon 的 title 属性应为默认 tooltip(无 reason)
    const flagIcon = wrapper.find('.flag-icon');
    expect(flagIcon.exists()).toBe(true);
    expect(flagIcon.attributes('title')).toBe('已标记');
  });

  it('flagTitle 含 reason 返回带 reason tooltip', async () => {
    const store = useVisitorsStore();
    store.setInitialList([
      {
        sessionId: 's1',
        fingerprint: 'fp-aaaaaaaaaaaa',
        isFlagged: true,
        flagReason: 'no_mouse_events',
      },
    ] as any);

    const wrapper = mount(VisitorList, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    const flagIcon = wrapper.find('.flag-icon');
    expect(flagIcon.attributes('title')).toBe('已标记:no_mouse_events');
  });

  it('selected visitor 应用 selected class', async () => {
    const store = useVisitorsStore();
    store.setInitialList([
      { sessionId: 's1', fingerprint: 'fp-aaaaaaaa', isFlagged: false },
    ] as any);

    const wrapper = mount(VisitorList, {
      global: {
        plugins: [pinia, i18n],
      },
    });

    // 点击触发 select
    await wrapper.find('li').trigger('click');
    expect(store.selectedSessionId).toBe('s1');

    // li 应有 selected class
    expect(wrapper.find('li').classes()).toContain('selected');
  });
});
