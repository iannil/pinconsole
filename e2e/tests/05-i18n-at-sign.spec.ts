// 切片 1z P0:i18n `@` SyntaxError regression
//
// 审计发现:vue-i18n v10 把 `@` 解析为 linked-message 引用,
// `admin@example.com` / `admin@pinconsole.local` 触发
// INVALID_LINKED_FORMAT 编译错误(error code 10)。
//
// 修复(1z P0):
// - 删除 i18n message 中的 `@` 字面量
// - LoginView 用 DEFAULT_ADMIN_EMAIL 常量作 placeholder
// - hint 文案用 {email} 参数占位
//
// regression 检查项:
// 1. LoginView 渲染无 console error / pageerror
// 2. placeholder 含 @ 符号(说明绕过了 i18n)
// 3. hint 渲染含完整 admin email(说明 {email} 插值生效)
// 4. console 无 INVALID_LINKED_FORMAT / @:linked-message 警告

import { test, expect } from '@playwright/test';

test.describe('1z P0: i18n @ SyntaxError', () => {
  test('LoginView 渲染无 i18n 编译错误 + placeholder/hint 含 @', async ({ page }) => {
    const errors: string[] = [];
    const warnings: string[] = [];
    page.on('pageerror', (err) => errors.push(`pageerror: ${err.message}`));
    page.on('console', (msg) => {
      const text = msg.text();
      if (msg.type() === 'error') errors.push(`console.error: ${text}`);
      if (msg.type() === 'warning') warnings.push(text);
    });

    await page.goto('/admin/login');
    // 等 SPA 挂载 + i18n 编译完成
    await page.waitForSelector('input[type="email"]', { timeout: 10_000 });

    // 1. placeholder 必须含 @(说明直接用常量,没经 i18n @-parsing)
    const placeholder = await page.locator('input[type="email"]').getAttribute('placeholder');
    expect(placeholder, 'email placeholder must contain @ (DEFAULT_ADMIN_EMAIL constant)').toContain('@');

    // 2. hint 必须含完整 admin email({email} 插值生效)
    const hintText = await page.locator('.default-hint').textContent();
    expect(hintText, 'default-hint must contain full admin email').toContain('admin@pinconsole.local');

    // 3. 给 vue-i18n 编译 + 渲染充分时间(编译错误是异步报出的)
    await page.waitForTimeout(500);

    // 4. 严格负断言:无 pageerror + 无 console.error
    // 过滤掉 /admin/login 加载时 router.ensureAuthInit 调 /api/auth/me → 401 的预期错误
    // (用户来登录页本来就是未登录状态,fetchMe 返回 401 是正常流程,不算 i18n regression)
    const i18nErrors = errors.filter((e) =>
      !/401|Unauthorized|auth\/me|Failed to load resource/i.test(e),
    );
    expect(i18nErrors, `unexpected i18n errors: ${JSON.stringify(i18nErrors)}`).toEqual([]);

    // 5. 严格负断言:console 无 vue-i18n linked-message 警告
    const i18nWarnings = warnings.filter((w) =>
      /INVALID_LINKED_FORMAT|Not acting as a modifier|linked message/i.test(w),
    );
    expect(i18nWarnings, `unexpected i18n warnings: ${JSON.stringify(i18nWarnings)}`).toEqual([]);
  });

  test('切换到英文后 LoginView 仍正常渲染(@ 不触发错误)', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(err.message));

    await page.goto('/admin/login');
    await page.waitForSelector('input[type="email"]', { timeout: 10_000 });

    // 先确保在中文(默认)— 直接切英文
    // 注:LoginView 自身没有 lang switch,但全局 i18n 切换不影响 @ 处理
    // 此场景验证 EN locale 加载也无 i18n 编译错误
    await page.evaluate(() => {
      // 通过修改 localStorage 模拟 EN locale 加载
      localStorage.setItem('mm_locale', 'en-US');
    });
    await page.reload();
    await page.waitForSelector('input[type="email"]', { timeout: 10_000 });

    const placeholder = await page.locator('input[type="email"]').getAttribute('placeholder');
    expect(placeholder).toContain('@');

    await page.waitForTimeout(500);
    const filtered = errors.filter((e) =>
      !/401|Unauthorized|auth\/me|Failed to load resource/i.test(e),
    );
    expect(filtered, `unexpected errors after reload: ${JSON.stringify(filtered)}`).toEqual([]);
  });
});
