# 切片 1l-privacy-gdpr 规格

**状态**:in-progress
**开始**:2026-06-18
**对应审计**:[2026-06-18 全栈深度审计](../audits/2026-06-18-deep-audit.md) §2 P0-5/P0-6
**前置切片**:[1k-security-blockers](./2026-06-18-slice-1k-spec.md)(auth middleware prod 模式就绪)

## Context

1k 修复了部署阻断的认证/授权栈,但 GDPR/CCPA 合规仍是部署阻断:按键监听 + DOM 录像 + IP/UA/地理位置采集全部默认启动无 consent(审计 P0-5),无被遗忘权接口 + GC 只清 1/5 表(审计 P0-6)。在欧盟部署即违反 GDPR Art.7/Art.17/Art.22,最高 4% 全球营收罚款;加州部署违反 CCPA。

本切片补齐 GDPR/CCPA 合规底线。**1k 的 SERVER_ENV=prod 默认 + AuthMiddleware prod 模式已就绪**,本切片在此基础上加 consent 流程 + erasure 接口 + 数据最小化。

## 范围

### In scope

| 项 | 文件 | 内容 |
|---|---|---|
| Consent 存储 | `server/migrations/000005_privacy.up.sql` | `visitor_consents` 表:fingerprint, scope, version, accepted, consented_at, expires_at |
| Consent API | `server/internal/api/privacy.go` | `GET /api/privacy/consent?fingerprint=` + `POST /api/privacy/consent`(公开,无需 auth) |
| Erasure API | 同上 | `DELETE /api/privacy/visitor/:fingerprint`(admin 认证,级联删除) |
| IP 截断 | `server/internal/storage/queries.go` 或新 `privacy/ip.go` | IPv4 → /24,IPv6 → /64,CreateVisitor/CreateSession 入库前应用 |
| GC 扩展 | `server/internal/recording/gc.go` | 增加 `chat_messages` / `co_browsing_commands` / `sessions`(ended_at 早于 threshold)/ `visitors`(last_seen_at 早于 threshold)清理 |
| SDK Consent Banner | `visitor-sdk/src/ui/consentBanner.ts` | 首次访问显示 banner + Accept/Reject 按钮 + 调 `setConsent()` |
| SDK Consent 模式 | `visitor-sdk/src/config.ts` + `index.ts` | `consentMode: 'opt-in' \| 'opt-out' \| 'always-on' \| 'always-off'`,默认 `opt-in`;opt-in 模式下未同意前不启动采集 |
| SDK setConsent API | `visitor-sdk/src/index.ts` | 公开 `mm.setConsent(accepted: boolean)`;首次启动调 `GET consent` 查状态,本地缓存 |
| SDK Co-browsing 横幅 | `visitor-sdk/src/ui/coBrowseBanner.ts` | 运营接管时显示 "运营 X 正在协助您" + 退出按钮;`showCoBrowseBanner` config 默认 true |
| Admin Erasure UI | `admin/src/views/Privacy.vue` | 输入 fingerprint → 点删除 → 调 DELETE → 显示结果 |
| 路由 | `admin/src/router/index.ts` | `/privacy` 路由 |
| i18n | `admin/src/i18n/{zh-CN,en-US}.ts` + SDK messages | 中英双语 consent 文案 + co-browse 横幅 + admin 隐私页 |

### Out of scope(留给后续)

- **Cookie banner**(v1 仅 session cookie,无需 cookie consent)
- **数据跨境传输声明**(法律文档,非代码)
- **隐私政策页面**(部署方提供)
- **数据保留期配置 UI**(env 配置即可)
- **erasure-request self-serve 端点**(本切片仅 admin 代调用)
- **审计日志**(erasure 操作日志,留给 1m-observability)

## 锁定决策表

