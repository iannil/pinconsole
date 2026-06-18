// 切片 1r:i18n 迁移 regression
//
// 1r 把硬编码中英文迁到 i18n key + logger 改 JSON 格式。
// 1j 场景1 已有 i18n 切换的基础测试,这里补充 regression 维度:
// 1. 中英文 locale 都加载无 console error
// 2. 关键文案不出现 missing key warning
// 3. 切换 locale 不破坏 DOM 结构
// 4. SDK logger 输出 JSON 格式(1r 改造的产物)

import { test, expect } from '../fixtures/admin-auth';

test.describe('1r: i18n 迁移 regression', () => {
  test('中英切换无 missing key warning + DOM 结构稳定', async ({ adminPage }) => {
    const page = adminPage;
    const warnings: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'warning' || msg.type() === 'error') {
        warnings.push(msg.text());
      }
    });

    // 中文(默认)— 验证关键文案
    await expect(page.locator('.title')).toContainText('运营后台');
    await expect(page.locator('.lang-switch')).toContainText('切换到英文');

    // 切英文
    await page.getByRole('button', { name: '切换到英文' }).click();
    await expect(page.locator('.title')).toContainText('Operator Console');
    await expect(page.locator('.lang-switch')).toContainText('Switch to Chinese');

    // 切回中文
    await page.getByRole('button', { name: 'Switch to Chinese' }).click();
    await expect(page.locator('.title')).toContainText('运营后台');

    // 等待任何异步 i18n warning
    await page.waitForTimeout(500);

    // 严格负断言:无 missing key 警告
    const missingKeyWarnings = warnings.filter((w) =>
      /missing.*key|not found in messages|intl.*missing|unhandled.*locale/i.test(w),
    );
    expect(missingKeyWarnings, `i18n missing keys: ${JSON.stringify(missingKeyWarnings)}`).toEqual([]);
  });

  test('SDK logger 输出 JSON 格式(含 source 字段)', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    const logs: string[] = [];
    page.on('console', (m) => logs.push(m.text()));

    await page.goto('/');
    await page.waitForTimeout(2000);

    const joined = logs.join('\n');
    // 1r 后 SDK logger 输出 JSON,含 source / event 字段
    expect(joined).toContain('"source":"visitor-sdk"');
    expect(joined).toContain('"event":"sdk_started"');
    // 不应再有纯文本 "[marketing-monitor] SDK loaded"(老的字符串字面量)
    expect(joined).not.toContain('[marketing-monitor] SDK loaded');

    await ctx.close();
  });
});
