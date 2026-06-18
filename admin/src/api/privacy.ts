// 1l-privacy-gdpr:REST 客户端
// 管理员代访客调用 GDPR Art.17 被遗忘权。

export interface ErasureResponse {
  ok: boolean;
  fingerprint: string;
  deleted_sessions: number;
  deleted_minio_objects: number;
  note?: string;
}

export async function deleteVisitorByFingerprint(fingerprint: string): Promise<ErasureResponse> {
  const resp = await fetch(`/api/privacy/visitor/${encodeURIComponent(fingerprint)}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  if (!resp.ok) {
    const text = await resp.text().catch(() => '');
    throw new Error(`HTTP ${resp.status}: ${text}`);
  }
  return resp.json();
}
