<template>
  <div class="domains-page">
    <h1>{{ t('domains.title') }}</h1>
    <p class="subtitle">{{ t('domains.subtitle') }}</p>

    <div v-if="loading" class="loading">{{ t('domains.loading') }}</div>

    <!-- 添加域名 -->
    <section class="pc-card add-domain">
      <h2>{{ t('domains.add_title') }}</h2>
      <div class="input-row">
        <input
          v-model="newDomain"
          type="text"
          class="pc-input"
          :placeholder="t('domains.domain_placeholder')"
          @keyup.enter="addDomain"
        />
        <button class="pc-btn pc-btn--primary" :disabled="adding" @click="addDomain">
          {{ adding ? t('domains.adding') : t('domains.add_btn') }}
        </button>
      </div>
      <p v-if="error" class="error-msg">{{ error }}</p>
    </section>

    <!-- 域名列表 -->
    <section class="pc-card">
      <h2>{{ t('domains.list_title') }}</h2>
      <table v-if="domains.length > 0" class="domains-table">
        <thead>
          <tr>
            <th>{{ t('domains.col_domain') }}</th>
            <th>{{ t('domains.col_status') }}</th>
            <th>{{ t('domains.col_created') }}</th>
            <th>{{ t('domains.col_actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="d in domains" :key="d.id">
            <td class="domain-cell">{{ d.domain }}</td>
            <td>
              <StatusBadge
                :variant="statusVariant(d.cert_status)"
                :dot="d.cert_status === 'active'"
              >
                {{ t(`domains.status_${d.cert_status}`) }}
              </StatusBadge>
            </td>
            <td class="date-cell">{{ formatDate(d.created_at) }}</td>
            <td>
              <button class="pc-btn pc-btn--danger" @click="removeDomain(d.id, d.domain)">
                {{ t('domains.delete') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="empty">{{ t('domains.empty') }}</p>
    </section>

    <!-- DNS 指引 -->
    <section class="pc-card">
      <h2>{{ t('domains.dns_title') }}</h2>
      <ol>
        <li v-for="i in 3" :key="i">{{ t(`domains.dns_step_${i}`) }}</li>
      </ol>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import StatusBadge from '../components/StatusBadge.vue';
import { fetchCustomDomains, createCustomDomain, deleteCustomDomain, type CustomDomain } from '../api/custom-domains';

const { t } = useI18n();

const domains = ref<CustomDomain[]>([]);
const loading = ref(true);
const newDomain = ref('');
const adding = ref(false);
const error = ref('');

function statusVariant(s: string): 'success' | 'warning' | 'neutral' {
  if (s === 'active') return 'success';
  if (s === 'pending') return 'warning';
  return 'neutral';
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

async function loadDomains() {
  loading.value = true;
  try {
    domains.value = await fetchCustomDomains();
  } catch {
    domains.value = [];
  } finally {
    loading.value = false;
  }
}

async function addDomain() {
  const domain = newDomain.value.trim().toLowerCase();
  if (!domain) return;
  adding.value = true;
  error.value = '';
  try {
    await createCustomDomain(domain);
    newDomain.value = '';
    await loadDomains();
  } catch (e: any) {
    error.value = e.message || t('domains.error_add');
  } finally {
    adding.value = false;
  }
}

async function removeDomain(id: number, _domain: string) {
  if (!confirm(t('domains.confirm_delete', { domain: _domain }))) return;
  try {
    await deleteCustomDomain(id);
    await loadDomains();
  } catch {
    error.value = t('domains.error_delete');
  }
}

onMounted(loadDomains);
</script>

<style scoped>
.domains-page {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--pc-space-section);
}
.subtitle {
  color: var(--pc-color-text-secondary);
  margin-bottom: var(--pc-space-section);
}
.input-row {
  display: flex;
  gap: var(--pc-space-field);
}
.domains-table {
  width: 100%;
  border-collapse: collapse;
}
.domains-table th,
.domains-table td {
  text-align: left;
  padding: 8px 4px;
  border-bottom: 1px solid var(--pc-color-border-default);
}
.domains-table th {
  font-size: var(--pc-text-xs);
  font-weight: var(--pc-weight-semibold);
  color: var(--pc-color-text-muted);
  text-transform: uppercase;
  letter-spacing: .5px;
}
.domain-cell {
  font-family: var(--pc-font-mono);
  font-size: var(--pc-text-sm);
}
.date-cell {
  color: var(--pc-color-text-secondary);
  font-size: var(--pc-text-xs);
}
.empty {
  color: var(--pc-color-text-secondary);
  text-align: center;
  padding: var(--pc-space-section);
}
.error-msg {
  color: var(--pc-color-danger);
  font-size: var(--pc-text-sm);
  margin-top: var(--pc-space-field);
}
.pc-card ol {
  padding-left: 20px;
  line-height: 1.8;
}
.loading {
  text-align: center;
  padding: 40px;
  color: var(--pc-color-text-secondary);
}
</style>
