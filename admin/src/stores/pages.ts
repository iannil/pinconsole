// pe-2: Page 编辑器 Pinia store
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { PageListItem, PageResponse, PageSchema, CreatePageRequest } from '@pinconsole/proto';
import * as api from '../api/pages';

export const usePagesStore = defineStore('pages', () => {
  // ── state ──
  const pages = ref<PageListItem[]>([]);
  const current = ref<PageResponse | null>(null);
  const loading = ref(false);
  const saving = ref(false);
  const error = ref<string | null>(null);

  // ── getters ──
  const draftCount = computed(() => pages.value.filter(p => p.status === 'draft').length);
  const publishedCount = computed(() => pages.value.filter(p => p.status === 'published').length);

  // ── actions ──
  async function loadPages() {
    loading.value = true;
    error.value = null;
    try {
      pages.value = await api.fetchPages();
    } catch (e: any) {
      error.value = e.message || 'load pages failed';
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function loadPage(slug: string) {
    loading.value = true;
    error.value = null;
    try {
      current.value = await api.fetchPage(slug);
    } catch (e: any) {
      error.value = e.message || 'load page failed';
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function createPage(data: CreatePageRequest): Promise<PageResponse> {
    error.value = null;
    try {
      const page = await api.createPage(data);
      pages.value.unshift({
        id: page.id,
        slug: page.slug,
        title: page.title,
        status: page.status,
        updated_at: page.updated_at,
      });
      return page;
    } catch (e: any) {
      error.value = e.message || 'create page failed';
      throw e;
    }
  }

  async function saveSchema(slug: string, schema: PageSchema) {
    saving.value = true;
    error.value = null;
    try {
      const updated = await api.updatePage(slug, { schema });
      current.value = updated;
      // 更新列表项
      const idx = pages.value.findIndex(p => p.slug === slug);
      if (idx >= 0) {
        pages.value[idx] = {
          id: updated.id,
          slug: updated.slug,
          title: updated.title,
          status: updated.status,
          updated_at: updated.updated_at,
        };
      }
    } catch (e: any) {
      error.value = e.message || 'save schema failed';
      throw e;
    } finally {
      saving.value = false;
    }
  }

  async function publish(slug: string, status: 'draft' | 'published') {
    error.value = null;
    try {
      const updated = await api.publishPage(slug, status);
      if (current.value && current.value.slug === slug) {
        current.value = updated;
      }
      const idx = pages.value.findIndex(p => p.slug === slug);
      if (idx >= 0) {
        pages.value[idx].status = status;
      }
    } catch (e: any) {
      error.value = e.message || 'publish failed';
      throw e;
    }
  }

  async function removePage(slug: string) {
    error.value = null;
    try {
      await api.deletePage(slug);
      pages.value = pages.value.filter(p => p.slug !== slug);
      if (current.value && current.value.slug === slug) {
        current.value = null;
      }
    } catch (e: any) {
      error.value = e.message || 'delete page failed';
      throw e;
    }
  }

  return {
    pages, current, loading, saving, error,
    draftCount, publishedCount,
    loadPages, loadPage, createPage, saveSchema, publish, removePage,
  };
});
