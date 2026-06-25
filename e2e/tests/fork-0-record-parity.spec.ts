/**
 * record-parity — Playwright E2E 夹具
 *
 * 验证 replay-core.record() 与原始 rrweb.record() 对相同 DOM + 相同交互
 * 产生相同的 events 流。
 *
 * 策略（两个 recorder 不能同时运行——都 patch DOM）：
 *   1. page.route() 拦截，按 query 参数 ?lib= 切换 bundle
 *   2. Round 1: ?lib=rrweb → 记录 events
 *   3. Round 2: ?lib=replay-core → 记录 events
 *   4. 比较 events（排除 timestamp）
 */
import { test, expect } from '@playwright/test';
import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, resolve } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const rrwebBundle = readFileSync(
  resolve(__dirname, '../../visitor-sdk/node_modules/rrweb/dist/rrweb.js'),
  'utf-8',
);
const replayCoreBundle = readFileSync(
  resolve(__dirname, '../../packages/replay-core/dist/replay-core.js'),
  'utf-8',
);

function makeRecordPage(importPath: string) {
  return `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Record Parity</title></head>
<body>
  <div id="app">
    <h1>Record Parity Test</h1>
    <button id="btn-add">Add Item</button>
    <button id="btn-remove">Remove Item</button>
    <ul id="list"><li class="item">Item 1</li><li class="item">Item 2</li></ul>
    <input id="txt-input" type="text" placeholder="type here">
    <span id="output"></span>
  </div>
  <script type="module">
    import { record } from '${importPath}';

    const events = [];
    const stopFn = record({ emit: (e) => events.push(e), maskAllInputs: false });

    // Wait for fullSnapshot, then run interactions, then stop
    await new Promise(r => setTimeout(r, 200));

    const a = document.getElementById('btn-add');
    const inp = document.getElementById('txt-input');
    const out = document.getElementById('output');
    const rm = document.getElementById('btn-remove');

    a.click();
    await new Promise(r => setTimeout(r, 50));
    a.click();
    await new Promise(r => setTimeout(r, 50));

    inp.focus();
    inp.value = 'hello';
    inp.dispatchEvent(new Event('input', { bubbles: true }));
    await new Promise(r => setTimeout(r, 50));
    inp.value = 'hello world';
    inp.dispatchEvent(new Event('input', { bubbles: true }));
    await new Promise(r => setTimeout(r, 50));

    out.textContent = 'done';
    await new Promise(r => setTimeout(r, 50));

    rm.click();
    await new Promise(r => setTimeout(r, 200));

    // Stop recording AFTER interactions complete
    stopFn();
    // Small delay for last mutations to flush
    await new Promise(r => setTimeout(r, 100));
    window.__events = events;
  </script>
</body></html>`;
}

function normalizeEvents(events: any[]) {
  return events.map(({ timestamp, data, ...rest }) => {
    // Normalize: strip href from Meta events (depends on page URL)
    if (rest.type === 4 /* Meta */ && data?.href) {
      const { href, ...restData } = data;
      return { ...rest, data: { ...restData, href: 'about:blank' } };
    }
    return { ...rest, data };
  });
}

const HOST = 'http://parity.test';

test.describe('record-parity', () => {
  test('相同 DOM + 相同交互 → 相同 events', async ({ page }) => {
    // Single route: HTML by query param, JS by path
    await page.route(`${HOST}/**`, (route) => {
      const url = route.request().url();
      const u = new URL(url);
      if (u.pathname === '/') {
        const lib = u.searchParams.get('lib') || 'rrweb';
        const importPath = lib === 'replay-core' ? '/replay-core.js' : '/rrweb.js';
        route.fulfill({ status: 200, contentType: 'text/html', body: makeRecordPage(importPath) });
      } else if (u.pathname === '/rrweb.js') {
        route.fulfill({ status: 200, contentType: 'application/javascript', body: rrwebBundle });
      } else if (u.pathname === '/replay-core.js') {
        route.fulfill({ status: 200, contentType: 'application/javascript', body: replayCoreBundle });
      } else {
        route.fulfill({ status: 404 });
      }
    });

    // ========== Round 1: Old rrweb ==========
    await page.goto(`${HOST}/?lib=rrweb`);
    await page.waitForFunction(() => (window as any).__events !== undefined, undefined, { timeout: 10_000 });
    const oldRaw = await page.evaluate(() => (window as any).__events);

    // ========== Round 2: New replay-core ==========
    await page.goto(`${HOST}/?lib=replay-core`);
    await page.waitForFunction(() => (window as any).__events !== undefined, undefined, { timeout: 10_000 });
    const newRaw = await page.evaluate(() => (window as any).__events);

    const oldEvents = normalizeEvents(oldRaw);
    const newEvents = normalizeEvents(newRaw);

    expect(oldEvents.length).toBeGreaterThan(0);
    expect(newEvents.length).toBeGreaterThan(0);

    if (oldEvents.length !== newEvents.length) {
      console.error(`RECORD PARITY: count mismatch — old=${oldEvents.length} new=${newEvents.length}`);
    }

    for (let i = 0; i < Math.max(oldEvents.length, newEvents.length); i++) {
      const o = oldEvents[i];
      const n = newEvents[i];
      expect(o, `event[${i}] mismatch`).toEqual(n);
    }
  });
});
