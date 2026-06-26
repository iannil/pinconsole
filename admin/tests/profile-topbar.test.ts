// ProfileMenu.vue + AppTopBar.vue 测试
import { describe, it, expect, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import { createRouter, createWebHistory } from 'vue-router';
import { setActivePinia, createPinia } from 'pinia';
import enUS from '../src/i18n/en-US';
import zhCN from '../src/i18n/zh-CN';
import ProfileMenu from '../src/components/ProfileMenu.vue';
import AppTopBar from '../src/components/AppTopBar.vue';
import { useAuthStore } from '../src/stores/auth';

vi.mock('@phosphor-icons/vue', () => ({
  PhMonitor: { template: '<span class="mock-icon" />' },
  PhPlayCircle: { template: '<span class="mock-icon" />' },
  PhShieldCheck: { template: '<span class="mock-icon" />' },
  PhUserCircle: { template: '<span class="mock-icon" />' },
  PhSignOut: { template: '<span class="mock-icon" />' },
  PhCaretDown: { template: '<span class="mock-icon" />' },
}));

// mock AppNav / LangToggle children
vi.mock('../src/components/AppNav.vue', () => ({
  default: { template: '<nav class="mock-nav" />' },
}));
vi.mock('../src/components/LangToggle.vue', () => ({
  default: { template: '<div class="mock-lang" />' },
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
    { name: 'login', path: '/login', component: { template: '<div/>' } },
  ],
});

describe('ProfileMenu.vue', () => {
  function mountWithAuth(displayName: string) {
    const pinia = createPinia();
    setActivePinia(pinia);
    const auth = useAuthStore();
    auth.$patch({ user: { display_name: displayName, email: `${displayName.toLowerCase()}@test.com` } });
    const w = mount(ProfileMenu, { global: { plugins: [i18n, router, pinia] } });
    return { w, pinia, auth };
  }

  it('显示 displayName 首字母', () => {
    const { w } = mountWithAuth('Alice');
    expect(w.find('.avatar').text()).toBe('A');
    w.unmount();
  });

  it('logout 跳转 login', async () => {
    const { w, auth } = mountWithAuth('Bob');
    vi.spyOn(auth, 'logout').mockResolvedValue();
    const pushSpy = vi.spyOn(router, 'push');
    // 打开下拉
    await w.find('.trigger').trigger('click');
    expect(w.find('.menu').isVisible()).toBe(true);
    // 点 logout
    await w.find('.item.danger').trigger('click');
    await flushPromises();
    expect(pushSpy).toHaveBeenCalledWith({ name: 'login' });
    w.unmount();
  });

  it('点击外部关闭下拉', async () => {
    const { w } = mountWithAuth('X');
    await w.find('.trigger').trigger('click');
    expect(w.find('.menu').isVisible()).toBe(true);
    // 外部点击
    document.body.click();
    await flushPromises();
    expect(w.find('.menu').exists()).toBe(false);
    w.unmount();
  });
});

describe('AppTopBar.vue', () => {
  function mountBar() {
    const pinia = createPinia();
    setActivePinia(pinia);
    const auth = useAuthStore();
    auth.$patch({ user: { display_name: 'Admin', email: 'admin@test.com' } });
    const w = mount(AppTopBar, { global: { plugins: [i18n, router, pinia] } });
    return { w, pinia, auth };
  }

  it('渲染品牌 + 导航 + lang + profile', () => {
    const { w } = mountBar();
    expect(w.text()).toContain('pinconsole');
    expect(w.find('.mock-nav').exists()).toBe(true);
    expect(w.find('.mock-lang').exists()).toBe(true);
    w.unmount();
  });

  it('extensions slot 渲染', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const auth = useAuthStore();
    auth.$patch({ user: { display_name: 'Admin', email: 'admin@test.com' } });
    const w = mount(AppTopBar, {
      slots: { extensions: '<span class="ext-test">EXT</span>' },
      global: { plugins: [i18n, router, pinia] },
    });
    expect(w.find('.ext-test').exists()).toBe(true);
    w.unmount();
  });
});
