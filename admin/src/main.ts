import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';

import App from './App.vue';
import { i18n } from './i18n';
import { router } from './router';

// 切片 1a：admin SPA 入口。
// 切片 1b：装配 Vue + Pinia + Element Plus + i18n + Router。
const app = createApp(App);
app.use(createPinia());
app.use(i18n);
app.use(router);
app.use(ElementPlus);
app.mount('#app');
