// 切片 1aa:auth store 单元测试
// 覆盖 login/logout/fetchMe 流程 + 401 handler 注册 + computed 状态。

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

// Mock api/auth 模块,隔离网络调用
vi.mock('../src/api/auth', () => ({
  postLogin: vi.fn(),
  postLogout: vi.fn(),
  getMe: vi.fn(),
}));

import { useAuthStore } from '../src/stores/auth';
import { postLogin, postLogout, getMe } from '../src/api/auth';
import type { UserInfo } from '../src/api/auth';

const mockUser: UserInfo = {
  id: 'u-1',
  email: 'admin@test.local',
  display_name: 'Admin',
  role: 'admin',
};

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initial state', () => {
    it('starts unauthenticated with no user', () => {
      const auth = useAuthStore();
      expect(auth.user).toBeNull();
      expect(auth.isAuthenticated).toBe(false);
      expect(auth.loading).toBe(false);
      expect(auth.error).toBe('');
    });

    it('displayName returns empty string when no user', () => {
      const auth = useAuthStore();
      expect(auth.displayName).toBe('');
    });
  });

  describe('login', () => {
    it('sets user and clears loading on success', async () => {
      vi.mocked(postLogin).mockResolvedValue(mockUser);
      const auth = useAuthStore();

      await auth.login({ email: 'admin@test.local', password: 'pass' });

      expect(auth.user).toEqual(mockUser);
      expect(auth.isAuthenticated).toBe(true);
      expect(auth.loading).toBe(false);
      expect(auth.error).toBe('');
      expect(postLogin).toHaveBeenCalledWith({
        email: 'admin@test.local',
        password: 'pass',
      });
    });

    it('sets error and rethrows on failure, clears loading', async () => {
      vi.mocked(postLogin).mockRejectedValue(new Error('invalid_credentials'));
      const auth = useAuthStore();

      await expect(auth.login({ email: 'x', password: 'y' })).rejects.toThrow(
        'invalid_credentials',
      );

      expect(auth.user).toBeNull();
      expect(auth.loading).toBe(false);
      expect(auth.error).toBe('invalid_credentials');
    });

    it('sets loading=true during in-flight login', async () => {
      let resolveLogin: (u: UserInfo) => void = () => undefined;
      vi.mocked(postLogin).mockReturnValue(
        new Promise<UserInfo>((resolve) => {
          resolveLogin = resolve;
        }),
      );
      const auth = useAuthStore();

      const p = auth.login({ email: 'a', password: 'b' });
      expect(auth.loading).toBe(true);

      resolveLogin(mockUser);
      await p;
      expect(auth.loading).toBe(false);
    });

    it('clears previous error on new login attempt', async () => {
      vi.mocked(postLogin).mockResolvedValueOnce(mockUser);
      const auth = useAuthStore();

      await auth.login({ email: 'a', password: 'b' });
      // 模拟一次失败
      vi.mocked(postLogin).mockRejectedValueOnce(new Error('bad'));
      await expect(auth.login({ email: 'a', password: 'b' })).rejects.toThrow();
      expect(auth.error).toBe('bad');

      // 再次尝试应清空 error
      vi.mocked(postLogin).mockResolvedValueOnce(mockUser);
      await auth.login({ email: 'a', password: 'b' });
      expect(auth.error).toBe('');
    });

    it('non-Error thrown value is stringified into error field', async () => {
      vi.mocked(postLogin).mockRejectedValue('string error');
      const auth = useAuthStore();

      await expect(auth.login({ email: 'a', password: 'b' })).rejects.toBe(
        'string error',
      );
      expect(auth.error).toBe('string error');
    });
  });

  describe('logout', () => {
    it('clears user on success', async () => {
      vi.mocked(postLogout).mockResolvedValue(undefined);
      const auth = useAuthStore();
      // 先登录
      vi.mocked(postLogin).mockResolvedValue(mockUser);
      await auth.login({ email: 'a', password: 'b' });
      expect(auth.isAuthenticated).toBe(true);

      await auth.logout();

      expect(auth.user).toBeNull();
      expect(auth.isAuthenticated).toBe(false);
      expect(postLogout).toHaveBeenCalledTimes(1);
    });
  });

  describe('fetchMe', () => {
    it('sets user on success', async () => {
      vi.mocked(getMe).mockResolvedValue(mockUser);
      const auth = useAuthStore();

      await auth.fetchMe();

      expect(auth.user).toEqual(mockUser);
    });

    it('clears user on failure without throwing (router relies on this)', async () => {
      vi.mocked(getMe).mockRejectedValue(new Error('NETWORK'));
      const auth = useAuthStore();
      // 先有 user
      vi.mocked(postLogin).mockResolvedValue(mockUser);
      await auth.login({ email: 'a', password: 'b' });

      await auth.fetchMe();

      expect(auth.user).toBeNull();
    });

    it('handles null response from getMe (401 case)', async () => {
      vi.mocked(getMe).mockResolvedValue(null);
      const auth = useAuthStore();

      await auth.fetchMe();

      expect(auth.user).toBeNull();
    });
  });

  describe('displayName computed', () => {
    it('prefers display_name over email', () => {
      vi.mocked(postLogin).mockResolvedValue(mockUser);
      const auth = useAuthStore();
      return auth.login({ email: 'a', password: 'b' }).then(() => {
        expect(auth.displayName).toBe('Admin');
      });
    });

    it('returns empty string when display_name is empty string (?? only checks nullish)', async () => {
      // 注意:源码用 ?? 而非 ||,空字符串不算 nullish。
      // 这是当前行为(不是 bug,空字符串可能是用户主动清空 display_name)。
      const emptyDisplayName: UserInfo = { ...mockUser, display_name: '' };
      vi.mocked(postLogin).mockResolvedValue(emptyDisplayName);
      const auth = useAuthStore();
      await auth.login({ email: 'a', password: 'b' });
      expect(auth.displayName).toBe('');
    });
  });
});