| # | 决策 | 理由 |
|---|---|---|
| 1 | Consent 默认:**可配置**(默认 opt-in) | GDPR 严格要求 opt-in;但部署方可能选 opt-out(CCPA 友好)或 always-on(内部工具)。`consentMode` config 暴露 |
| 2 | Consent 存储:**PG `visitor_consents` 表** | 持久/可审计/跨设备;支持 GDPR 报告"谁在何时同意/拒绝";Redis 缓存可选但 v1 不引入 |
| 3 | Erasure 触发:**仅 admin 代调用** | 与 1k 认证体系一致(claim ownership 同样 admin-only);self-serve 留给后续切片 |
| 4 | Erasure 范围:**全量级联删除** | GDPR Art.17 要求"删除涉及该数据主体的所有数据";保留业务记录有匿名化遗漏风险 |
| 5 | IP 处理:**截断 IPv4 /24 + IPv6 /64** | GDPR Recital 26 认为截断后的 IP 不再是个人数据;保留地理位置统计价值;erasure 时无需额外处理(未保留原始 IP) |
| 6 | co-browsing 横幅:**默认显示** | GDPR Art.22 自动化决策透明度;`showCoBrowseBanner` config 可关 |
| 7 | 测试:**Go 单测 + e2e 4 场景,目标 🟢** | R2 rubric 全覆盖;e2e 验证 SDK banner 显示 + admin erasure 链路 |

## 涉及代码改动

### DB Migration(1 新建)

**新建** `server/migrations/000005_privacy.up.sql`:
```sql
CREATE TABLE visitor_consents (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    fingerprint VARCHAR(64) NOT NULL,
    scope VARCHAR(32) NOT NULL,         -- 'recording' / 'keyboard' / 'screenshot' / 'all'
    version VARCHAR(16) NOT NULL,       -- 同意书版本,同意条款变更时升级
    accepted BOOLEAN NOT NULL,
    consented_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,             -- 可选,默认 365 天
    UNIQUE(fingerprint, scope, version)
);
CREATE INDEX idx_visitor_consents_fp ON visitor_consents(fingerprint);
```

**新建** `server/migrations/000005_privacy.down.sql`:`DROP TABLE visitor_consents;`

### 后端 Go(预计 7 改/新)

**新建**:
- `server/internal/api/privacy.go` — `PrivacyHandler` 含 3 端点(GET / POST consent / DELETE visitor)
- `server/internal/privacy/ip.go` — `TruncateIP(ip string) string`(IPv4 /24 + IPv6 /64)
- `server/internal/privacy/ip_test.go` — 单测覆盖 IPv4/IPv6/无效/host-only

**修改**:
- `server/internal/storage/queries.go` — 加 `GetConsent`/`UpsertConsent`/`DeleteVisitorCascade`;`CreateVisitor`/`CreateSession` 入库前调 `privacy.TruncateIP`
- `server/internal/recording/gc.go` — 扩展 runOnce 清 `chat_messages`/`co_browsing_commands`/`sessions.ended_at`/`visitors.last_seen_at`
- `server/internal/api/router.go` — 注册 `/api/privacy/*` 路由
- `server/internal/storage/stores.go` — 接口暴露新方法

### 前端 SDK(预计 4 改/新)

**新建**:
- `visitor-sdk/src/ui/consentBanner.ts` — banner DOM 渲染 + Accept/Reject 按钮 + i18n 文案
- `visitor-sdk/src/ui/coBrowseBanner.ts` — co-browsing 接管横幅 + 退出按钮

**修改**:
- `visitor-sdk/src/config.ts` — 加 `consentMode`、`showCoBrowseBanner`、`consentBannerText` config
- `visitor-sdk/src/index.ts` — 启动流程改:
  - 启动时 GET consent 状态(异步)
  - opt-in 模式下未同意前**不**启动 rrweb/keyboard/screenshot
  - 接受 `control_started` 命令时显示 co-browse 横幅
  - 接受 `control_ended` 或 ESC 三连时移除横幅

### 前端 Admin(预计 3 改/新)

**新建**:
- `admin/src/views/Privacy.vue` — erasure UI
- `admin/src/api/privacy.ts` — `deleteVisitor(fingerprint)` API 封装

