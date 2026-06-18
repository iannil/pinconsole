// 1h-ui:认证 REST 客户端
// 对应后端 server/internal/api/auth.go

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
  const resp = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
    credentials: 'include',
  });
  if (!resp.ok) {
    const data = await resp.json().catch(() => ({}));
    throw new Error(data.error || `HTTP ${resp.status}`);
  }
  return resp.json();
}

export async function postLogout(): Promise<void> {
  await fetch('/api/auth/logout', {
    method: 'POST',
    credentials: 'include',
  });
}

export async function getMe(): Promise<UserInfo | null> {
  const resp = await fetch('/api/auth/me', {
    credentials: 'include',
  });
  if (resp.status === 401) return null;
  if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
  return resp.json();
}
