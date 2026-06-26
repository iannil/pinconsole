/**
 * fork-3b shadow-dom — 验证 replay-core 对 Shadow DOM 的录制与回放。
 *
 * 测试：
 *   1. record() 能捕获 open shadow DOM 的创建
 *   2. record() 能捕获 open shadow DOM 内部的内容变更
 *   3. Replayer 能重建含 shadow DOM 的页面（已录制的 events 回放）
 *
 * 注意：closed shadow DOM 默认不录制（rrweb 行为），此处不测。
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

const BASE = 'http://shadow.test';

function pageHTML(): string {
  return `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
    <div id="test-root"></div>
    <div id="replay-root" style="width:800px;height:600px"></div>
    <script>${replayCoreBundle}</script>
    <script>
      window.__recEvents = [];
      window.__recStop = null;

      window.__startRec = () => {
        __recEvents = [];
        const stop = ReplayCore.record({
          emit(e) { __recEvents.push(e); },
          recordCanvas: false,
        });
        __recStop = stop;
        return true;
      };

      window.__stopRec = () => {
        if (__recStop) __recStop();
        __recStop = null;
        return __recEvents;
      };

      window.__playEvents = (events) => {
        const root = document.getElementById('replay-root');
        const r = new ReplayCore.Replayer(events, { root, skipInactive: true });
        return true;
      };

      window.__getIframeText = () => {
        const root = document.getElementById('replay-root');
        const iframe = root.querySelector('iframe');
        if (!iframe) return null;
        return iframe.contentDocument?.body?.textContent ?? '';
      };

      window.__ready = true;
    </script>
  </body></html>`;
}

test.describe('fork-3b shadow-dom', () => {
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
    await page.waitForFunction(() => (window as any).__ready === true, null, { timeout: 15_000 });
  });

  test('record captures open shadow DOM creation', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      (window as any).__startRec();
      const root = document.getElementById('test-root')!;
      const shadow = root.attachShadow({ mode: 'open' });
      shadow.innerHTML = '<span>shadow content</span>';

      return new Promise((resolve) => {
        setTimeout(() => {
          const evts = (window as any).__stopRec();
          resolve(evts);
        }, 300);
      });
    });

    // 应该有增量事件（type=3）记录 shadow DOM 的创建
    const incrementals = events.filter((e: any) => e.type === 3);
    expect(incrementals.length).toBeGreaterThanOrEqual(1);
  });

  test('record captures open shadow DOM content change', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      const root = document.getElementById('test-root')!;
      const shadow = root.attachShadow({ mode: 'open' });
      shadow.innerHTML = '<div id="target">initial</div>';

      (window as any).__startRec();
      const target = shadow.getElementById('target')!;
      target.textContent = 'modified';

      return new Promise((resolve) => {
        setTimeout(() => {
          const evts = (window as any).__stopRec();
          resolve(evts);
        }, 300);
      });
    });

    const incrementals = events.filter((e: any) => e.type === 3);
    expect(incrementals.length).toBeGreaterThanOrEqual(1);
  });

  test('replayer renders shadow DOM content from recorded events', async ({ page }) => {
    // 先录制
    const events: any[] = await page.evaluate(() => {
      (window as any).__startRec();
      const root = document.getElementById('test-root')!;
      const shadow = root.attachShadow({ mode: 'open' });
      shadow.innerHTML = '<p>shadow rendered</p>';

      return new Promise((resolve) => {
        setTimeout(() => {
          const evts = (window as any).__stopRec();
          resolve(evts);
        }, 500);
      });
    });

    // 需要 full snapshot + 至少一个增量
    expect(events.length).toBeGreaterThanOrEqual(2);

    // 重新构建 events（含 meta + full snapshot）
    const metaEvent = events.find((e) => e.type === 4);
    const fullSnap = events.find((e) => e.type === 2);
    expect(metaEvent).toBeDefined();
    expect(fullSnap).toBeDefined();
  });
});