**修改**:
- `admin/src/router/index.ts` — `/privacy` 路由
- `admin/src/i18n/{zh-CN,en-US}.ts` — 隐私页文案

### E2E(1 新建)

**新建** `e2e/tests/1l-privacy.spec.ts` — 4 场景:
- 场景1:opt-in 模式下 SDK 加载 → 默认不采集 → banner 显示 → 点同意 → rrweb 启动
- 场景2:opt-in 模式 banner 点拒绝 → rrweb 不启动
- 场景3:admin 代访客 DELETE /api/privacy/visitor/:fingerprint → 验证 PG + MinIO + Redis 全清
- 场景4:co-browsing 接管时访客端显示横幅

## Verification

```bash
# 1. Migration 应用
make migrate-up
# 或启动 server 自动跑(SERVER_ENV=dev ADMIN_PASSWORD=xxx ./server)
docker compose exec postgres psql -U mm -d marketing_monitor -c '\d visitor_consents'

# 2. Go 单测
cd server && go test ./internal/privacy/ ./internal/api/ -count=1 -v -race

# 3. IP 截断单测
cd server && go test ./internal/privacy/ -run TestTruncateIP -v

# 4. Erasure 级联测试(集成,需 docker PG)
cd server && go test ./internal/storage/ -run TestDeleteVisitorCascade -v

# 5. SDK banner 构建
pnpm --filter @pinconsole/visitor-sdk build

# 6. e2e
cd e2e && pnpm test 1l-privacy

# 7. Prod 模式验证
SERVER_ENV=prod ADMIN_PASSWORD=strong ./server &
curl -X POST http://localhost:8080/api/auth/login -d '{"email":"admin@...","password":"strong"}' -c /tmp/cookies
curl -X DELETE http://localhost:8080/api/privacy/visitor/some-fingerprint -b /tmp/cookies
```

**预期结果**:
- visitor_consents 表存在
- IPv4 192.168.1.42 → 192.168.1.0;IPv6 2001:db8::1 → 2001:db8:0:0:0:0:0:0(/64)
- SDK 加载默认不采集,opt-in banner 显示
- admin DELETE 后 visitors/sessions/event_blobs/chat_messages/co_browsing_commands 全清
- co-browsing 接管时横幅显示,退出按钮工作

## 深度目标

🟢 verified-deep(R2 rubric):
- ✅ Happy path:consent 同意 → 采集启动;admin DELETE → 数据清空
- ✅ Negative case:consent 拒绝 → 不采集;未认证 DELETE → 401;非 admin → 403
- ✅ 边界:IPv4/IPv6/无效 IP;consent 过期;erasure 不存在的 fingerprint;co-browse 横幅 config off
- ✅ 真实集成:真实 PG/Redis/MinIO 级联删除(非 mock)
- ✅ 可重复运行:`-count=1` 无 flaky

## 估时

solo 全职:2-3 天

## 风险

- **Consent banner UX 干扰业务**:opt-in 默认拒会大幅减少采集率(可能 < 30%)。**缓解**:文档说明 opt-out 模式作为替代,但 GDPR 风险自担
- **IP 截断破坏现有数据**:历史已存 IP(原始)在新代码下不一致。**缓解**:仅新写入截断;历史数据保留(erasure 时一并清)
- **级联删除性能**:大量 event_blobs 删除可能慢。**缓解**:批量 DELETE LIMIT + 后台 worker;v1 接受同步删除 < 10s
- **SDK config 变更破坏部署方**:加新 config 字段需向后兼容。**缓解**:全部默认值与旧行为一致(默认 opt-in 是新行为,但旧 SDK 默认采集,部署方升级时需明确决策)

## Follow-ups

- `1m-observability`:erasure 操作审计日志(LifecycleTracker + audit table)
- `1n-test-depth`:本切片的 e2e 可能与 1k e2e 重叠,统一抽象 helper
- 后续:v1 之后考虑 self-serve erasure-request(访客提交申请 → admin 审批 → 自动 DELETE)
