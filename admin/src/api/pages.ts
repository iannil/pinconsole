// pe-2: Page 编辑器 API 客户端
import { apiFetch, apiJson } from './client';
import type { PageResponse, PageListItem, CreatePageRequest, UpdatePageRequest, PageSchema, FormSubmission } from '@pinconsole/proto';

/** 获取页面列表 */
export async function fetchPages(): Promise<PageListItem[]> {
  const { data } = await apiJson<PageListItem[]>('/api/pages');
  return data;
}

/** 创建页面 */
export async function createPage(req: CreatePageRequest): Promise<PageResponse> {
  const { data } = await apiJson<PageResponse>('/api/pages', { method: 'POST', body: JSON.stringify(req) });
  return data;
}

/** 获取单个页面详情 */
export async function fetchPage(slug: string): Promise<PageResponse> {
  const { data } = await apiJson<PageResponse>(`/api/pages/${slug}`);
  return data;
}

/** 更新页面（schema/title/status） */
export async function updatePage(slug: string, req: UpdatePageRequest): Promise<PageResponse> {
  const { data } = await apiJson<PageResponse>(`/api/pages/${slug}`, { method: 'PUT', body: JSON.stringify(req) });
  return data;
}

/** 删除页面 */
export async function deletePage(slug: string): Promise<void> {
  await apiFetch(`/api/pages/${slug}`, { method: 'DELETE' });
}

/** 发布/取消发布 */
export async function publishPage(slug: string, status: 'draft' | 'published'): Promise<PageResponse> {
  const { data } = await apiJson<PageResponse>(`/api/pages/${slug}/publish`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  });
  return data;
}

/** 获取表单提交记录 */
export async function fetchPageLeads(slug: string): Promise<FormSubmission[]> {
  const { data } = await apiJson<FormSubmission[]>(`/api/pages/${slug}/leads`);
  return data;
}

/** 默认空 schema */
export function emptyPageSchema(): PageSchema {
  return {
    meta: { title: '' },
    style: { primary_color: '#0f766e', background: '#ffffff' },
    sections: [],
  };
}
