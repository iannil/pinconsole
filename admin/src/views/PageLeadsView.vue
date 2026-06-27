<script setup lang="ts">
// pe-3: PageLeadsView — 表单提交记录
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { PhArrowLeft } from '@phosphor-icons/vue';
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
        <button class="pc-btn pc-btn--ghost" @click="goBack"><PhArrowLeft :size="18" /> {{ t('page_leads.back') }}</button>
        <h1>{{ t('page_leads.title') }} — {{ props.slug }}</h1>
      </div>
    </header>

    <div v-if="loading" class="loading">{{ t('widgets.loading') }}</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="leads.length === 0" class="empty-state">
      <p>{{ t('page_leads.no_leads') }}</p>
    </div>

    <div v-else class="leads-list">
      <div v-for="lead in leads" :key="lead.id" class="pc-card lead-card">
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
.leads-view { padding: var(--pc-space-section); max-width: 720px; margin: 0 auto; }
.view-header { margin-bottom: var(--pc-space-section); }
.header-left { display: flex; align-items: center; gap: var(--pc-space-component); }
.header-left h1 { font-size: var(--pc-text-lg); font-weight: var(--pc-weight-semibold); margin: 0; }
.loading, .empty-state, .error { padding: 48px; text-align: center; color: var(--pc-color-text-muted); }
.error { color: var(--pc-color-danger); }
.leads-list { display: flex; flex-direction: column; gap: var(--pc-space-component); }
.lead-card { padding: var(--pc-space-card); }
.lead-meta { font-size: var(--pc-text-xs); color: var(--pc-color-text-muted); margin-bottom: var(--pc-space-field); }
.leads-table { width: 100%; border-collapse: collapse; }
.leads-table td { padding: 4px 0; font-size: var(--pc-text-base); }
.field-key { font-weight: var(--pc-weight-medium); color: var(--pc-color-text-secondary); width: 120px; vertical-align: top; }
.field-value { color: var(--pc-color-text-primary); word-break: break-all; }
</style>
