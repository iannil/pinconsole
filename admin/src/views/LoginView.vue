<template>
  <div class="login-page">
    <div class="login-card">
      <h1>{{ t('login.title') }}</h1>
      <p class="subtitle">{{ t('login.subtitle') }}</p>

      <form @submit.prevent="onSubmit">
        <div class="field">
          <label>{{ t('login.email') }}</label>
          <input
            v-model="email"
            type="email"
            autocomplete="email"
            :placeholder="DEFAULT_ADMIN_EMAIL"
            :disabled="auth.loading"
            required
          />
        </div>
        <div class="field">
          <label>{{ t('login.password') }}</label>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            :placeholder="t('login.password_placeholder')"
            :disabled="auth.loading"
            required
          />
        </div>

        <div v-if="auth.error" class="error">{{ errorText }}</div>

        <button type="submit" :disabled="auth.loading">
          {{ auth.loading ? t('login.signing_in') : t('login.sign_in') }}
        </button>
      </form>

      <p v-if="defaultHint" class="default-hint">{{ defaultHint }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { useAuthStore } from '../stores/auth';

// 默认 admin 邮箱(与 server config.AdminEmail 默认值同步)。
// 抽为常量而非 i18n key:vue-i18n 把 `@` 解析为 linked-message 引用,
// 写在 message 里会触发 INVALID_LINKED_FORMAT 编译错误。
const DEFAULT_ADMIN_EMAIL = 'admin@marketing-monitor.local';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const auth = useAuthStore();

const email = ref(DEFAULT_ADMIN_EMAIL);
const password = ref('');

// dev 模式默认账号提示(便于快速登录)
const defaultHint = computed(() => {
  if (email.value === DEFAULT_ADMIN_EMAIL) {
    return t('login.default_email_hint', { email: DEFAULT_ADMIN_EMAIL });
  }
  return '';
});

const errorText = computed(() => {
  if (auth.error === 'invalid_credentials') return t('login.error_credentials');
  if (auth.error === 'SESSION_EXPIRED') return t('login.error_session_expired');
  return auth.error;
});

async function onSubmit() {
  try {
    await auth.login({ email: email.value, password: password.value });
    const redirect = (route.query.redirect as string) || '/dashboard';
    router.push(redirect);
  } catch {
    // 错误已在 store 中;UI 显示
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: system-ui, sans-serif;
}
.login-card {
  background: #fff;
  padding: 40px;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  width: 100%;
  max-width: 400px;
}
h1 {
  margin: 0 0 8px;
  font-size: 24px;
  color: #303133;
}
.subtitle {
  margin: 0 0 24px;
  font-size: 14px;
  color: #606266;
}
.field {
  margin-bottom: 16px;
}
label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}
input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  box-sizing: border-box;
}
input:focus {
  outline: none;
  border-color: #409eff;
}
button {
  width: 100%;
  padding: 12px;
  background: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
  margin-top: 8px;
}
button:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}
.error {
  color: #f56c6c;
  font-size: 13px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #fef0f0;
  border-radius: 4px;
}
.default-hint {
  margin-top: 16px;
  font-size: 12px;
  color: #909399;
  text-align: center;
}
</style>
