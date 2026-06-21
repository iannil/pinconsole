<script setup lang="ts">
// Calm Crafted 中/EN 切换 —— 2 段 pill toggle(代替单按钮)
// 来源:docs/design-system.md §3 (App Shell 顶栏)
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

const { locale, availableLocales } = useI18n();

// 取语言短码:zh-CN → 中,en-US → EN。availableLocales 里查不到时回退首字母大写。
const options = computed(() =>
  availableLocales.map((l) => ({
    code: l,
    label: l.startsWith('zh') ? '中' : l.startsWith('en') ? 'EN' : l.slice(0, 2).toUpperCase(),
  })),
);

function setLocale(code: string): void {
  locale.value = code;
}
</script>

<template>
  <div class="lang-toggle" role="group" :aria-label="$t('app.switch_lang')">
    <button
      v-for="opt in options"
      :key="opt.code"
      type="button"
      class="seg"
      :class="{ active: locale === opt.code }"
      :aria-pressed="locale === opt.code"
      @click="setLocale(opt.code)"
    >
      {{ opt.label }}
    </button>
  </div>
</template>

<style scoped>
.lang-toggle {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  background: var(--pc-color-bg-subtle);
  border-radius: var(--pc-radius-pill);
}
.seg {
  padding: 4px 10px;
  font-size: var(--pc-text-xs);
  font-weight: var(--pc-weight-medium);
  color: var(--pc-color-text-muted);
  border-radius: var(--pc-radius-pill);
  transition: color var(--pc-duration-fast) var(--pc-easing),
    background var(--pc-duration-fast) var(--pc-easing);
  min-width: 32px;
  line-height: 1.2;
}
.seg:hover:not(.active) {
  color: var(--pc-color-text-secondary);
}
.seg.active {
  background: var(--pc-color-bg-surface);
  color: var(--pc-color-text-primary);
  box-shadow: var(--pc-shadow-xs);
}
</style>
