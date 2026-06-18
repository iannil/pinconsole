import { defineConfig, devices } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __dirname_e2e = dirname(fileURLToPath(import.meta.url));
const projectRoot = resolve(__dirname_e2e, '..');

// 切片 1a：Playwright 配置。
// 仅跑 chromium；Firefox / WebKit 在 CI 中按需添加。
//
// 1v:加 webServer 自动起 server,让 pnpm test:e2e 不再依赖手动 ./ops.sh start。
// reuseExistingServer:若开发者已起 server,复用;否则启动 + 等待 healthz。
// SKIP_MM_WEBSERVER=1 可禁用(用于 CI 已有 server fixture 的场景)。
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: process.env.SKIP_MM_WEBSERVER
    ? undefined
    : {
        command: './ops.sh start',
        cwd: projectRoot,
        url: 'http://localhost:8080/healthz',
        reuseExistingServer: true,
        timeout: 120_000, // 首次 build 可能要 60s+
        stdout: 'pipe',
        stderr: 'pipe',
      },
});
