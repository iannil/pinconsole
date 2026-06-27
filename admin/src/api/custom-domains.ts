// cd-1 API:自定义域名
import { apiJson } from './client';

export interface CustomDomain {
  id: number;
  tenant_id: string;
  domain: string;
  cert_status: 'pending' | 'active' | 'failed';
  cert_error: string;
  created_at: string;
  updated_at: string;
}

export async function fetchCustomDomains(): Promise<CustomDomain[]> {
  return apiJson<CustomDomain[]>('/api/custom-domains');
}

export async function createCustomDomain(domain: string): Promise<CustomDomain> {
  return apiJson<CustomDomain>('/api/custom-domains', {
    method: 'POST',
    body: JSON.stringify({ domain }),
  });
}

export async function deleteCustomDomain(id: number): Promise<void> {
  await apiJson<void>(`/api/custom-domains/${id}`, { method: 'DELETE' });
}
