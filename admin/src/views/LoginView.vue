<script setup lang="ts">
// Calm Crafted 登录页 —— Centered Card on Cream
// 来源:docs/design-system.md §4.1
// 替代旧版紫色渐变(frontend-design 明令禁止的 AI-slop tell)
//
// 关键约束:
// - 背景 bg-canvas(暖 cream),不渐变
// - 卡片 400px 宽,padding 40px,shadow-lg,radius-xl
// - 主按钮 accent-default full-width
// - 卡外底部 text-xs muted 表法律告知 + 三标签(Privacy / GDPR / Source)
//   注:v1 暂作纯文本展示(deployer 后期可挂外链)

import { ref, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { PhMonitor } from '@phosphor-icons/vue';
import { useAuthStore } from '../stores/auth';

// 默认 admin 邮箱(与 server config.AdminEmail 默认值同步)。
// 抽为常量而非 i18n key:vue-i18n 把 `@` 解析为 linked-message 引用,
// 写在 message 里会触发 INVALID_LINKED_FORMAT 编译错误。
const DEFAULT_ADMIN_EMAIL = 'admin@pinconsole.local';

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

<template>
  <div class="login-page">
    <main class="login-card" role="main">
      <header class="brand">
        <PhMonitor :size="32" weight="fill" aria-hidden="true" />
        <span class="wordmark">{{ t('app.wordmark') }}</span>
      </header>
      <p class="tagline">{{ t('app.tagline') }}</p>

      <form class="form" @submit.prevent="onSubmit">
        <div class="field">
          <label class="pc-label" for="email">{{ t('login.email') }}</label>
          <input
            id="email"
            v-model="email"
            class="pc-input"
            type="email"
            autocomplete="email"
            :placeholder="t('login.email_placeholder')"
            :disabled="auth.loading"
            required
          />
        </div>
        <div class="field">
          <label class="pc-label" for="password">{{ t('login.password') }}</label>
          <input
            id="password"
            v-model="password"
            class="pc-input"
            type="password"
            autocomplete="current-password"
            :placeholder="t('login.password_placeholder')"
            :disabled="auth.loading"
            required
          />
        </div>

        <p v-if="errorText" class="error" role="alert">
          {{ errorText }}
        </p>

        <button
          type="submit"
          class="pc-btn pc-btn--primary submit"
          :disabled="auth.loading"
        >
          <span v-if="auth.loading" class="spinner" aria-hidden="true" />
          {{ auth.loading ? t('login.signing_in') : t('login.sign_in') }}
        </button>
      </form>

      <p v-if="defaultHint" class="default-hint">{{ defaultHint }}</p>
    </main>

    <footer class="legal">
      <p class="legal-ack">{{ t('login.legal_ack') }}</p>
      <nav class="legal-links" aria-label="legal">
        <span class="link">{{ t('login.privacy_link') }}</span>
        <span class="sep" aria-hidden="true">·</span>
        <span class="link">{{ t('login.gdpr_link') }}</span>
        <span class="sep" aria-hidden="true">·</span>
        <span class="link">{{ t('login.source_link') }}</span>
      </nav>
    </footer>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--pc-space-section);
  padding: var(--pc-space-page);
  background: var(--pc-color-bg-canvas);
}

.login-card {
  width: 100%;
  max-width: 400px;
  padding: 40px;
  background: var(--pc-color-bg-surface);
  border: 1px solid var(--pc-color-border-default);
  border-radius: var(--pc-radius-xl);
  box-shadow: var(--pc-shadow-lg);
}

.brand {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--pc-color-accent-default);
  margin-bottom: 4px;
}

.brand .wordmark {
  font-size: var(--pc-text-xl);
  font-weight: var(--pc-weight-semibold);
  letter-spacing: -0.01em;
}

.tagline {
  margin: 0 0 var(--pc-space-section);
  font-size: var(--pc-text-sm);
  color: var(--pc-color-text-muted);
  text-align: center;
}

.form {
  display: flex;
  flex-direction: column;
}

.field {
  margin-bottom: var(--pc-space-component);
}

.field:last-of-type {
  margin-bottom: var(--pc-space-section);
}

.error {
  margin: 0 0 var(--pc-space-component);
  padding: 8px 12px;
  font-size: var(--pc-text-sm);
  color: var(--pc-color-danger);
  background: var(--pc-color-danger-subtle);
  border-radius: var(--pc-radius-md);
}

.submit {
  width: 100%;
  min-height: 44px;
  font-size: var(--pc-text-md);
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: var(--pc-radius-pill);
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinner {
    animation-duration: 2s;
  }
}

.default-hint {
  margin: var(--pc-space-section) 0 0;
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
  text-align: center;
}

.legal {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  max-width: 400px;
  text-align: center;
}

.legal-ack {
  margin: 0;
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
}

.legal-links {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
}

/* v1 暂作纯文本展示。后期 deployer 可改为 <a target="_blank">:
 * Privacy → /privacy(admin)或外链
 * GDPR → https://gdpr.eu/
 * Source → 项目 repo URL
 */
.legal-links .link {
  cursor: default;
  user-select: none;
}

.legal-links .sep {
  opacity: 0.6;
}
</style>
