<script setup lang="ts">
// AppTopBar —— 56px sticky 全局顶栏
// 来源:docs/design-system.md §3 (App Shell)
// 结构:Logo · AppNav · spacer · slot(right-side route-specific)· LangToggle · ProfileMenu
import { PhMonitor } from '@phosphor-icons/vue';
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';
import AppNav from './AppNav.vue';
import LangToggle from './LangToggle.vue';
import ProfileMenu from './ProfileMenu.vue';

const { t } = useI18n();
</script>

<template>
  <header class="top-bar" role="banner">
    <div class="left">
      <RouterLink to="/dashboard" class="brand" aria-label="pinconsole">
        <PhMonitor :size="22" weight="fill" aria-hidden="true" />
        <span class="wordmark">{{ t('app.wordmark') }}</span>
      </RouterLink>
      <AppNav />
    </div>

    <div class="right">
      <!-- route-specific extensions (e.g. WS status on Dashboard) -->
      <slot name="extensions" />

      <LangToggle />
      <ProfileMenu />
    </div>
  </header>
</template>

<style scoped>
.top-bar {
  position: sticky;
  top: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--pc-space-component);
  height: var(--pc-topbar-height);
  padding: 0 var(--pc-space-section);
  background: var(--pc-color-bg-surface);
  border-bottom: 1px solid var(--pc-color-border-default);
}

.left,
.right {
  display: inline-flex;
  align-items: center;
  gap: var(--pc-space-section);
  min-width: 0;
}

.left {
  flex: 0 1 auto;
}

.right {
  flex: 0 0 auto;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: var(--pc-color-accent-default);
  font-size: var(--pc-text-lg);
  font-weight: var(--pc-weight-semibold);
  letter-spacing: -0.01em;
  transition: color var(--pc-duration-fast) var(--pc-easing);
  flex-shrink: 0;
}

.brand:hover {
  color: var(--pc-color-accent-hover);
}

.wordmark {
  /* latin 字号 17 时 mono/sans 等宽近似,让 logo 看着稳 */
  font-family: var(--pc-font-sans);
}
</style>
