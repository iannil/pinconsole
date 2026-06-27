# cd-1 自定义域名后端实现报告

**切片编号**：cd-1
**状态**：completed（2026-06-26）
**持续时间**：~1 天（grill + 实现）
**关联**：[cd-1 spec](../progress/2026-06-26-slice-cd1-spec.md)

## 实现内容

### 文件变更

| 文件 | 类型 | 内容 |
|---|---|---|
| `server/internal/config/config.go` | 新增 | ACME Email/DataDir/Staging/PlatformDomain 配置字段 |
| `.env.example` | 更新 | 新增 6 个 ACME 配置示例 |
| `server/migrations/000009_custom_domains.up.sql` | 新增 | custom_domains 表（BIGSERIAL + tenant + domain + cert_status） |
| `server/migrations/000009_custom_domains.down.sql` | 新增 | DROP TABLE |
| `server/internal/storage/types.go` | 更新 | 新增 CustomDomain struct |
| `server/internal/storage/postgres.go` | 更新 | 新增 ErrConflict + pgErrCode 辅助函数 |
| `server/internal/storage/custom_domain_repo.go` | 新增 | Get/List/Create/UpdateStatus/Delete + ListActiveCustomDomains |
| `server/internal/storage/custom_domain_repo_test.go` | 新增 | CRUD + 重复创建 PG 集成测试 |
| `server/internal/cert/cert_manager.go` | 新增 | certmagic 封装（New/AddDomain/RemoveDomain/TLSConfig/HTTPChallengeHandler） |
| `server/internal/api/custom_domain_handler.go` | 新增 | REST handler（List/Create/Delete） |
| `server/internal/api/router.go` | 更新 | Options.CertManager + Options.PlatformDomain + routes + HostDomainMiddleware |
| `server/cmd/server/main.go` | 更新 | certManager 初始化 + HTTPS listener(443) + ACME listener(80) |

### 新增依赖
- `github.com/caddyserver/certmagic v0.25.4`（ACME 自动证书）

### 设计决策

| 决策 | 结论 |
|---|---|
| ACME 库 | certmagic（支持 HTTP-01 + TLS-ALPN-01） |
| HTTPS 模式 | 内建 HTTPS（单二进制监听 443 + 80） |
| 证书存储 | certmagic FileStorage |
| 域名管理 | Admin SPA REST API → certmagic.ObtainCertAsync |
| 路由 | HostDomainMiddleware 验证 Host 头 |
| 平台域名 | `PLATFORM_DOMAIN` env 配置，不走 certmagic |

### 验证

- Go build: ✅
- storage tests: ✅ (3.3s)
- api tests: ✅ (9.1s)
- JS tests: admin 204 ✅, visitor-sdk 219 ✅
