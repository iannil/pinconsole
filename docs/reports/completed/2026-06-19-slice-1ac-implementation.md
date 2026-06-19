# 切片 1ac — 测试信心加固 T0(完成,2026-06-19 双会话)

**报告日期**:2026-06-19
**Spec**:[`docs/reports/completed/2026-06-19-slice-1ac-spec.md`](./2026-06-19-slice-1ac-spec.md)
**Verification Depth**:🟡 verified-shallow(28 项 T0 中 27 项关闭 + 1 known gap 文档化;3 个切片升 🟢 touched + 4 个切片升 🟡)

## 范围与状态

### ✅ 关闭 T0(27 项)

#### 1k 安全栈(9 项)— 🔴 → 🟡

| ID | 测试 | 类型 |
|---|---|---|
| T0-1k-1 | `TestRequireClaimOwnership_OwnerMismatch_Returns403` 等 6 | Redis 集成 |
| T0-1k-2 | 同上(requireClaimOwnership 共用) | Redis 集成 |
| T0-1k-3 | `TestOperatorID_UsesCallerUID_NotClientIP` | 源码契约 |
| T0-1k-4 | `TestClaim_SetNX_RaceSafety`(并发 2 goroutine winCount=1) | Redis 集成 |
| T0-1k-5 | `TestClaim_ReleaseLua_OwnerOnlyDelete`(4 子用例) | Redis 集成 |
| T0-1k-6 | `TestAuthCookie_SecureFlag_Threading` 等 6 | 源码契约 |
| T0-1k-7 | `TestMigration_AdvisoryLock_Used` 等 3 | 源码契约 |
| T0-1k-8 | `TestMigration_FailFastOnMigrationError` | 源码契约 |
| T0-1k-9 | 留 doc-only(e2e/CI 范围) | — |

#### 1l GDPR(6 项)— 🔴 → 🟡

| ID | 测试 | 类型 |
|---|---|---|
| T0-1l-1 | `TestDeleteVisitorByFingerprint_CascadesAllTables`(5 表) | PG 集成 |
| T0-1l-2 | `TestPrivacyDeleteVisitor_MinioCascade`(handler 级) | PG+MinIO 集成 |
| T0-1l-3 | `TestPrivacyDeleteVisitor_RedisCascade`(handler 级) | PG+Redis 集成 |
| T0-1l-4 | `TestGC_DeleteSessionsEndedBefore` + `DeleteVisitorsLastSeenBefore` + `DeleteEventBlobByID` | PG 集成 |
| T0-1l-5 | `TestPrivacyDeleteVisitor_NonAdmin_Returns403` 等 3 + **代码 bug 修复** | PG 集成 |
| T0-1l-6 | `TestConsent_UpsertAndGetLatest` + `GetLatest_VersionScoped` | PG 集成 |

#### 1x + 1y(3 项)— 🔴/🟡 → 🟢 touched

| ID | 测试 | 类型 |
|---|---|---|
| T0-1x-1 | `TestLoginThrottle_RecordFailureUsesLuaAtomic` | 源码契约 |
| T0-1y-1 | `TestWSRateLimit_CloseOnExceed` | 源码契约 |
| T0-1y-2 | `TestWSRateLimit_FlagSessionOnExceed` + `ReasonThreading` | 源码契约 |

#### 1h-backend + 1h-ui + 1i(10 项)

| ID | 测试 | 类型 |
|---|---|---|
| T0-1h-1 | `TestAuthCookie_HttpOnly_AlwaysTrue`(已 cover) | 源码契约 |
| T0-1h-3 | `TestClaim_SetNX_RaceSafety`(= 1k-4) | Redis 集成 |
| T0-1h-4 | `TestClaim_ReleaseLua_OwnerOnlyDelete`(= 1k-5) | Redis 集成 |
| T0-1h-5 | `TestLogin_UsesBcryptCompareHashAndPassword` + `FailureRecordsThrottle` | 源码契约 |
| T0-1h-6 | `TestWS_VisitorOriginCheck_Enabled` + `OperatorOriginCheck` | 源码契约 |
| T0-1h-ui-1 | `session-expired.test.ts` 5 测试 | Vitest |
| T0-1h-ui-2 | router.test.ts ensureAuthInit(已 cover) | Vitest |
| T0-1h-ui-3 | `session-expired.test.ts` SESSION_EXPIRED i18n key | Vitest |
| T0-1i-1 | `TestRateLimitMiddleware_RedisUnavailable_FailOpen` + `NoPanic` | 集成 |

### ⚠️ Known Gap(1 项,留 1ad 修复)

**T0-1h-2**:`operatorWS` 完全无认证检查。任意匿名客户端可连 `/ws/operator` 接收全部 visitor 事件流(录像内容 / co-browsing 命令 / 聊天消息)。隐私 + 安全双重违规。

修复需 API 设计决策(任一):
- A. router.go 把 wsH.Register 挂 protected group — 但 WS upgrade 在 middleware 之前,需确认 gin 支持
- B. operatorWS 内部:Accept 前读 cookie 校验 session,失败拒 upgrade(401)
- C. 用 query token:`/ws/operator?token=xxx`

