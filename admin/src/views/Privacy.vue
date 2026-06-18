<template>
  <div class="privacy-page">
    <h1>{{ t('privacy.title') }}</h1>
    <p class="desc">{{ t('privacy.description') }}</p>

    <section class="card">
      <h2>{{ t('privacy.erasure_title') }}</h2>
      <p class="warn">{{ t('privacy.erasure_warning') }}</p>

      <div class="form">
        <input
          v-model="fingerprint"
          type="text"
          :placeholder="t('privacy.fingerprint_placeholder')"
          class="fp-input"
          :disabled="loading"
        />
        <button
          :disabled="loading || !fingerprint.trim()"
          class="delete-btn"
          @click="onErase"
        >
          {{ loading ? t('privacy.deleting') : t('privacy.delete') }}
        </button>
      </div>

      <div v-if="result" class="result success">
        {{ t('privacy.deleted_sessions', { n: result.deleted_sessions }) }}
        ·
        {{ t('privacy.deleted_objects', { n: result.deleted_minio_objects }) }}
      </div>
      <div v-if="error" class="result error">{{ error }}</div>
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
  padding: 24px 32px;
  font-family: system-ui, sans-serif;
  color: #303133;
}
.desc {
  color: #606266;
  line-height: 1.6;
  margin-bottom: 24px;
}
.card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 20px 24px;
  margin-bottom: 16px;
}
.card h2 {
  font-size: 18px;
  margin: 0 0 12px;
}
.warn {
  color: #856404;
  background: #fff3cd;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 13px;
  margin-bottom: 16px;
}
.form {
  display: flex;
  gap: 8px;
}
.fp-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
}
.delete-btn {
  background: #dc3545;
  color: white;
  border: none;
  padding: 8px 20px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}
.delete-btn:disabled {
  background: #c0c4cc;
  cursor: not-allowed;
}
.result {
  margin-top: 12px;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 13px;
}
.result.success {
  background: #f0f9eb;
  color: #67c23a;
}
.result.error {
  background: #fef0f0;
  color: #f56c6c;
}
</style>
