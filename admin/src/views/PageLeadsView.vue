<script setup lang="ts">
// pe-3: PageLeadsView — 表单提交记录
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { PhArrowLeft, PhEye } from '@phosphor-icons/vue';
import type { FormSubmission } from '@pinconsole/proto';
import { fetchPageLeads } from '../api/pages';

const props = defineProps<{ slug: string }>();
const { t } = useI18n();
const router = useRouter();

const leads = ref<FormSubmission[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    leads.value = await fetchPageLeads(props.slug);
  } catch (e: any) {
    error.value = e.message || 'load leads failed';
  } finally {
    loading.value = false;
  }
});

function goBack() {
  router.push('/pages');
}
</script>

<template>
  <div class="leads-view">
    <header class="view-header">
      <div class="header-left">
        <button class="btn-text" @click="goBack"><PhArrowLeft :size="18" /> {{ t('page_leads.back') }}</button>
        <h1>{{ t('page_leads.title') }} — {{ props.slug }}</h1>
      </div>
    </header>

    <div v-if="loading" class="loading">{{ t('widgets.loading') }}</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="leads.length === 0" class="empty-state">
      <p>{{ t('page_leads.no_leads') }}</p>
    </div>

    <div v-else class="leads-list">
      <div v-for="lead in leads" :key="lead.id" class="lead-card">
        <div class="lead-meta">{{ t('page_leads.submitted_at') }}: {{ new Date(lead.created_at).toLocaleString() }}</div>
        <table class="leads-table">
          <tr v-for="(value, key) in lead.fields" :key="key">
            <td class="field-key">{{ key }}</td>
            <td class="field-value">{{ value }}</td>
          </tr>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.leads-view { padding: 24px; max-width: 720px; margin: 0 auto; }
.view-header { margin-bottom: 24px; }
.header-left { display: flex; align-items: center; gap: 12px; }
.header-left h1 { font-size: 1.125rem; font-weight: 600; margin: 0; }
.btn-text { display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px; border: none; background: transparent; cursor: pointer; border-radius: 6px; font-size: 14px; font-family: inherit; color: var(--color-text-secondary, #57534e); }
.btn-text:hover { background: var(--color-bg-subtle, #f5f1ec); }
.loading, .empty-state, .error { padding: 48px; text-align: center; color: var(--color-text-muted, #78716c); }
.error { color: #dc2626; }
.leads-list { display: flex; flex-direction: column; gap: 12px; }
.lead-card { border: 1px solid var(--color-border-default, #e7e5e4); border-radius: 8px; padding: 16px; }
.lead-meta { font-size: 12px; color: var(--color-text-muted, #78716c); margin-bottom: 8px; }
.leads-table { width: 100%; border-collapse: collapse; }
.leads-table td { padding: 4px 0; font-size: 14px; }
.field-key { font-weight: 500; color: var(--color-text-secondary, #57534e); width: 120px; vertical-align: top; }
.field-value { color: var(--color-text-primary, #1c1917); word-break: break-all; }
</style>
