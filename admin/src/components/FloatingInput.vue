<script setup lang="ts">
// 浮动输入框：运营点 input 后弹出，输入值 + onBlur 后 fill_input
// 详见 docs/progress/2026-06-17-slice-1f-spec.md §浮动输入框

import { ref, watch, nextTick } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps<{
  /** 输入框位置（视口坐标） */
  x: number;
  y: number;
  /** 当前选中的 nodeID（来自 postMessage） */
  nodeId: number;
  /** 字段名提示（rrweb-player 返回） */
  fieldName?: string;
  /** 运营员名字 */
  operatorName?: string;
}>();

const emit = defineEmits<{
  (e: 'fill', nodeId: number, value: string): void;
  (e: 'cancel'): void;
}>();

const value = ref('');
const inputEl = ref<HTMLInputElement | null>(null);

// 弹出时自动聚焦
watch(
  () => props.x,
  async () => {
    await nextTick();
    inputEl.value?.focus();
  },
);

function onBlur() {
  // 防抖 300ms（与 PLAN.md 一致）；这里直接 onBlur 发送，不防抖
  if (value.value && props.nodeId) {
    emit('fill', props.nodeId, value.value);
  }
  emit('cancel');
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault();
    if (value.value && props.nodeId) {
      emit('fill', props.nodeId, value.value);
    }
    emit('cancel');
  } else if (e.key === 'Escape') {
    emit('cancel');
  }
}
</script>

<template>
  <div class="floating-input" :style="{ left: `${x}px`, top: `${y}px` }">
    <label v-if="fieldName">{{ fieldName }}</label>
    <input
      ref="inputEl"
      v-model="value"
      type="text"
      :placeholder="fieldName || t('floating_input.placeholder_default')"
      @blur="onBlur"
      @keydown="onKeydown"
    />
    <div class="hint">{{ t('floating_input.hint') }}</div>
  </div>
</template>

<style scoped>
.floating-input {
  position: fixed;
  z-index: 1000;
  background: var(--pc-color-bg-surface);
  border: 2px solid var(--pc-color-accent-default);
  border-radius: var(--pc-radius-md);
  padding: var(--pc-space-field);
  box-shadow: var(--pc-shadow-lg);
  min-width: 220px;
  font-family: var(--pc-font-sans);
}
label {
  display: block;
  font-size: var(--pc-text-xs);
  color: var(--pc-color-text-secondary);
  margin-bottom: 4px;
}
input {
  width: 100%;
  padding: 4px 8px;
  border: 1px solid var(--pc-color-border-default);
  border-radius: var(--pc-radius-sm);
  font-size: var(--pc-text-base);
  outline: none;
  background: var(--pc-color-bg-surface);
  color: var(--pc-color-text-primary);
}
input:focus {
  border-color: var(--pc-color-accent-default);
  box-shadow: var(--pc-focus-ring);
}
.hint {
  font-size: 11px;
  color: var(--pc-color-text-muted);
  margin-top: 4px;
}
</style>
