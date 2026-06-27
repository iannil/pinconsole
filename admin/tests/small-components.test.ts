// AppNav + StatusBadge + LangToggle — 小型组件联合测试
import { describe, it, expect, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { createRouter, createWebHistory } from 'vue-router';
import enUS from '../src/i18n/en-US';
import zhCN from '../src/i18n/zh-CN';
import StatusBadge from '../src/components/StatusBadge.vue';
import AppNav from '../src/components/AppNav.vue';
import LangToggle from '../src/components/LangToggle.vue';

// === mock phosphor icons ===
vi.mock('@phosphor-icons/vue', () => ({
  PhMonitor: { template: '<span class="mock-icon" />' },
  PhPlayCircle: { template: '<span class="mock-icon" />' },
  PhShieldCheck: { template: '<span class="mock-icon" />' },
  PhPencilSimple: { template: '<span class="mock-icon" />' },
  PhFile: { template: '<span class="mock-icon" />' },
  PhUserCircle: { template: '<span class="mock-icon" />' },
  PhSignOut: { template: '<span class="mock-icon" />' },
  PhCaretDown: { template: '<span class="mock-icon" />' },
}));

const i18n = createI18n({
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en: enUS, 'zh-CN': zhCN },
});

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/dashboard', component: { template: '<div/>' } },
    { path: '/replay', component: { template: '<div/>' } },
    { path: '/privacy', component: { template: '<div/>' } },
    { name: 'login', path: '/login', component: { template: '<div/>' } },
  ],
});

// ============== StatusBadge ==============
describe('StatusBadge.vue', () => {
  it('默认 neutral 样式', () => {
    const w = mount(StatusBadge, { slots: { default: '12 live' } });
    expect(w.text()).toContain('12 live');
    expect(w.find('.pc-badge--neutral').exists()).toBe(true);
    w.unmount();
  });

  it('variant=success', () => {
    const w = mount(StatusBadge, { props: { variant: 'success' }, slots: { default: 'ok' } });
    expect(w.find('.pc-badge--success').exists()).toBe(true);
    w.unmount();
  });

  it('dot 属性显示圆点', () => {
    const w = mount(StatusBadge, { props: { dot: true } });
    expect(w.find('.dot').exists()).toBe(true);
    w.unmount();
  });

  it('pulse 添加 pulse 类', () => {
    const w = mount(StatusBadge, { props: { dot: true, pulse: true } });
    expect(w.find('.dot.pulse').exists()).toBe(true);
    w.unmount();
  });
});

// ============== AppNav ==============
describe('AppNav.vue', () => {
  it('渲染 5 个导航项', () => {
    const w = mount(AppNav, { global: { plugins: [i18n, router] } });
    const items = w.findAll('.nav-item');
    expect(items).toHaveLength(5);
    expect(items[0].text()).toContain('Live Monitor');
    expect(items[1].text()).toContain('History');
    expect(items[2].text()).toContain('Privacy');
    expect(items[3].text()).toContain('Widgets');
    expect(items[4].text()).toContain('Pages');
    w.unmount();
  });
});

// ============== LangToggle ==============
describe('LangToggle.vue', () => {
  it('渲染 EN/中 两个按钮', () => {
    const w = mount(LangToggle, { global: { plugins: [i18n] } });
    const btns = w.findAll('.seg');
    expect(btns).toHaveLength(2);
    expect(btns[0].text()).toBe('EN');
    expect(btns[1].text()).toBe('中');
    w.unmount();
  });

  it('点击按钮切换 locale', async () => {
    const w = mount(LangToggle, { global: { plugins: [i18n] } });
    // 初始 en,第一个按钮 EN 高亮
    expect(w.findAll('.seg')[0].classes()).toContain('active');
    // 点第二个(中)
    await w.findAll('.seg')[1].trigger('click');
    expect(w.findAll('.seg')[1].classes()).toContain('active');
    expect(w.findAll('.seg')[0].classes()).not.toContain('active');
    w.unmount();
  });
});
