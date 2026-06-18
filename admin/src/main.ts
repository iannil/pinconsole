import { createApp } from 'vue';
import { createPinia } from 'pinia';

import App from './App.vue';
import { i18n } from './i18n';
import { router } from './router';

// 切片 1a:admin SPA 入口。
// 切片 1b:装配 Vue + Pinia + i18n + Router。
// 1q:移除 Element Plus(注册后零使用,bundle 减重 ~700KB)
const app = createApp(App);
app.use(createPinia());
app.use(i18n);
app.use(router);
app.mount('#app');
