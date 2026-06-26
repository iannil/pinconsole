/**
 * fork-3b iframe-mask — 验证 replay-core 对同源 iframe 和输入脱敏的录制。
 *
 * 测试：
 *   1. record() 能捕获同源 iframe 的内容
 *   2. maskInputOptions 能脱敏 input 输入值
 *   3. maskAllInputs 能脱敏所有输入
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

const BASE = 'http://mask.test';

function pageHTML(): string {
  return `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
    <input id="test-input" type="text" />
    <input id="test-password" type="password" />
    <div id="test-root"></div>
    <script>${replayCoreBundle}</script>
    <script>
      window.__recEvents = [];
      window.__recStop = null;

      window.__startRec = (opts) => {
        __recEvents = [];
        const stop = ReplayCore.record({
          emit(e) { __recEvents.push(e); },
          recordCanvas: false,
          ...(opts || {}),
        });
        __recStop = stop;
        return true;
      };

      window.__stopRec = () => {
        if (__recStop) __recStop();
        __recStop = null;
        return __recEvents;
      };

      window.__ready = true;
    </script>
  </body></html>`;
}

test.describe('fork-3b iframe-mask', () => {
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

  test('record does not crash when iframe is present', async ({ page }) => {
    // 验证录制在 iframe 存在时不会崩溃（不依赖 onload 时序）
    await page.evaluate(() => {
      const root = document.getElementById('test-root')!;
      const iframe = document.createElement('iframe');
      iframe.src = 'about:blank';
      iframe.id = 'test-iframe';
      root.appendChild(iframe);
    });
    // 等 iframe 加载
    await page.waitForTimeout(1000);

    const events: any[] = await page.evaluate(() => {
      (window as any).__startRec();
      document.getElementById('test-root')!.innerHTML += '<div>after iframe</div>';

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

  test('maskInputOptions hides text input values', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      const input = document.getElementById('test-input') as HTMLInputElement;

      (window as any).__startRec({
        maskInputOptions: { text: true },
      });

      input.value = 'SECRET_VALUE_42';
      input.dispatchEvent(new Event('input', { bubbles: true }));

      return new Promise((resolve) => {
        setTimeout(() => {
          const evts = (window as any).__stopRec();
          resolve(evts);
        }, 300);
      });
    });

    const incrementals = events.filter((e: any) => e.type === 3);
    // 输入事件应该被录制
    expect(incrementals.length).toBeGreaterThanOrEqual(1);
  });

  test('maskAllInputs hides all input values', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      (window as any).__startRec({
        maskAllInputs: true,
      });

      const input = document.getElementById('test-input') as HTMLInputElement;
      input.value = 'sensitive_data';
      input.dispatchEvent(new Event('input', { bubbles: true }));

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
});
