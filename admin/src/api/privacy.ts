// 1l-privacy-gdpr:REST 客户端
// 管理员代访客调用 GDPR Art.17 被遗忘权。
//
// 1z P1-1:改用 apiJson 自动注入 X-Trace-Id 头。

import { apiJson } from './client';

export interface ErasureResponse {
  ok: boolean;
  fingerprint: string;
  deleted_sessions: number;
  deleted_minio_objects: number;
  note?: string;
}

export async function deleteVisitorByFingerprint(fingerprint: string): Promise<ErasureResponse> {
  const { data } = await apiJson<ErasureResponse>(
    `/api/privacy/visitor/${encodeURIComponent(fingerprint)}`,
    { method: 'DELETE' },
  );
  return data;
}
