<script setup lang="ts">
// pe-2: PagesView — 落地页列表
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { PhPlus, PhTrash, PhPencilSimple, PhEye, PhListDashes, PhX } from '@phosphor-icons/vue';
import { usePagesStore } from '../stores/pages';

const { t } = useI18n();
const router = useRouter();
const store = usePagesStore();

const showNewDialog = ref(false);
const newTitle = ref('');

onMounted(() => {
  store.loadPages();
});

async function handleCreate() {
  if (!newTitle.value.trim()) return;
  try {
    const page = await store.createPage({ title: newTitle.value.trim() });
    showNewDialog.value = false;
    newTitle.value = '';
    router.push(`/pages/${page.slug}/edit`);
  } catch {
    // error handled by store
  }
}

function handleEdit(slug: string) {
  router.push(`/pages/${slug}/edit`);
}

async function handleDelete(slug: string) {
  if (!confirm(t('pages.delete_confirm'))) return;
  await store.removePage(slug);
}

async function togglePublish(slug: string, current: string) {
  const next = current === 'published' ? 'draft' : 'published';
  await store.publish(slug, next);
}

function handlePreview(slug: string) {
  window.open(`/p/${slug}`, '_blank');
}

function handleLeads(slug: string) {
  router.push(`/pages/${slug}/leads`);
}
</script>

<template>
  <div class="pages-view">
    <header class="view-header">
      <h1>{{ t('pages.title') }}</h1>
      <button class="pc-btn pc-btn--primary" @click="showNewDialog = true">
        <PhPlus :size="16" weight="bold" /> {{ t('pages.new_page') }}
      </button>
    </header>

    <!-- 新建对话框 -->
    <div v-if="showNewDialog" class="dialog-overlay" @click.self="showNewDialog = false">
      <div class="dialog-card">
        <div class="dialog-header">
          <h3>{{ t('pages.new_page') }}</h3>
          <button class="pc-btn pc-btn--ghost pc-btn--icon" @click="showNewDialog = false" :aria-label="t('pages.cancel')">
            <PhX :size="20" />
          </button>
        </div>
        <input
          v-model="newTitle"
          :placeholder="t('pages.new_page')"
          class="pc-input"
          @keyup.enter="handleCreate"
          autofocus
        />
        <div class="dialog-actions">
          <button class="pc-btn pc-btn--ghost" @click="showNewDialog = false">{{ t('pages.cancel') }}</button>
          <button class="pc-btn pc-btn--primary" @click="handleCreate" :disabled="!newTitle.trim()">{{ t('pages.edit') }}</button>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="store.loading" class="loading">{{ t('widgets.loading') }}</div>

    <!-- 空状态 -->
    <div v-else-if="store.pages.length === 0" class="empty-state">
      <p>{{ t('pages.no_pages') }}</p>
    </div>

    <!-- 列表 -->
    <table v-else class="data-table">
      <thead>
        <tr>
          <th>{{ t('pages.slug') }}</th>
          <th>{{ t('pages.status') }}</th>
          <th>{{ t('pages.updated') }}</th>
          <th class="actions-th">{{ t('pages.edit') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="page in store.pages" :key="page.slug">
          <td class="slug-cell">{{ page.slug }}</td>
          <td>
            <span :class="page.status === 'published' ? 'pc-badge pc-badge--success' : 'pc-badge pc-badge--accent'">
              {{ t(`pages.${page.status}`) }}
            </span>
          </td>
          <td class="date-cell">{{ new Date(page.updated_at).toLocaleString() }}</td>
          <td class="actions-cell">
            <button class="pc-btn pc-btn--ghost pc-btn--icon" @click="handleEdit(page.slug)" :title="t('pages.edit')">
              <PhPencilSimple :size="16" />
            </button>
            <button class="pc-btn pc-btn--ghost pc-btn--icon" @click="handlePreview(page.slug)" :title="t('pages.preview')">
              <PhEye :size="16" />
            </button>
            <button class="pc-btn pc-btn--ghost pc-btn--icon" @click="handleLeads(page.slug)" :title="t('page_leads.title')">
              <PhListDashes :size="16" />
            </button>
            <button
              class="pc-btn pc-btn--ghost"
              @click="togglePublish(page.slug, page.status)"
              :title="page.status === 'published' ? t('pages.unpublish') : t('pages.publish')"
            >
              {{ page.status === 'published' ? t('pages.unpublish') : t('pages.publish') }}
            </button>
            <button class="pc-btn pc-btn--ghost pc-btn--icon" @click="handleDelete(page.slug)" :title="t('pages.delete')">
              <PhTrash :size="16" />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.pages-view { padding: var(--pc-space-page); max-width: 960px; margin: 0 auto; }
.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--pc-space-section); }
.view-header h1 { font-size: var(--pc-text-xl); font-weight: var(--pc-weight-semibold); margin: 0; }
.dialog-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(28,25,23,0.4); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.dialog-card { background: var(--pc-color-bg-surface); border-radius: var(--pc-radius-xl); padding: var(--pc-space-section); width: 400px; max-width: 90vw; box-shadow: var(--pc-shadow-lg); }
.dialog-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--pc-space-component); }
.dialog-header h3 { margin: 0; font-size: var(--pc-text-md); }
.dialog-actions { display: flex; justify-content: flex-end; gap: var(--pc-space-field); margin-top: var(--pc-space-component); }
.loading { padding: 48px; text-align: center; color: var(--pc-color-text-muted); }
.empty-state { padding: 64px 24px; text-align: center; color: var(--pc-color-text-muted); }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th { text-align: left; padding: 8px var(--pc-space-component); font-size: var(--pc-text-xs); font-weight: var(--pc-weight-semibold); text-transform: uppercase; letter-spacing: .5px; color: var(--pc-color-text-muted); border-bottom: 1px solid var(--pc-color-border-default); }
.data-table td { padding: 10px var(--pc-space-component); border-bottom: 1px solid var(--pc-color-border-default); font-size: var(--pc-text-base); }
.slug-cell { font-weight: var(--pc-weight-medium); }
.date-cell { color: var(--pc-color-text-secondary); font-size: var(--pc-text-sm); }
.actions-cell { display: flex; gap: 4px; justify-content: flex-end; align-items: center; }
.actions-th { text-align: right; }
</style>
