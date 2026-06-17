import { test } from '@playwright/test';

test('debug: replay viewer 错误', async ({ browser }) => {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();

  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(String(e)));
  page.on('console', (m) => {
    if (m.type() === 'error') errors.push(`[console.error] ${m.text()}`);
  });

  await page.goto('/admin/replay/07fae50d-20e1-44ec-add8-68b04d2e904e');
  await page.waitForTimeout(3000);

  console.log('=== errors ===');
  for (const e of errors) console.log(e);

  await ctx.close();
});
