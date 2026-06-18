// 切片 1x:login throttle regression + smoke
//
// 1x P1-3:Redis-based login brute-force throttle.
// server/internal/api/auth.go:
//   loginMaxAttempts   = 5
//   checkLoginThrottle():count >= 5 → locked → 429
//   recordLoginFailure():INCR count,首次失败时设 TTL
//
// regression(直连 seed counter 到 5):
//   seed counter=5 → 1 次错密码 → 验证返回 429 + Retry-After 头
//
// smoke(真实 6 次错密码):
//   连续 6 次错误密码 → 前 5 次返回 401(counter 涨) → 第 6 次返回 429(锁定)
//   验证后续正确密码也被拒
//   然后清理 Redis key,避免污染后续测试
//
// 注意:用专属 email `e2e-1x@test.local` 避免污染真实 admin 的 throttle 计数器。

import { test, expect } from '@playwright/test';
import { setLoginThrottleCounter, clearLoginThrottle, closeDBFixtures } from '../fixtures/db';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const ENV = readFileSync(resolve(PROJECT_ROOT, '.env'), 'utf-8');
const ADMIN_EMAIL_FOR_TEST =
  ENV.match(/^ADMIN_EMAIL\s*=\s*(.+?)\s*$/m)?.[1]?.replace(/^["']|["']$/g, '') ??
  'admin@marketing-monitor.local';
const ADMIN_PASSWORD_FOR_TEST =
  ENV.match(/^ADMIN_PASSWORD\s*=\s*(.+?)\s*$/m)?.[1]?.replace(/^["']|["']$/g, '') ?? '';

const TEST_EMAIL = 'e2e-1x-throttle@test.local';
const CLIENT_IP = '::1';

test.describe('1x: login throttle', () => {
  test.afterAll(async () => {
    await clearLoginThrottle(TEST_EMAIL, CLIENT_IP);
    await clearLoginThrottle(ADMIN_EMAIL_FOR_TEST, CLIENT_IP);
    await closeDBFixtures();
  });

  test('regression:counter=5 时下次登录直接 429 + Retry-After', async ({ request }) => {
    await clearLoginThrottle(TEST_EMAIL, CLIENT_IP);
    // seed 到阈值 = 已锁定
    await setLoginThrottleCounter(TEST_EMAIL, CLIENT_IP, 5);

    const resp = await request.post('/api/auth/login', {
      data: { email: TEST_EMAIL, password: 'wrong-password' },
    });
    expect(resp.status(), 'when counter=5, login should return 429 (locked)').toBe(429);
    const body = await resp.json();
    expect(body.error).toBe('too_many_attempts');
    expect(body.retry_after).toBeGreaterThan(0);
    const retryAfter = resp.headers()['retry-after'];
    expect(retryAfter, 'Retry-After header should be set').toBeTruthy();

    await clearLoginThrottle(TEST_EMAIL, CLIENT_IP);
  });

  test('regression:counter=4 时下次登录正常返回 401(counter 涨到 5)', async ({ request }) => {
    await clearLoginThrottle(TEST_EMAIL, CLIENT_IP);
    await setLoginThrottleCounter(TEST_EMAIL, CLIENT_IP, 4);

    // counter < 5,未锁定,正常走 auth → 401(密码错)
    const resp = await request.post('/api/auth/login', {
      data: { email: TEST_EMAIL, password: 'wrong-password' },
    });
    expect(resp.status(), 'when counter=4, login should return 401 (still under threshold)').toBe(401);
    const body = await resp.json();
    expect(body.error).toBe('invalid_credentials');

    await clearLoginThrottle(TEST_EMAIL, CLIENT_IP);
  });

  test('smoke:6 次连续错密码 → 第 6 次触发 429 + 后续被锁', async ({ request }) => {
    await clearLoginThrottle(TEST_EMAIL, CLIENT_IP);

    // 前 5 次:每次 counter < 5,401 + INCR
    for (let i = 1; i <= 5; i++) {
      const r = await request.post('/api/auth/login', {
        data: { email: TEST_EMAIL, password: 'wrong-pwd' },
      });
      expect(r.status(), `attempt ${i} should be 401`).toBe(401);
    }

    // 第 6 次:counter=5 >= 5,锁定,429
    const r6 = await request.post('/api/auth/login', {
      data: { email: TEST_EMAIL, password: 'wrong-pwd' },
    });
    expect(r6.status(), '6th attempt should be 429').toBe(429);
    const body6 = await r6.json();
    expect(body6.error).toBe('too_many_attempts');

    // 即使正确密码,锁定期间也拒
    const r7 = await request.post('/api/auth/login', {
      data: { email: TEST_EMAIL, password: ADMIN_PASSWORD_FOR_TEST },
    });
    expect(r7.status(), 'even with correct password, locked account returns 429').toBe(429);

    await clearLoginThrottle(TEST_EMAIL, CLIENT_IP);
  });

  test('regression:成功登录清零计数器', async ({ request }) => {
    await clearLoginThrottle(ADMIN_EMAIL_FOR_TEST, CLIENT_IP);
    await setLoginThrottleCounter(ADMIN_EMAIL_FOR_TEST, CLIENT_IP, 3);

    // 真实 admin 凭证登录(成功)— server 清掉 counter
    const resp = await request.post('/api/auth/login', {
      data: { email: ADMIN_EMAIL_FOR_TEST, password: ADMIN_PASSWORD_FOR_TEST },
    });
    expect(resp.ok(), 'admin login should succeed').toBe(true);

    // 再错一次 — 应是 401(counter 已被清零,这次是第 1 次失败)
    const resp2 = await request.post('/api/auth/login', {
      data: { email: ADMIN_EMAIL_FOR_TEST, password: 'wrong' },
    });
    expect(resp2.status(), 'after success+1fail, should be 401 not 429').toBe(401);

    await clearLoginThrottle(ADMIN_EMAIL_FOR_TEST, CLIENT_IP);
  });
});
