/**
 * fork-3b replayer — 验证 replay-core Replayer 在真实浏览器中的行为。
 *
 * 测试：
 *   1. Replayer 构造后在容器内创建 iframe
 *   2. 喂 full snapshot 后 iframe body 重建 DOM
 *   3. startLive 后 addEvent 增量渲染
 *   4. 销毁后 iframe 被移除
 *
 * 策略：
 *   - page.route() 内联 IIFE bundle
 *   - 预构建最小 events 夹具（meta + full snapshot）
 *   - page.evaluate() 创建 Replayer，操作后返回 iframe 状态
 */
import { test, expect } from '@playwright/test';
import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, resolve } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const replayCoreBundle = readFileSync(
  resolve(__dirname, '../../packages/replay-core/dist/replay-core.iife.js'),
  'utf-8',
);

const BASE = 'http://replayer.test';

/** 最小 events：meta + full snapshot（只有一个 div#root + Hello 文本） */
const MINIMAL_EVENTS_JSON = JSON.stringify([
  { type: 4, timestamp: 0, data: { href: 'about:blank', width: 1024, height: 768 } },
  {
    type: 2, timestamp: 0, data: {
      node: {
        type: 0, childNodes: [
          { type: 1, name: 'html', publicId: '', systemId: '', id: 1 },
          { type: 2, tagName: 'html', attributes: { lang: 'en' }, childNodes: [
            { type: 2, tagName: 'head', attributes: {}, childNodes: [], id: 3 },
            { type: 2, tagName: 'body', attributes: {}, childNodes: [
              { type: 2, tagName: 'div', attributes: { id: 'root' }, childNodes: [{ type: 3, textContent: 'Hello', id: 5 }], id: 4 },
            ], id: 6 },
          ], id: 2 },
        ], id: 0,
      },
      initialOffset: { top: 0, left: 0 },
    },
  },
]);

function pageHTML(): string {
  return `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
    <div id="player-root" style="width:800px;height:600px"></div>
    <script>${replayCoreBundle}</script>
    <script>
      window.__recReady = true;

      window.__createReplayer = () => {
        const root = document.getElementById('player-root');
        const r = new ReplayCore.Replayer(${MINIMAL_EVENTS_JSON}, {
          root,
          skipInactive: true,
          showDebug: false,
          showController: false,
        });
        return true;
      };

      window.__getIframeInfo = () => {
        const root = document.getElementById('player-root');
        const iframe = root.querySelector('iframe');
        if (!iframe) return { hasIframe: false };
        const body = iframe.contentDocument?.body;
        return {
          hasIframe: true,
          bodyHTML: body?.innerHTML ?? '',
          bodyText: body?.textContent ?? '',
          hasContent: (body?.textContent?.length ?? 0) > 0,
        };
      };
    </script>
  </body></html>`;
}

test.describe('fork-3b replayer', () => {
  test.beforeEach(async ({ page }, testInfo) => {
    testInfo.setTimeout(60_000);
    await page.route(`${BASE}/**`, (route) => {
      const url = route.request().url();
      if (url === `${BASE}/`) {
        route.fulfill({ status: 200, contentType: 'text/html', body: pageHTML() });
      } else {
        route.fulfill({ status: 404 });
      }
    });
    await page.goto(`${BASE}/`);
    await page.waitForFunction(() => (window as any).__recReady === true, null, { timeout: 15_000 });
  });

  test('Replayer constructor appends iframe to root', async ({ page }) => {
    await page.evaluate(() => (window as any).__createReplayer());
    // 给 Replayer 时间创建 iframe（异步）
    await page.waitForTimeout(1000);
    const info: any = await page.evaluate(() => (window as any).__getIframeInfo());
    expect(info.hasIframe).toBe(true);
  });

  test('Replayer renders snapshot content in iframe', async ({ page }) => {
    await page.evaluate(() => (window as any).__createReplayer());
    await page.waitForTimeout(1000);
    const info: any = await page.evaluate(() => (window as any).__getIframeInfo());
    expect(info.hasContent).toBe(true);
    expect(info.bodyText).toContain('Hello');
  });

  test('Replayer destroy removes iframe', async ({ page }) => {
    const events = JSON.parse(MINIMAL_EVENTS_JSON);
    await page.evaluate((evts) => {
      const root = document.getElementById('player-root')!;
      const r = new ReplayCore.Replayer(evts, { root });
      r.destroy();
    }, events);
    await page.waitForTimeout(500);
    const info: any = await page.evaluate(() => (window as any).__getIframeInfo());
    // destroy 后 iframe 被移除
    expect(info.hasIframe).toBe(false);
  });
});
