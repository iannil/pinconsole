// 1h-ui:认证 REST 客户端
// 对应后端 server/internal/api/auth.go
//
// 1z P1-1:改用 apiFetch/apiJson 自动注入 X-Trace-Id 头,
// admin SPA → server 端 trace_id 端到端传播。

import { apiFetch, apiJson } from './client';

export interface UserInfo {
  id: string;
  email: string;
  display_name: string;
  role: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export async function postLogin(req: LoginRequest): Promise<UserInfo> {
  const { data } = await apiJson<UserInfo>('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  return data;
}

export async function postLogout(): Promise<void> {
  // logout 即使失败也不抛,保持原语义
  await apiFetch('/api/auth/logout', { method: 'POST' });
}

export async function getMe(): Promise<UserInfo | null> {
  // 401 是合法响应(未登录),特殊处理:不走 apiJson(会抛)
  const { response } = await apiFetch<UserInfo>('/api/auth/me');
  if (response.status === 401) return null;
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
}