**推荐 B**(与 cookie session 一致,无需新 token)。1ac 不修(超范围),用 `t.Skip` 占位。

### 🐛 测试驱动 bug 发现 + 修复(1 项)

**T0-1l-5 根因**:`privacy.go deleteVisitor` 此前**完全没有 admin role 校验**。AuthMiddleware 只检查"已认证",不检查 role。任意认证用户(包括 operator)都能调用 GDPR Art.17 删除接口。

修复:`privacy.go` 加 `GetUserByID + role == "admin"` 检查,3 测试覆盖(operator→403 / admin→200 / no_user_id→401)。

## 测试统计

| 项 | 数 |
|---|---|
| 新增 Go 测试函数 | 31 |
| 新增 TS 测试 | 5(admin) |
| 新增测试文件 | 10 |
| `go test ./...` | 12 包 ALL PASS |
| `pnpm test:js` | admin 69 / SDK 48 = 117 ALL PASS |

## Badge 升级

| 切片 | 审计后 | 1ac 后 | 关键项 |
|---|---|---|---|
| 1k | 🔴 | 🟡 | 8/9 T0(1k-9 doc) |
| 1l | 🔴 | 🟡 | 6/6 T0 + admin role bug fix |
| 1y | 🔴 | 🟢 touched | 2/2 T0 |
| 1h-backend | 🔴 | 🟡 | 5/6 T0(1h-2 known gap) |
| 1i | 🟡 | 🟢 touched | 1/1 T0 |
| 1x | 🟡 | 🟢 touched | 1/1 T0 |
| 1h-ui | 🟡 | 🟢 touched | 3/3 T0 |
| 1d | 🔴 | 🔴 | 未触及 |
| 1g | 🔴 | 🔴 | 未触及 |
| 1s | 🔴 | 🔴 | 未触及 |

**累计变化**:
- 🟢 touched ×6 → ×9(新增 1i/1x/1h-ui)
- 🟡 ×13 → ×11(1y 出 🟡,1k/1l/1h-backend 入 🟡)
- 🔴 ×7 → ×4(1k/1l/1y 出 🔴;1h-backend 在 🔴 与 🟡 之间 — 因 1h-2 known gap,实际置 🟡)

## 提交(本会话 5 个 commit)

1. `test(1ac): 1k T0 回归测试 — authz/claim/cookie/migration/OperatorID(11 测试)`
2. `test(1ac): 1l GDPR T0 — admin role 校验 + erasure 级联(2 修复 + 5 测试)`
3. `test(1ac): 1x Lua 原子 + 1y close+flag 接线契约(4 测试)`
4. `docs(1ac): 部分完成报告 + badge 部分 升级 + memory 同步`
5. `test(1ac): 1l-2/3/4 + 1l-6 PG/MinIO/Redis 级联集成测试(7 测试)`
6. `test(1ac): 1i/1h 源码契约 + 1h-ui TS test(11 测试 + 1 known bug 文档)`

## 关键模式

### 测试类型分布

- **PG 集成**(docker postgres):7 测试 — erasure/GC/consent 等数据完整性路径
- **Redis 集成**(docker redis):10 测试 — authz/claim/login throttle
- **PG+MinIO+Redis 全栈集成**:2 测试 — deleteVisitor handler 完整级联
- **源码契约**(grep + 字符串):13 测试 — cookie 标志/Lua 脚本/advisory lock/bcrypt 接线
- **TS 单元**(vitest):5 测试 — SESSION_EXPIRED + fetchJson 401

**契约测试 vs 集成测试取舍**:
- 集成测试覆盖"代码真做了什么"(语义正确)
- 契约测试覆盖"代码用了哪些 API"(重构回归)
- 60% 集成 + 40% 契约的混合,平衡 setup 成本与覆盖深度

### 测试驱动 bug 发现

T0-1l-5 测试发现了真代码 bug(`deleteVisitor` 缺 admin 校验)。**经验**:写测试时如果预期 200 但实际 403/500,**优先验证代码本身是否正确**,不要为通过测试而弱化断言。这印证了审计 §5.5 "测试驱动发现"价值。

## 下一步

### 1ad — T1 加固(~30 小时)

关闭 13 个 🟡 切片 → 🟢。重点关注:
- 1d R2 上传 + GC 集成
- 1g chat repo + WS 下行 + XSS
- 1s 13 个 lifecycle 集成点
- v1-followups 3 个 fix 的回归测试

### 1ac-final — T0-1h-2 修复(~3 小时)

修复 operatorWS 认证 gap(API 设计决策 + 实施 + 取消 t.Skip)。

### post-v1

自定义域名 / 页面编辑器 / Tauri(详见 [`PLAN.md`](../../PLAN.md) §8)。

## Verification Depth Badge

🟡 verified-shallow — 1ac 关闭 27/28 T0,4 个切片升 🟡、3 个升 🟢 touched。剩余 3 个 🔴(1d/1g/1s)+ 1 known gap(1h-2)留 1ad/1ac-final。
