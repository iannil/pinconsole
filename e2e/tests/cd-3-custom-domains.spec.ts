// cd-3: 自定义域名 Admin UI e2e 测试。
//
// 验证:
// 1. /admin/domains 页面加载并显示"空状态"
// 2. 添加域名后出现在列表中（cert_status = pending）
// 3. 删除域名后从列表消失
//
// 前置条件: ops.sh start + docker infra 就绪
// 注意: ACME 证书签发需要真实 DNS + Let's Encrypt staging,不在本测试范围内。
// 本测试只验证 API + UI 交互流,证书签发状态 cert_status 由后端异步更新。

import { test, expect } from '../fixtures/admin-auth';

test.describe('cd-3 custom domains UI', () => {
  test.beforeEach(async ({}, testInfo) => {
    testInfo.setTimeout(60_000);
  });

  test('add, view, and remove a custom domain', async ({ adminPage: admin }) => {
    const domain = `e2e-test-${Date.now()}.example.com`;

    // 1. 导航到 /admin/domains
    await admin.goto('/admin/domains');
    await admin.waitForTimeout(1000);

    // 2. 验证空状态
    await expect(admin.locator('.empty')).toBeVisible({ timeout: 5000 });

    // 3. 输入域名并添加
    const input = admin.locator('.add-domain input');
    await input.fill(domain);

    const addBtn = admin.locator('.add-domain .btn-primary');
    await addBtn.click();

    // 4. 验证域名出现在列表中
    await expect(admin.locator('.domains-table')).toBeVisible({ timeout: 5000 });
    const domainCell = admin.locator('.domain-cell').first();
    await expect(domainCell).toHaveText(domain, { timeout: 5000 });

    // 5. 验证状态 badge 显示（pending / active / failed）
    const statusBadge = admin.locator('.pc-badge').first();
    await expect(statusBadge).toBeVisible({ timeout: 5000 });

    // 6. 删除域名
    const deleteBtn = admin.locator('.btn-danger').first();
    // 处理 confirm 弹窗
    admin.once('dialog', (dialog) => dialog.accept());
    await deleteBtn.click();
    await admin.waitForTimeout(1000);

    // 7. 验证回到空状态
    await expect(admin.locator('.empty')).toBeVisible({ timeout: 5000 });
  });
});
