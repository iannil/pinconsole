<script setup lang="ts">
// AppShell —— 认证路由的 layout 包壳
// 来源:docs/design-system.md §3 (App Shell = Top bar + Content)
//
// 挂载方式:router/index.ts 把 /dashboard /replay /privacy 等作为本组件的 children,
// 本组件的 <router-view> 渲染具体页面。
// LoginView 留在 shell 之外(独立路由)。
import AppTopBar from '../components/AppTopBar.vue';
</script>

<template>
  <div class="app-shell">
    <AppTopBar />
    <main class="app-main">
      <slot>
        <router-view />
      </slot>
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--pc-color-bg-canvas);
  color: var(--pc-color-text-primary);
}

.app-main {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  height: 100%;
}
</style>
