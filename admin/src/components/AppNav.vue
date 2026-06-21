<script setup lang="ts">
// AppNav —— 顶栏主导航(Dashboard / Replay / Privacy)
// 来源:docs/design-system.md §3 (App Shell)
// active 态:RouterLink router-link-active 类 + accent 下划线
import { useI18n } from 'vue-i18n';
import { PhMonitor, PhPlayCircle, PhShieldCheck } from '@phosphor-icons/vue';

const { t } = useI18n();

interface NavItem {
  to: string;
  labelKey: string;
  Icon: typeof PhMonitor;
}

// 注意:replay 列表 + replay/:session_id 都应高亮 "Replay" 项
// 通过 RouterLink 的 active-class 控制 + 自定义匹配规则
const items: NavItem[] = [
  { to: '/dashboard', labelKey: 'nav.dashboard', Icon: PhMonitor },
  { to: '/replay', labelKey: 'nav.replay', Icon: PhPlayCircle },
  { to: '/privacy', labelKey: 'nav.privacy', Icon: PhShieldCheck },
];
</script>

<template>
  <nav class="app-nav" aria-label="primary">
    <RouterLink
      v-for="item in items"
      :key="item.to"
      :to="item.to"
      class="nav-item"
      active-class="active"
      :class="{ 'active-prefix': item.to === '/replay' }"
    >
      <component :is="item.Icon" :size="18" weight="regular" aria-hidden="true" />
      <span>{{ t(item.labelKey) }}</span>
    </RouterLink>
  </nav>
</template>

<style scoped>
.app-nav {
  display: inline-flex;
  align-items: center;
  gap: var(--pc-space-component);
  height: 100%;
}

.nav-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 4px;
  height: 100%;
  font-size: var(--pc-text-sm);
  font-weight: var(--pc-weight-medium);
  color: var(--pc-color-text-secondary);
  text-decoration: none;
  position: relative;
  transition: color var(--pc-duration-fast) var(--pc-easing);
}

.nav-item:hover {
  color: var(--pc-color-text-primary);
}

/* active 态:accent 下划线 + primary 文本 */
.nav-item.active,
/* /replay/:session_id 也算 replay active(prefix match via active-class on parent path) */
.nav-item.active-prefix.router-link-active {
  color: var(--pc-color-text-primary);
}

.nav-item.active::after,
.nav-item.active-prefix.router-link-active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px; /* 覆盖顶栏下边框 */
  height: 2px;
  background: var(--pc-color-accent-default);
  border-radius: var(--pc-radius-pill);
}
</style>
