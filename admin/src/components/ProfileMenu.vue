<script setup lang="ts">
// ProfileMenu —— 顶栏右端账户菜单(头像 + 下拉 + logout)
// 来源:docs/design-system.md §3 (App Shell)
// 注:当前 v1 不显示具体头像图(无 avatar URL 字段),用首字母圆形占位
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { PhUserCircle, PhSignOut, PhCaretDown } from '@phosphor-icons/vue';
import { useAuthStore } from '../stores/auth';

const { t } = useI18n();
const router = useRouter();
const auth = useAuthStore();

const open = ref(false);
const rootEl = ref<HTMLElement | null>(null);

// 取首字母(displayName 首字符,英文取首字母,中文取首字)
const initial = computed(() => {
  const name = auth.displayName;
  if (!name) return '?';
  return name.charAt(0).toUpperCase();
});

function toggle(): void {
  open.value = !open.value;
}

function close(): void {
  open.value = false;
}

async function onLogout(): Promise<void> {
  close();
  await auth.logout();
  router.push({ name: 'login' });
}

function onDocClick(e: MouseEvent): void {
  if (!rootEl.value) return;
  if (!rootEl.value.contains(e.target as Node)) {
    close();
  }
}

function onEsc(e: KeyboardEvent): void {
  if (e.key === 'Escape') close();
}

onMounted(() => {
  document.addEventListener('click', onDocClick);
  document.addEventListener('keydown', onEsc);
});

onUnmounted(() => {
  document.removeEventListener('click', onDocClick);
  document.removeEventListener('keydown', onEsc);
});
</script>

<template>
  <div ref="rootEl" class="profile-menu">
    <button
      type="button"
      class="trigger"
      :aria-label="t('profile.menu_aria')"
      :aria-expanded="open"
      @click="toggle"
    >
      <span class="avatar" aria-hidden="true">{{ initial }}</span>
      <PhCaretDown :size="14" weight="regular" aria-hidden="true" />
    </button>

    <Transition name="dropdown">
      <div v-if="open" class="menu" role="menu">
        <div class="header">
          <PhUserCircle :size="24" weight="regular" aria-hidden="true" />
          <div class="meta">
            <div class="name">{{ auth.displayName || '—' }}</div>
            <div class="hint">{{ t('profile.signed_in_as', { name: auth.displayName || '—' }) }}</div>
          </div>
        </div>
        <button type="button" class="item danger" role="menuitem" @click="onLogout">
          <PhSignOut :size="16" weight="regular" aria-hidden="true" />
          <span>{{ t('profile.logout') }}</span>
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.profile-menu {
  position: relative;
}

.trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 6px 4px 4px;
  border-radius: var(--pc-radius-pill);
  color: var(--pc-color-text-secondary);
  transition: background var(--pc-duration-fast) var(--pc-easing),
    color var(--pc-duration-fast) var(--pc-easing);
}

.trigger:hover {
  background: var(--pc-color-bg-subtle);
  color: var(--pc-color-text-primary);
}

.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--pc-radius-pill);
  background: var(--pc-color-accent-default);
  color: var(--pc-color-accent-on);
  font-size: var(--pc-text-xs);
  font-weight: var(--pc-weight-semibold);
  text-transform: uppercase;
  letter-spacing: 0;
}

.menu {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  min-width: 240px;
  background: var(--pc-color-bg-surface);
  border: 1px solid var(--pc-color-border-default);
  border-radius: var(--pc-radius-lg);
  box-shadow: var(--pc-shadow-lg);
  padding: var(--pc-space-field);
  z-index: 100;
}

.header {
  display: flex;
  align-items: center;
  gap: var(--pc-space-field);
  padding: var(--pc-space-field) 6px var(--pc-space-component);
  border-bottom: 1px solid var(--pc-color-border-default);
  color: var(--pc-color-text-secondary);
  margin-bottom: var(--pc-space-field);
}

.header .meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.header .name {
  font-size: var(--pc-text-sm);
  font-weight: var(--pc-weight-semibold);
  color: var(--pc-color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header .hint {
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item {
  display: flex;
  align-items: center;
  gap: var(--pc-space-field);
  width: 100%;
  padding: 8px 6px;
  font-size: var(--pc-text-sm);
  color: var(--pc-color-text-secondary);
  border-radius: var(--pc-radius-md);
  transition: background var(--pc-duration-fast) var(--pc-easing),
    color var(--pc-duration-fast) var(--pc-easing);
}

.item:hover {
  background: var(--pc-color-bg-subtle);
  color: var(--pc-color-text-primary);
}

.item.danger:hover {
  background: var(--pc-color-danger-subtle);
  color: var(--pc-color-danger);
}

/* dropdown transition:fade + slight scale */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity var(--pc-duration-fast) var(--pc-easing),
    transform var(--pc-duration-fast) var(--pc-easing);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.96);
}
</style>
