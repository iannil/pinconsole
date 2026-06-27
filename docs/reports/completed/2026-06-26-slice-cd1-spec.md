# cd-1 自定义域名：后端基础设施

**切片编号**：cd-1（Custom Domain v1 — Backend）
**类型**：新功能后端层（migration + repo + certmagic 集成 + middleware + config）
**创建时间**：2026-06-26
**状态**：in_progress
**关联**：[PLAN.md §8 post-v1 #2](../../PLAN.md)

## Context

通过 page-editor 访客可以创建落地页（SSR 渲染），但只能通过平台域名访问。CD-1
允许用户将自有域名绑定到平台，实现 HTTPS + 自定义域名落地页。后续切片 cd-2 补
Admin UI，cd-3 补 e2e 验证。

## 范围

### 做

1. **PG migration `000009_custom_domains`** — custom_domains 表
2. **`server/internal/storage/custom_domain_repo.go`** — CRUD + 域名验证
3. **`server/internal/cert/cert_manager.go`** — certmagic 集成（自动证书签发/续签）
4. **`server/config.go` 扩展** — ACME 配置（email、data_dir、staging 开关）
5. **`server/cmd/server/main.go`** — 启动 HTTPS listener + HTTP-01 challenge handler
6. **`server/internal/api/router.go`** — Host-header 路由 middleware
7. **`server/internal/api/custom_domain_handler.go`** — admin REST API（List/Create/Delete）

### 不做

- Admin SPA 界面（cd-2）
- Playwright e2e（cd-3）
- 平台域名自身 HTTPS（继续由反向代理处理，或通过同样 certmagic 自动处理）
- DNS-01 challenge（post-v2，仅 HTTP-01 + TLS-ALPN-01）
- 通配符域名

## 设计决策

### 存储方案
- **表**：`custom_domains`（单表，无需关联其他实体）
- **certmagic 缓存**：文件缓存（`data_dir/certmagic/`），单节点足够
- **证书状态跟踪**：DB 中 `cert_status` 字段（pending/active/failed），certmagic 文件状态作为事实来源

### ACME 策略
- **库**：`github.com/caddyserver/certmagic`
- **Challenge**：HTTP-01（默认，端口 80） + TLS-ALPN-01（端口 443 fallback）
- **Staging**：开发期用 Let's Encrypt staging endpoint（防 rate limit）
- **动态域名**：启动时从 DB 加载已激活域名；admin 新增后调用 `ObtainCertAsync`

### 网络架构
- **端口 443**：主 HTTPS listener（certmagic 提供 TLSConfig）
- **端口 80**：HTTP-01 challenge handler + 301 重定向到 HTTPS
- **平台域名**：仍可通过原端口访问，或同样走 certmagic（用户一键绑定）

### 路由
- **自定义域名** → Host-header middleware 验证 → 同一条路由链
- **未知域名** → 404
- **平台域名不走 certmagic**（配置 `PLATFORM_DOMAIN` 排除）

## API 设计

### `GET /api/custom-domains`（admin）
列出当前 tenant 的所有自定义域名。

### `POST /api/custom-domains`（admin）
添加新域名。body: `{ "domain": "example.com" }`

响应：
- 201：已创建，cert_status = "pending"（certmagic 异步签发中）
- 400：域名格式无效/已存在
- 502：certmagic ObtainCert 同步失败

### `DELETE /api/custom-domains/:id`（admin）
删除域名（DB 删除 + certmagic 不做 revoked，仅停止管理）。

### `GET /api/custom-domains/:id/verify-status`（admin）
返回当前证书状态（pending/active/failed + 失败原因）。

## 实现步骤

1. 扩展 Config（ACME Email / DataDir / Staging / PlatformDomain）
2. PG migration 000009
3. storage/custom_domain_repo.go + test
4. internal/cert/cert_manager.go（certmagic wrapper）
5. api/custom_domain_handler.go（REST）
6. api/router.go（Host-header middleware 注册）
7. cmd/server/main.go（双端口 listener + certmagic 启动）
