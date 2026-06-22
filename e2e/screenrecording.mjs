// 操作全流程录屏脚本：Playwright + recordVideo → webm
// 运行: node e2e/screenrecording.mjs
import playwrightPkg from '/Users/rong.zhu/Code/pinconsole/node_modules/.pnpm/playwright@1.61.0/node_modules/playwright/index.js';
const { chromium } = playwrightPkg;
import { mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

const OUT_DIR = resolve('docs/progress/screenrecording-2026-06-22/video');
mkdirSync(OUT_DIR, { recursive: true });

const ADMIN_EMAIL = 'admin@pinconsole.local';
const ADMIN_PASS = 'devpass_test_only_1781760461';
const BASE = 'http://localhost:8080';

async function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function run() {
  console.log('[1/12] Launching chromium with recordVideo →', OUT_DIR);
  const browser = await chromium.launch({
    headless: false,
    args: ['--window-size=1440,900'],
  });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    recordVideo: {
      dir: OUT_DIR,
      size: { width: 1440, height: 900 },
    },
    userAgent:
      'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
  });

  // ============== Admin tab ==============
  console.log('[2/12] Admin login');
  const admin = await context.newPage();
  await admin.goto(`${BASE}/admin/login`);
  await sleep(1500);
  await admin.getByRole('textbox', { name: '密码' }).fill(ADMIN_PASS);
  await admin.getByRole('button', { name: '登录' }).click();
  await admin.waitForURL('**/admin/dashboard');
  await sleep(1500);

  // ============== Visitor tab ==============
  console.log('[3/12] Visitor opens demo (clear storage → reload for consent banner)');
  const visitor = await context.newPage();
  await visitor.goto(`${BASE}/landing/demo/`);
  await sleep(1500);
  await visitor.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  });
  await visitor.goto(`${BASE}/landing/demo/`);
  await sleep(2500);

  // 同意 consent banner(如果有)
  const acceptBtn = visitor.getByRole('button', { name: '同意' });
  if (await acceptBtn.count()) {
    console.log('    → consent banner shown, clicking 同意');
    await acceptBtn.click();
    await sleep(1500);
  }

  // 访客填表
  console.log('[4/12] Visitor fills form');
  await visitor.getByRole('textbox', { name: '姓名' }).fill('张三');
  await visitor.getByRole('textbox', { name: '手机' }).fill('13800138000');
  await visitor.getByRole('textbox', { name: '备注' }).fill('想咨询一下意外险产品');
  await sleep(1500);

  // ============== Admin: select visitor & live replay ==============
  console.log('[5/12] Admin selects visitor, live rrweb replay with mask');
  await admin.bringToFront();
  await sleep(1000);
  // 选 events 最多那个
  const items = admin.locator('li', { hasText: /events/ });
  const count = await items.count();
  let target = items.first();
  if (count > 1) target = items.nth(0);
  await target.click();
  await sleep(2500);

  // ============== Start Co-browsing ==============
  console.log('[6/12] Admin: Start Co-browsing');
  await admin.getByRole('button', { name: 'Start Co-browsing' }).click();
  await sleep(2000);

  // ============== admin → visitor chat ==============
  console.log('[7/12] Admin → Visitor chat');
  await admin.getByRole('textbox', { name: '输入消息' }).fill(
    '您好，我看到您在看意外险产品，需要我介绍一下吗？',
  );
  await admin.getByRole('button', { name: '发送' }).click();
  await sleep(2000);

  // ============== visitor receives + replies ==============
  console.log('[8/12] Visitor opens chat, replies');
  await visitor.bringToFront();
  await sleep(1000);
  await visitor.getByRole('button', { name: '客服' }).click();
  await sleep(2000);
  await visitor.getByRole('textbox', { name: '输入消息' }).fill(
    '好的，请介绍一下，我主要关意外医疗的保额',
  );
  await visitor.getByRole('button', { name: '发送' }).click();
  await sleep(2500);

  // ============== admin receives reply + pushes popup ==============
  console.log('[9/12] Admin: switch to popup tab, push popup');
  await admin.bringToFront();
  await sleep(1500);
  await admin.getByRole('tab', { name: '弹窗' }).click();
  await sleep(800);
  await admin.getByRole('textbox', { name: '标题' }).fill('限时优惠：意外险 6 折');
  await admin.getByRole('textbox', { name: '正文' }).fill(
    '现在投保意外险可享 6 折优惠，还赠 $200 意外医疗保额。点击了解详情。',
  );
  await admin.getByRole('textbox', { name: '按钮文字（可选）' }).fill('立即查看');
  await admin.getByRole('textbox', { name: '按钮链接（可选）' }).fill('https://example.com/promo');
  await admin.getByRole('button', { name: '发送弹窗' }).click();
  await sleep(2000);

  // ============== visitor receives popup + closes ==============
  console.log('[10/12] Visitor: receive popup, close it');
  await visitor.bringToFront();
  await sleep(2000);
  // 点右上角 X 关闭
  const xBtn = visitor.locator('button[aria-label="关闭"]').first();
  if (await xBtn.count()) {
    await xBtn.click();
  } else {
    // fallback：按文字关闭
    await visitor.getByRole('button', { name: '关闭' }).last().click();
  }
  await sleep(1500);

  // ============== admin Stop Co-browsing ==============
  console.log('[11/12] Admin: Stop Co-browsing');
  await admin.bringToFront();
  await sleep(1000);
  await admin.getByRole('button', { name: 'Stop Co-browsing' }).click();
  await sleep(2000);

  // ============== Replay ==============
  console.log('[12/12] Replay list → single session → play 1x → 4x');
  await admin.goto(`${BASE}/admin/replay`);
  await sleep(2500);
  // 选 112 events 那行
  const row = admin.locator('tr', { hasText: '112' }).first();
  if (await row.count()) {
    await row.click();
  } else {
    // fallback：选第一个
    await admin.locator('tbody tr').first().click();
  }
  await admin.waitForURL('**/admin/replay/**');
  await sleep(2500);

  // 播放
  await admin.getByRole('button', { name: 'Play' }).click();
  await sleep(6000);

  // 切 4x 再播
  await admin.getByRole('button', { name: '4×' }).click();
  await sleep(500);
  await admin.getByRole('button', { name: 'Play' }).click();
  await sleep(4000);

  // ============== 收尾：保存视频 ==============
  console.log('Closing context (this finalizes the webm)...');
  const adminVideo = await admin.video()?.path();
  const visitorVideo = await visitor.video()?.path();
  await context.close();
  await browser.close();
  console.log('---');
  console.log('Admin page video:  ', adminVideo);
  console.log('Visitor page video:', visitorVideo);
  console.log('All videos in:     ', OUT_DIR);
}

run().catch((err) => {
  console.error('FATAL:', err);
  process.exit(1);
});
