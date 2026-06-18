<script setup lang="ts">
// 1b:App 仅作 router-view 容器
// 1h-ui:fetchMe 已迁移到 router.beforeEach 的 ensureAuthInit(见 router/index.ts)
// 原因:在 onMounted 调 fetchMe 时,router 守卫在挂载前已执行,
// 导致刷新 /admin/* 误跳 /login。修复后首次 navigation 即触发 fetchMe 并 await。
import { } from 'vue';
</script>

<template>
  <router-view />
</template>
