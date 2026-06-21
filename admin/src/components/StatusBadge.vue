<script setup lang="ts">
// 通用状态 pill —— live/idle/error 等
// 来源:docs/design-system.md §4.2 (Dashboard list item 状态点) + §3 (Topbar LIVE 状态)
// 用法:<StatusBadge variant="success" dot>12 live</StatusBadge>
import { computed } from 'vue';

type Variant = 'success' | 'warning' | 'danger' | 'info' | 'accent' | 'neutral';

const props = withDefaults(
  defineProps<{
    variant?: Variant;
    /** 是否带前置圆点(默认 false) */
    dot?: boolean;
    /** 圆点是否闪烁(仅 active 状态用,如 LIVE) */
    pulse?: boolean;
  }>(),
  {
    variant: 'neutral',
    dot: false,
    pulse: false,
  },
);

const variantClass = computed(() => `pc-badge--${props.variant}`);
</script>

<template>
  <span class="pc-badge status-badge" :class="[variantClass, { 'has-dot': dot }]">
    <span v-if="dot" class="dot" :class="[variantClass, { pulse: pulse }]" aria-hidden="true" />
    <slot />
  </span>
</template>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--pc-font-mono);
  font-size: var(--pc-text-xs);
  font-weight: var(--pc-weight-medium);
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

/* dot 默认色 = neutral;用 variantClass 覆盖 */
.dot {
  width: 6px;
  height: 6px;
  border-radius: var(--pc-radius-pill);
  background: var(--pc-color-text-muted);
  flex-shrink: 0;
}
.dot.pc-badge--success {
  background: var(--pc-color-success);
}
.dot.pc-badge--warning {
  background: var(--pc-color-warning);
}
.dot.pc-badge--danger {
  background: var(--pc-color-danger);
}
.dot.pc-badge--info {
  background: var(--pc-color-info);
}
.dot.pc-badge--accent {
  background: var(--pc-color-accent-default);
}

/* pulse 仅 active 状态(LIVE 等),用 success 色 */
.dot.pulse {
  position: relative;
}
.dot.pulse::after {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: var(--pc-radius-pill);
  background: var(--pc-color-success);
  opacity: 0.4;
  animation: pulse-ring 1.6s var(--pc-easing) infinite;
}
@keyframes pulse-ring {
  0% {
    transform: scale(0.8);
    opacity: 0.5;
  }
  70%,
  100% {
    transform: scale(1.8);
    opacity: 0;
  }
}

/* prefers-reduced-motion 关闭 pulse */
@media (prefers-reduced-motion: reduce) {
  .dot.pulse::after {
    animation: none;
    display: none;
  }
}
</style>
