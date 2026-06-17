import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';

import App from './App.vue';
import { i18n } from './i18n';

// 切片 1a：admin SPA 入口。
// 装配 Vue + Pinia + Element Plus + i18n（中英双语 from day 1）。
const app = createApp(App);
app.use(createPinia());
app.use(i18n);
app.use(ElementPlus);
app.mount('#app');
