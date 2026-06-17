// 共享 e2e setup：visitor + admin 双 context、订阅访客、等待上线
// 抽自原 realtime.spec.ts，被 1b-1g 各切片 spec 复用
import type { Browser, BrowserContext, Page } from '@playwright/test';

export interface VisitorAdmin {
  visitorCtx: BrowserContext;
  adminCtx: BrowserContext;
  visitor: Page;
  admin: Page;
}

/**
 * 打开 visitor demo 和 admin SPA 双 context。
 * 用法：
 *   const va = await openVisitorAndAdmin(browser);
 *   // ... 交互
 *   await closeVisitorAndAdmin(va);
 */
export async function openVisitorAndAdmin(
  browser: Browser,
  opts: { adminDelayMs?: number; visitorDelayMs?: number } = {},
): Promise<VisitorAdmin> {
  const visitorCtx = await browser.newContext();
  const adminCtx = await browser.newContext();
  const visitor = await visitorCtx.newPage();
  const admin = await adminCtx.newPage();

  await admin.goto('/admin/');
  await admin.waitForTimeout(opts.adminDelayMs ?? 1500);

  await visitor.goto('/');
  await visitor.waitForTimeout(opts.visitorDelayMs ?? 2000);

  return { visitorCtx, adminCtx, visitor, admin };
}

export async function closeVisitorAndAdmin(va: VisitorAdmin): Promise<void> {
  await va.visitorCtx.close();
  await va.adminCtx.close();
}

/**
 * 在 admin 端选中第一个非空访客并订阅实时。
 */
export async function subscribeFirstVisitor(admin: Page): Promise<void> {
  await admin.locator('.visitor-list li:not(.empty)').first().click();
  await admin.getByRole('button', { name: '订阅实时' }).click();
}

/**
 * 等待 admin 列表中出现至少 N 个访客。
 */
export async function waitForVisitors(admin: Page, minCount = 1, timeoutMs = 10_000): Promise<void> {
  await expect
    .poll(async () => admin.locator('.visitor-list li:not(.empty)').count(), {
      timeout: timeoutMs,
    })
    .toBeGreaterThanOrEqual(minCount);
}

// `expect` is global in playwright test files but not in helper modules.
// Re-import for type safety in helpers.
import { expect } from '@playwright/test';
