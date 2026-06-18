// Admin auth fixtures for e2e tests.
//
// 1h-ui 上线后,admin SPA 走 Pinia store + Vue Router 守卫,isAuthenticated
// 默认 null,任何 /admin/* 路由直接 redirect 到 /login。
//
// 1k fail-secure 后 release binary 不再有 dev bypass,login endpoint 永远验密码。
//
// 1i antiscrape 拦 HeadlessChrome UA(playwright 默认 APIRequestContext / newContext UA),
// 因此 fixture 显式注入干净 Chrome UA(与 playwright.config.ts 的 Desktop Chrome device 一致)。
//
// 已知 SPA bug(2026-06-18):App.vue 在 onMounted 才调 fetchMe,但 router 守卫在
// 挂载前已执行。所以**仅靠 cookie 无法 restore session**(刷新 /admin/dashboard 会跳
// /admin/login)。fixture 必须走 UI login 触发 store.login() 把 user 写入 Pinia,
// 后续页面才能留在 dashboard 上。
//
// 用法:
//   import { test, expect } from '../fixtures/admin-auth';
//   test('...', async ({ adminPage }) => { ... });
//   test('...', async ({ adminContext }) => { /* 自己 newPage */ });
//   test('...', async ({ adminRequest }) => { /* REST API */ });

import { test as base, expect, type Page, type BrowserContext, type APIRequestContext } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const PROJECT_ROOT = resolve(__dirname, '..', '..');

// 干净 Chrome UA,避免被 1i antiscrape UABlockMiddleware 拦截(默认 HeadlessChrome)。
const CLEAN_CHROME_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.7827.55 Safari/537.36';

interface EnvConfig {
  adminEmail: string;
  adminPassword: string;
}

function loadEnv(): EnvConfig {
  const envPath = resolve(PROJECT_ROOT, '.env');
  let adminEmail = 'admin@marketing-monitor.local';
  let adminPassword = '';
  try {
    const raw = readFileSync(envPath, 'utf-8');
    for (const line of raw.split('\n')) {
      const m = line.match(/^\s*(ADMIN_EMAIL|ADMIN_PASSWORD)\s*=\s*(.+?)\s*$/);
      if (!m) continue;
      const val = m[2].replace(/^["']|["']$/g, '');
      if (m[1] === 'ADMIN_EMAIL') adminEmail = val;
      if (m[1] === 'ADMIN_PASSWORD') adminPassword = val;
    }
  } catch {
    // .env 不存在
  }
  if (!adminPassword) {
    throw new Error('ADMIN_PASSWORD missing in .env — e2e admin-auth fixture 无法登录');
  }
  return { adminEmail, adminPassword };
}

const ENV = loadEnv();
export const ADMIN_EMAIL = ENV.adminEmail;
export const ADMIN_PASSWORD = ENV.adminPassword;

// 用 UI 流程登录一个 context(打开 page → 填表单 → 提交 → 等 dashboard)。
// 这是必需的,因为 SPA 的 fetchMe 在 router 守卫之后跑,cookie-only restore 不工作。
// 返回登录完成的 page(在 /admin/dashboard 上)。
async function loginViaUI(context: BrowserContext): Promise<Page> {
  const page = await context.newPage();
  await page.goto('/admin/login');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/admin\/(dashboard|replay)/, { timeout: 15_000 });
  return page;
}

interface AdminAuthFixtures {
  // 已登录的 Page,在 /admin/dashboard 上,可直接用。
  adminPage: Page;
  // 已登录的 BrowserContext。Test 自己 newPage() 时 cookie 已注入,但**新 page
  // 仍需调用 page.goto('/admin/login') 走 UI 登录**(因为 SPA fetchMe 时序 bug)。
  // 通常测试应该用 adminPage 而不是 adminContext + newPage。
  adminContext: BrowserContext;
  // 已登录的 APIRequestContext(测试 REST API 时用)。Cookie 单独管理,
  // 不走 SPA,所以不受 fetchMe 时序 bug 影响。
  adminRequest: APIRequestContext;
}

export const test = base.extend<AdminAuthFixtures>({
  adminContext: async ({ browser }, use) => {
    // userAgent 显式注入,否则默认 HeadlessChrome 被 1i antiscrape 拦截。
    const ctx = await browser.newContext({ userAgent: CLEAN_CHROME_UA });
    // loginViaUI 在 ctx 内开 page + 走 UI 表单,设置 cookie + Pinia store
    await loginViaUI(ctx);
    await use(ctx);
    await ctx.close();
  },
  adminPage: async ({ adminContext }, use) => {
    // adminContext 已经有一个登录完成的 page(loginViaUI 返回的)
    const pages = adminContext.pages();
    const page = pages[pages.length - 1];
    await use(page);
  },
  adminRequest: async ({ playwright }, use) => {
    const ctx = await playwright.request.newContext({
      baseURL: process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:8080',
      userAgent: CLEAN_CHROME_UA,
    });
    const resp = await ctx.post('/api/auth/login', {
      data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
    });
    if (!resp.ok()) {
      throw new Error(`admin-auth fixture API login failed: ${resp.status()} ${await resp.text()}`);
    }
    await use(ctx);
    await ctx.dispose();
  },
});

export { expect };
