import { createApp } from 'vue';
import { createPinia } from 'pinia';

// 全局样式接入顺序(来源:docs/design-system.md §6.1):
//   reset → tokens → fonts → base
// reset 统一浏览器默认;tokens 提供 :root CSS variables(fonts/base 依赖);
// fonts 接入 @fontsource(self-host woff2,Vite bundle 到 dist/);base 提供原子 utility(.pc-btn / .pc-input / .pc-card 等)
import './styles/reset.css';
import './styles/tokens.css';
import './styles/fonts.css';
import './styles/base.css';

import App from './App.vue';
import { i18n } from './i18n';
import { router } from './router';

// 切片 1a:admin SPA 入口。
// 切片 1b:装配 Vue + Pinia + i18n + Router。
// 1q:移除 Element Plus(注册后零使用,bundle 减重 ~700KB)
// design-system Phase 1:接入 Calm Crafted 设计 token(IBM Plex + Stone+Teal+Amber + Phosphor 待接入)
const app = createApp(App);
app.use(createPinia());
app.use(i18n);
app.use(router);
app.mount('#app');
