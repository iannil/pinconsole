// Claim/release API:1:1 锁定一个 session 给当前 admin。
// POST /api/sessions/:id/claim           — 拿锁(其他 admin 不能 claim)
// POST /api/sessions/:id/release         — 放锁
// POST /api/sessions/:id/claim/refresh   — 续 TTL(owner-only,P1-claim-TTL 修复)
// GET  /api/sessions/:id/claim           — 查询当前 claim 状态(谁锁的)

import { apiJson } from './client';

export interface ClaimState {
  claimed: boolean;
  claimed_by?: string;
}

export async function claimSession(sessionId: string): Promise<ClaimState> {
  const { data } = await apiJson<ClaimState>(
    `/api/sessions/${encodeURIComponent(sessionId)}/claim`,
    { method: 'POST' },
  );
  return data;
}

export async function releaseSession(sessionId: string): Promise<{ ok: boolean }> {
  const { data } = await apiJson<{ ok: boolean }>(
    `/api/sessions/${encodeURIComponent(sessionId)}/release`,
    { method: 'POST' },
  );
  return data;
}

// refreshClaim 续 claim TTL(owner-only)。返回 false 表示 claim 已丢(TTL 过期 / 被他人 release)。
// Admin SPA 应在 co-browsing active 时每 60s 调本函数,避免 5min 自然过期。
export async function refreshClaim(sessionId: string): Promise<boolean> {
  try {
    await apiJson<{ ok: boolean }>(
      `/api/sessions/${encodeURIComponent(sessionId)}/claim/refresh`,
      { method: 'POST' },
    );
    return true;
  } catch (err: unknown) {
    // 403 = not_claim_owner(claim 已丢);其他错误(network/5xx)不计为丢 claim
    const status = (err as { response?: { status?: number } })?.response?.status;
    if (status === 403) return false;
    // 网络错误等不致命,返回 true 避免误触发 claim-lost UI
    return true;
  }
}

export async function getClaimState(sessionId: string): Promise<ClaimState> {
  const { data } = await apiJson<ClaimState>(
    `/api/sessions/${encodeURIComponent(sessionId)}/claim`,
  );
  return data;
}
