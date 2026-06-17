import { test, expect } from '@playwright/test';
import { readFileSync, existsSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

// e2e tests run from e2e/ subdir,project root 是上一级
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const PROJECT_ROOT = resolve(__dirname, '..', '..');
const pathFromRoot = (p: string) => resolve(PROJECT_ROOT, p);

// 切片 1j i18n + 部署 + CI:e2e 验收
// 前置:release 模式构建的二进制在 :8080,infra 起在 docker compose
// 深度判定:见 docs/standards/verification-depth.md
//
// R2 升级说明(2026-06-18):
// - 4 个场景全部实化(原占位跳过)
// - 配套修复:admin 子组件硬编码中文全部抽 i18n key
// - i18n 切换 e2e 真验证中英文文案切换
// - docker-prod e2e 真启动 prod profile(用 8090 端口避免冲突)
// - CI workflow lint 验证 ci.yml 结构
// - README 命令验证

test.describe('1j', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(120_000);
  });

  test('1j 场景1:i18n 中英切换', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto('/admin/');

    // 默认应为中文
    await expect(page.locator('.title')).toContainText('运营后台');

    // 找到语言切换按钮(中文显示"切换到英文"),点击切英文
    const switchBtn = page.getByRole('button', { name: '切换到英文' });
    await switchBtn.click();

    // 切换后应为英文(Operator Console)
    await expect(page.locator('.title')).toContainText('Operator Console');

    // 切换按钮文字应变(英文下显示 "Switch to Chinese")
    await expect(page.getByRole('button', { name: 'Switch to Chinese' })).toBeVisible();

    await ctx.close();
  });

  test('1j 场景2:docker-prod 启动(用 8090 避免冲突)', async () => {
    test.skip(!existsSync(pathFromRoot('docker-compose.yml')), 'docker-compose.yml required');

    // 用 SERVER_PORT=8090 启 prod 容器,避免与本机 release 二进制(8080)冲突
    const env = { ...process.env, SERVER_PORT: '8090' };
    try {
      execFileSync(
        'docker',
        ['compose', '--profile', 'prod', 'up', '-d', '--build', 'server'],
        { timeout: 240_000, stdio: 'pipe', env, cwd: PROJECT_ROOT },
      );

      // 等待 server healthy(最多 90s)
      let healthy = false;
      for (let i = 0; i < 45; i++) {
        try {
          const psOut = execFileSync(
            'docker',
            ['compose', 'ps', '--format', '{{.Name}} {{.Health}}'],
            { encoding: 'utf-8', env, cwd: PROJECT_ROOT },
          );
          if (/server.*healthy/i.test(psOut)) {
            healthy = true;
            break;
          }
        } catch {
          // ignore
        }
        // 等 2s
        await new Promise((r) => setTimeout(r, 2000));
      }

      expect(healthy, 'server container should be healthy within 90s').toBe(true);

      // 验证 prod server 真响应
      const healthResp = execFileSync('curl', ['-s', 'http://localhost:8090/healthz'], {
        encoding: 'utf-8',
      });
      expect(healthResp).toContain('"status":"alive"');
    } finally {
      // 停 prod server(保留 infra)
      try {
        execFileSync('docker', ['compose', 'stop', 'server'], { stdio: 'pipe', env, cwd: PROJECT_ROOT });
        execFileSync('docker', ['compose', 'rm', '-f', 'server'], { stdio: 'pipe', env, cwd: PROJECT_ROOT });
      } catch {
        // ignore cleanup errors
      }
    }
  });

  test('1j 场景3:CI workflow 含必需任务', async () => {
    // 验证 .github/workflows/ci.yml 存在且含 go test / pnpm test / docker build / compose smoke
    const ciPath = pathFromRoot('.github/workflows/ci.yml');
    expect(existsSync(ciPath), 'ci.yml should exist').toBe(true);

    const ciContent = readFileSync(ciPath, 'utf-8');

    // 关键 job 必须存在
    expect(ciContent, 'must have go-check job').toContain('go-check');
    expect(ciContent, 'must have js-check job').toContain('js-check');
    expect(ciContent, 'must have docker-build job').toContain('docker-build');
    expect(ciContent, 'must have compose-smoke job').toContain('compose-smoke');

    // 关键 step 必须存在
    expect(ciContent, 'must run go test').toContain('go test');
    expect(ciContent, 'must run pnpm test or build').toMatch(/pnpm.*(test|build)/);
    expect(ciContent, 'must apply migrations in compose-smoke').toMatch(/migration|\.up\.sql/);
  });

  test('1j 场景4:README 含快速开始命令', async () => {
    // 验证 README.md 含核心命令
    const readmePath = pathFromRoot('README.md');
    expect(existsSync(readmePath), 'README.md should exist').toBe(true);

    const readme = readFileSync(readmePath, 'utf-8');

    // 快速开始 / quick start
    expect(
      readme.toLowerCase(),
      'should mention docker compose up',
    ).toMatch(/docker[-\s]?compose\s+up/);

    // 关键命令存在(README 用原始命令,Makefile 是 wrapper)
    expect(readme, 'should contain docker compose up').toContain('docker compose up');
    expect(readme, 'should contain go build command').toMatch(/go build.*release/);
    expect(readme, 'should contain pnpm command').toMatch(/pnpm\s+\S+/);
  });
});
