// Claim/release API:1:1 锁定一个 session 给当前 admin。
// POST /api/sessions/:id/claim   — 拿锁(其他 admin 不能 claim)
// POST /api/sessions/:id/release — 放锁
// GET  /api/sessions/:id/claim   — 查询当前 claim 状态(谁锁的)

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

export async function getClaimState(sessionId: string): Promise<ClaimState> {
  const { data } = await apiJson<ClaimState>(
    `/api/sessions/${encodeURIComponent(sessionId)}/claim`,
  );
  return data;
}
