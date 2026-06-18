import { test, expect } from '@playwright/test';
import { ADMIN_EMAIL, ADMIN_PASSWORD } from '../fixtures/admin-auth';

// 切片 1h-ui:e2e 验收
// 前置:server 在 :8080,admin SPA 在 /admin/
// 1k fail-secure 后 release binary 不再有 dev bypass,login endpoint 永远验密码;
// 凭证从根 .env 读(与 server 启动用同一份)。

test.describe('1h-ui', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });

  test('1h-ui 场景1:未登录访问 /dashboard 重定向到 /login', async ({ page }) => {
    // 用独立 context 确保 cookie 干净
    await page.goto('/admin/dashboard');
    await page.waitForURL(/\/admin\/login/);
    expect(page.url()).toContain('/admin/login');
    // redirect 参数应保留
    expect(page.url()).toContain('redirect=');
  });

  test('1h-ui 场景2:输入正确凭证 → 登录成功 → 跳回 dashboard', async ({ page }) => {
    await page.goto('/admin/login');
    await page.fill('input[type="email"]', ADMIN_EMAIL);
    await page.fill('input[type="password"]', ADMIN_PASSWORD);
    await page.click('button[type="submit"]');

    // 登录成功后 LoginView 跳 /dashboard
    await page.waitForURL(/\/admin\/dashboard/, { timeout: 10_000 });
    expect(page.url()).toContain('/admin/dashboard');
  });

  test('1h-ui 场景3:错误密码 → 显示错误提示', async ({ page }) => {
    await page.goto('/admin/login');
    // 用不存在的邮箱,避免污染真实 admin 的 throttle counter(1x 限流 5 次/15min)
    await page.fill('input[type="email"]', 'no-such-user@test.local');
    await page.fill('input[type="password"]', 'wrong-password-xyz');
    await page.click('button[type="submit"]');

    // 应停留在 login 并显示错误
    await page.waitForTimeout(1500);
    expect(page.url()).toContain('/admin/login');
    // 错误文案来自 i18n login.error_credentials
    await expect(page.locator('.error')).toBeVisible({ timeout: 5000 });
  });
});
