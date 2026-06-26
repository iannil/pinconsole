/**
 * fork-3b observer — 验证 replay-core record() 在真实浏览器中录制 DOM 变更。
 *
 * 测试 MutationObserver 驱动的录制能正确捕获：
 *   - appendChild / removeChild
 *   - attribute 变更
 *   - 文本节点变更
 *
 * 策略：
 *   1. page.route() 拦截所有请求，内联 IIFE bundle
 *   2. page.evaluate() 调用 ReplayCore.record()，执行 DOM 操作，stop()
 *   3. 返回收集的 events，断言类型和数量
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

const BASE = 'http://observer.test';

function pageHTML(): string {
  return `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
    <div id="test-root"></div>
    <div id="extra-root"></div>
    <script>${replayCoreBundle}</script>
    <script>
      window.__recordEvents = [];
      window.__recStop = null;

      window.__startRecord = () => {
        window.__recordEvents = [];
        const stop = ReplayCore.record({
          emit(event) {
            window.__recordEvents.push(event);
          },
          // 不录制额外信息，减少噪音
          recordCanvas: false,
          collectFonts: false,
          sampling: {
            scroll: 999999,  // 几乎不采样滚动
            input: 999999,
          },
        });
        window.__recStop = stop;
        return true;
      };

      window.__stopRecord = () => {
        if (window.__recStop) {
          window.__recStop();
          window.__recStop = null;
        }
        return window.__recordEvents;
      };

      window.__recReady = true;
    </script>
  </body></html>`;
}

test.describe('fork-3b observer', () => {
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

  test('record captures full snapshot on start', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      (window as any).__startRecord();
      const stop = (window as any).__recStop;

      // 小延迟确保 full snapshot 已触发
      return new Promise((resolve) => {
        setTimeout(() => {
          stop();
          resolve((window as any).__recordEvents);
        }, 200);
      });
    });

    expect(events.length).toBeGreaterThanOrEqual(1);
    // type=2 是 FullSnapshot, type=4 是 Meta
    const fullSnapshots = events.filter((e: any) => e.type === 2);
    expect(fullSnapshots.length).toBeGreaterThanOrEqual(1);
  });

  test('record captures DOM appendChild', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      (window as any).__startRecord();

      // 执行 DOM 操作
      const root = document.getElementById('test-root')!;
      const child = document.createElement('div');
      child.id = 'appended';
      child.textContent = 'new child';
      root.appendChild(child);

      return new Promise((resolve) => {
        setTimeout(() => {
          (window as any).__recStop();
          resolve((window as any).__recordEvents);
        }, 300);
      });
    });

    // 验证有增量事件（type=3）
    const incrementals = events.filter((e: any) => e.type === 3);
    expect(incrementals.length).toBeGreaterThanOrEqual(1);
  });

  test('record captures attribute modification', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      const root = document.getElementById('test-root')!;
      root.setAttribute('data-test', 'initial');
      (window as any).__startRecord();

      root.setAttribute('data-test', 'modified');
      root.style.color = 'red';

      return new Promise((resolve) => {
        setTimeout(() => {
          (window as any).__recStop();
          resolve((window as any).__recordEvents);
        }, 300);
      });
    });

    const incrementals = events.filter((e: any) => e.type === 3);
    expect(incrementals.length).toBeGreaterThanOrEqual(1);
  });

  test('record captures text content change', async ({ page }) => {
    const events: any[] = await page.evaluate(() => {
      const root = document.getElementById('test-root')!;
      root.textContent = 'original';
      (window as any).__startRecord();

      root.textContent = 'modified text';

      return new Promise((resolve) => {
        setTimeout(() => {
          (window as any).__recStop();
          resolve((window as any).__recordEvents);
        }, 300);
      });
    });

    const incrementals = events.filter((e: any) => e.type === 3);
    expect(incrementals.length).toBeGreaterThanOrEqual(1);
  });
});
