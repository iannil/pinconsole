// 1h-ui:认证 Pinia store
// 管理 user 信息 + login/logout/fetchMe 方法 + 401 handler 注册。

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { postLogin, postLogout, getMe, type UserInfo, type LoginRequest } from '../api/auth';
import { setUnauthorizedHandler } from '../utils/fetchJson';

export const useAuthStore = defineStore('auth', () => {
  const user = ref<UserInfo | null>(null);
  const loading = ref(false);
  const error = ref('');

  const isAuthenticated = computed(() => user.value !== null);
  const displayName = computed(() => user.value?.display_name ?? user.value?.email ?? '');

  // 注册全局 401 handler:清空 user + 让 router 处理重定向
  // router.beforeEach 会检查 isAuthenticated,自动跳 /login
  setUnauthorizedHandler(() => {
    user.value = null;
    error.value = 'SESSION_EXPIRED';
  });

  async function login(req: LoginRequest): Promise<void> {
    loading.value = true;
    error.value = '';
    try {
      user.value = await postLogin(req);
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function logout(): Promise<void> {
    await postLogout();
    user.value = null;
  }

  async function fetchMe(): Promise<void> {
    try {
      const me = await getMe();
      user.value = me;
    } catch {
      user.value = null;
    }
  }

  return {
    user,
    loading,
    error,
    isAuthenticated,
    displayName,
    login,
    logout,
    fetchMe,
  };
});
