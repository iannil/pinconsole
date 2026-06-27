<template>
  <div class="privacy-page">
    <h1 class="page-title">{{ t('privacy.title') }}</h1>
    <p class="page-desc">{{ t('privacy.description') }}</p>

    <section class="pc-card">
      <h2 class="section-title">{{ t('privacy.erasure_title') }}</h2>
      <p class="warn">{{ t('privacy.erasure_warning') }}</p>

      <div class="form-row">
        <input
          v-model="fingerprint"
          type="text"
          :placeholder="t('privacy.fingerprint_placeholder')"
          class="pc-input"
          :disabled="loading"
        />
        <button
          :disabled="loading || !fingerprint.trim()"
          class="pc-btn pc-btn--danger"
          @click="onErase"
        >
          {{ loading ? t('privacy.deleting') : t('privacy.delete') }}
        </button>
      </div>

      <div v-if="result" class="result result-success">
        {{ t('privacy.deleted_sessions', { n: result.deleted_sessions }) }}
        ·
        {{ t('privacy.deleted_objects', { n: result.deleted_minio_objects }) }}
      </div>
      <div v-if="error" class="result result-error">{{ error }}</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { deleteVisitorByFingerprint, type ErasureResponse } from '../api/privacy';

const { t } = useI18n();
const fingerprint = ref('');
const loading = ref(false);
const result = ref<ErasureResponse | null>(null);
const error = ref('');

async function onErase() {
  if (!fingerprint.value.trim()) return;
  if (!confirm(t('privacy.confirm'))) return;

  loading.value = true;
  error.value = '';
  result.value = null;

  try {
    result.value = await deleteVisitorByFingerprint(fingerprint.value.trim());
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.privacy-page {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--pc-space-page);
}

.page-title {
  font-size: var(--pc-text-2xl);
  font-weight: var(--pc-weight-semibold);
  margin: 0 0 var(--pc-space-field);
}

.page-desc {
  color: var(--pc-color-text-secondary);
  line-height: var(--pc-leading-relaxed);
  margin: 0 0 var(--pc-space-section);
}

.section-title {
  font-size: var(--pc-text-lg);
  font-weight: var(--pc-weight-semibold);
  margin: 0 0 var(--pc-space-component);
}

.warn {
  color: var(--pc-color-warning);
  background: var(--pc-color-warning-subtle);
  padding: var(--pc-space-field) var(--pc-space-component);
  border-radius: var(--pc-radius-md);
  font-size: var(--pc-text-sm);
  margin: 0 0 var(--pc-space-card);
}

.form-row {
  display: flex;
  gap: var(--pc-space-field);
}

.form-row .pc-input {
  flex: 1;
}

.result {
  margin-top: var(--pc-space-component);
  padding: var(--pc-space-field) var(--pc-space-component);
  border-radius: var(--pc-radius-md);
  font-size: var(--pc-text-sm);
}

.result-success {
  background: var(--pc-color-success-subtle);
  color: var(--pc-color-success);
}

.result-error {
  background: var(--pc-color-danger-subtle);
  color: var(--pc-color-danger);
}
</style>
