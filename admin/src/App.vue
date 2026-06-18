<script setup lang="ts">
// 1b:App 仅作 router-view 容器
// 1h-ui:加载时 fetchMe 验证 cookie,触发 router 守卫
import { onMounted } from 'vue';
import { useAuthStore } from './stores/auth';

const auth = useAuthStore();

onMounted(async () => {
  // 异步 fetchMe,完成后 router 才能正确判断 isAuthenticated
  await auth.fetchMe();
});
</script>

<template>
  <router-view />
</template>
