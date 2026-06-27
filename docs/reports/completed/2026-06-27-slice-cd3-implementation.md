# cd-3 自定义域名：e2e 验证 + 部署文档

**切片编号**：cd-3（Custom Domain v1 — e2e + docs）
**状态**：completed（2026-06-27）
**关联**：[cd-1 spec](../reports/completed/2026-06-26-slice-cd1-spec.md)、[cd-1 impl](../reports/completed/2026-06-26-slice-cd1-implementation.md)

## 实现内容

### 文件变更

| 文件 | 类型 | 内容 |
|---|---|---|
| `e2e/tests/cd-3-custom-domains.spec.ts` | 新增 | Playwright e2e：admin 添加/查看/删除自定义域名 |
| `README.md` | 更新 | 新增"Custom domains (ACME / Let's Encrypt)"部署文档 + 更新 Roadmap ✅ |

### 测试场景

Playwright e2e `cd-3 custom domains UI`：
1. 导航到 `/admin/domains` → 验证空状态可见
2. 输入域名并添加 → 域名出现在表格中
3. 验证 StatusBadge 渲染
4. 删除域名 → 确认弹窗 → 回到空状态

### 部署文档

README.md `Production deploy` 下新增表格说明：

| Variable | Default | Description |
|---|---|---|
| `PLATFORM_DOMAIN` | `""` | 主域名，豁免 Host-header 校验 |
| `ACME_EMAIL` | `""` | Let's Encrypt 注册邮箱（必填） |
| `ACME_STAGING` | `true` | staging CA（生产设 false） |
| `ACME_DATA_DIR` | `./data/certmagic` | 证书缓存目录 |
| `ACME_HTTP_PORT` | `80` | HTTP-01 challenge 端口 |

### 验证

- Go build: ✅
- storage 测试: ✅ (5.9s)
- api 测试: ✅ (9.3s)
- admin JS 测试: 204/204 ✅
- Playwright 解析: 1 test listed ✅
