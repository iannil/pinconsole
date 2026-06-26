<script setup lang="ts">
// pe-2: PagesView — 落地页列表
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
import { PhPlus, PhTrash, PhPencilSimple, PhEye, PhListDashes } from '@phosphor-icons/vue';
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
      <button class="btn btn-primary" @click="showNewDialog = true">
        <PhPlus :size="16" weight="bold" /> {{ t('pages.new_page') }}
      </button>
    </header>

    <!-- 新建对话框 -->
    <div v-if="showNewDialog" class="dialog-overlay" @click.self="showNewDialog = false">
      <div class="dialog-card">
        <h3>{{ t('pages.new_page') }}</h3>
        <input
          v-model="newTitle"
          :placeholder="t('pages.new_page')"
          class="input"
          @keyup.enter="handleCreate"
          autofocus
        />
        <div class="dialog-actions">
          <button class="btn btn-text" @click="showNewDialog = false">{{ t('app.hello')?.split(' ')[0] || 'Cancel' }}</button>
          <button class="btn btn-primary" @click="handleCreate" :disabled="!newTitle.trim()">{{ t('pages.edit') }}</button>
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
            <span :class="['badge', page.status === 'published' ? 'badge-success' : 'badge-default']">
              {{ t(`pages.${page.status}`) }}
            </span>
          </td>
          <td class="date-cell">{{ new Date(page.updated_at).toLocaleString() }}</td>
          <td class="actions-cell">
            <button class="btn-icon" @click="handleEdit(page.slug)" :title="t('pages.edit')">
              <PhPencilSimple :size="16" />
            </button>
            <button class="btn-icon" @click="handlePreview(page.slug)" :title="t('pages.preview')">
              <PhEye :size="16" />
            </button>
            <button class="btn-icon" @click="handleLeads(page.slug)" title="Leads">
              <PhListDashes :size="16" />
            </button>
            <button class="btn-icon btn-icon-danger" @click="togglePublish(page.slug, page.status)" :title="page.status === 'published' ? t('pages.unpublish') : t('pages.publish')">
              {{ page.status === 'published' ? t('pages.unpublish') : t('pages.publish') }}
            </button>
            <button class="btn-icon btn-icon-danger" @click="handleDelete(page.slug)" :title="t('pages.delete')">
              <PhTrash :size="16" />
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.pages-view { padding: 24px; max-width: 960px; margin: 0 auto; }
.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
.view-header h1 { font-size: 1.25rem; font-weight: 600; margin: 0; }
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 16px; border-radius: 8px; font-size: 14px; font-weight: 500; cursor: pointer; border: none; font-family: inherit; }
.btn-primary { background: var(--color-accent-default, #0f766e); color: #fff; }
.btn-primary:hover { opacity: .9; }
.btn-primary:disabled { opacity: .5; cursor: not-allowed; }
.btn-text { background: transparent; color: var(--color-text-secondary, #57534e); }
.btn-text:hover { background: var(--color-bg-subtle, #f5f1ec); }
.btn-icon { display: inline-flex; align-items: center; gap: 4px; padding: 4px 8px; border: none; background: transparent; cursor: pointer; border-radius: 6px; color: var(--color-text-secondary, #57534e); font-size: 13px; font-family: inherit; }
.btn-icon:hover { background: var(--color-bg-subtle, #f5f1ec); }
.btn-icon-danger:hover { color: #dc2626; }
.dialog-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,.3); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.dialog-card { background: #fff; border-radius: 12px; padding: 24px; width: 400px; max-width: 90vw; box-shadow: 0 8px 24px rgba(0,0,0,.12); }
.dialog-card h3 { margin: 0 0 16px; font-size: 1rem; }
.dialog-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
.input { width: 100%; padding: 8px 12px; border: 1px solid var(--color-border-default, #e7e5e4); border-radius: 8px; font-size: 14px; font-family: inherit; }
.input:focus { outline: none; border-color: var(--color-accent-default, #0f766e); }
.loading { padding: 48px; text-align: center; color: var(--color-text-muted, #78716c); }
.empty-state { padding: 64px 24px; text-align: center; color: var(--color-text-muted, #78716c); }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th { text-align: left; padding: 8px 12px; font-size: 12px; font-weight: 600; text-transform: uppercase; letter-spacing: .5px; color: var(--color-text-muted, #78716c); border-bottom: 1px solid var(--color-border-default, #e7e5e4); }
.data-table td { padding: 10px 12px; border-bottom: 1px solid var(--color-border-default, #e7e5e4); font-size: 14px; }
.slug-cell { font-weight: 500; }
.date-cell { color: var(--color-text-secondary, #57534e); font-size: 13px; }
.actions-cell { display: flex; gap: 4px; justify-content: flex-end; }
.actions-th { text-align: right; }
.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 12px; font-weight: 500; }
.badge-success { background: #d1fae5; color: #065f46; }
.badge-default { background: #f5f5f4; color: #44403c; }
</style>
