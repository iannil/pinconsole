/**
 * replay-parity — Playwright E2E 夹具
 *
 * 验证 replay-core Replayer 与原始 rrweb Replayer 对相同 events
 * 产生相同的 iframe 渲染输出。
 *
 * 策略（钻穿到原始 Replayer，不经过 rrweb-player/Svelte）：
 *   1. page.route() 拦截所有请求，提供 HTML + rrweb.js + replay-core.js
 *   2. page.goto() → 加载 HTML，ES import 双 Replayer
 *   3. 喂相同 fixture events → 等待双方 Finish → 比较 iframe 内容
 */
import { test, expect } from '@playwright/test';
import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, resolve } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// ============================================================
// 加载 bundle 内容
// ============================================================
const rrwebBundle = readFileSync(
  resolve(__dirname, '../../node_modules/.pnpm/rrweb@2.0.0-alpha.20/node_modules/rrweb/dist/rrweb.js'),
  'utf-8',
);

const replayCoreBundle = readFileSync(
  resolve(__dirname, '../../packages/replay-core/dist/replay-core.js'),
  'utf-8',
);

// ============================================================
// 最小 events 夹具
// ============================================================
const FIXTURE_EVENTS = [
  { type: 4, timestamp: 0, data: { href: 'about:blank', width: 1024, height: 768 } },
  {
    type: 2, timestamp: 0, data: {
      node: {
        type: 0, childNodes: [
          { type: 1, name: 'html', publicId: '', systemId: '', id: 1 },
          { type: 2, tagName: 'html', attributes: { lang: 'en' }, childNodes: [
            { type: 2, tagName: 'head', attributes: {}, childNodes: [], id: 3 },
            { type: 2, tagName: 'body', attributes: {}, childNodes: [
              { type: 2, tagName: 'div', attributes: { id: 'root', class: 'container' }, childNodes: [{ type: 3, textContent: 'Hello Parity!', id: 6 }], id: 4 },
              { type: 2, tagName: 'span', attributes: {}, childNodes: [{ type: 3, textContent: 'world', id: 8 }], id: 7 },
            ], id: 5 },
          ], id: 2 },
        ], id: 0,
      },
      initialOffset: { top: 0, left: 0 },
    },
  },
];

const PARITY_HOST = 'http://parity.test';

// ============================================================
// 测试
// ============================================================
test.describe('replay-parity', () => {
  test('相同 events → 相同 iframe outerHTML', async ({ page }) => {
    // route: 只拦截 parity.test 的顶层请求
    await page.route(`${PARITY_HOST}/**`, (route) => {
      const url = route.request().url();
      if (url === `${PARITY_HOST}/`) {
        // HTML 测试页
        route.fulfill({
          status: 200,
          contentType: 'text/html',
          body: `<!DOCTYPE html><html><body>
            <div id="old-container"></div><div id="new-container"></div>
            <script type="module">
              import { Replayer as OldReplayer } from '/rrweb.js';
              import { Replayer as NewReplayer } from '/replay-core.js';

              const events = ${JSON.stringify(FIXTURE_EVENTS)};
              const config = { speed: 999999 };

              // Create both replayers; the constructor auto-starts replay.
              const oldR = new OldReplayer(events, config);
              const newR = new NewReplayer(events, config);

              // Wait for both to settle (rrweb uses async replay via setTimeout)
              await new Promise(r => setTimeout(r, 1000));

              // Extract iframe outerHTML
              function getIframeHTML(replayer) {
                const w = replayer.wrapper;
                if (!w) return '';
                const frames = w.querySelectorAll('iframe');
                if (!frames.length) return '';
                // Last iframe is the renderer (first may be mirror in some modes)
                const f = frames[frames.length - 1];
                // Normalize: remove rrweb internal id attributes for comparison
                return f.outerHTML;
              }

              window.__result = [getIframeHTML(oldR), getIframeHTML(newR)];
            <\/script></body></html>`,
        });
      } else if (url === `${PARITY_HOST}/rrweb.js`) {
        route.fulfill({ status: 200, contentType: 'application/javascript', body: rrwebBundle });
      } else if (url === `${PARITY_HOST}/replay-core.js`) {
        route.fulfill({ status: 200, contentType: 'application/javascript', body: replayCoreBundle });
      } else {
        route.fulfill({ status: 404 });
      }
    });

    await page.goto(PARITY_HOST);

    // 等待 parity 结果（模块脚本异步执行）
    await page.waitForFunction(() => (window as any).__result !== undefined, undefined, { timeout: 30_000 });

    const result = await page.evaluate(() => {
      return (window as any).__result as [string, string] | undefined;
    });

    expect(result).toBeDefined();
    const [oldHTML, newHTML] = result!;

    expect(oldHTML).toBeTruthy();
    expect(newHTML).toBeTruthy();

    if (oldHTML !== newHTML) {
      console.error('REPLAY PARITY MISMATCH');
      console.error('=== OLD (rrweb) ===');
      console.error(oldHTML.substring(0, 500));
      console.error('=== NEW (replay-core) ===');
      console.error(newHTML.substring(0, 500));
    }

    expect(oldHTML).toBe(newHTML);
  });
});
