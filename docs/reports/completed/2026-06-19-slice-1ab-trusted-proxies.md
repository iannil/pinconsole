# 切片 1ab-trusted-proxies-hardening — Spec + Implementation

**切片编号**:1ab
**类型**:安全
**创建时间**:2026-06-19
**完成时间**:2026-06-19
**状态**:completed
**深度**:🟢 verified-deep(Go 单测 5 场景 + integration test 6 场景,断言真实 ClientIP 行为)
**关联**:[deep-audit P1-5 已关闭](../../audits/2026-06-18-deep-audit.md)、[1o 生产硬化](./2026-06-18-slice-1o-implementation.md)

## Context

deep-audit P1-5 指出 `gin.SetTrustedProxies()` 未配置时,rate limit / login throttle 等 IP-based 功能可被 `X-Forwarded-For` 伪造绕过,或在反代部署下共用单 IP 预算导致功能退化。1o 接入 env var 但 validate 不校验、0 单测、0 e2e。本切片补齐。

## 决策表

| # | 决策点 | 选定 | 理由 |
|---|---|---|---|
| 1 | 检测反代方式 | 显式 env var `BEHIND_REVERSE_PROXY` | 推断不可靠,运行时中间件复杂,显式声明最稳定 |
| 2 | validate 行为 | fail-fast reject | 与 1k/1z fail-secure 一致 |
| 3 | 直接暴露 + TrustedProxies set | silent allow | 用户防御性配置,无害(XFF 被忽略) |
| 4 | CIDR 格式校验 | 严格 `net.ParseCIDR` + `net.ParseIP` | 防 typo |
| 5 | 测试覆盖 | Go 单测 + Go integration test | integration test 用 httptest 验证 ClientIP 实际行为,比 Playwright 更可靠 |

## Changes Delivered

### 代码改动

- ✅ **`server/internal/config/config.go`**:
  - 新增 `BehindReverseProxy bool env:"BEHIND_REVERSE_PROXY" envDefault:"false"`
  - `validate()` 加 4 组合校验:BEHIND_REVERSE_PROXY=true 时 TrustedProxies 必填且 CIDR 合法
  - 新增 `validateProxyCIDRList()` helper:`net.ParseCIDR` + `net.ParseIP` 双格式 + whitespace 容忍
- ✅ **`.env.example`**:新增 `BEHIND_REVERSE_PROXY=false` 文档(含使用说明)

### Go 测试

- ✅ **`server/internal/config/config_test.go`**(扩展)— 5 新测试 + 1 表驱动:
  - `TestLoad_DirectExposure_Default_NoTrustedProxies` — 默认部署 OK
  - `TestLoad_BehindProxy_RequiresTrustedProxies` — 反代忘配 reject
  - `TestLoad_BehindProxy_ValidCIDRs_OK` — 反代 + 有效 CIDR OK
  - `TestLoad_BehindProxy_InvalidCIDR_Rejects`(表驱动 4 case) — not enough octets / bad prefix / non-IP / trailing garbage
  - `TestLoad_DirectExposure_TrustedProxiesSet_SilentAllow` — 直接 + set silent OK
  - `TestValidateProxyCIDRList_Formats`(表驱动 10 case) — empty / CIDR / IP / IPv6 / mixed / whitespace / bad prefix / non-IP / trailing comma
- ✅ **`server/internal/api/router_trusted_proxies_test.go`**(新建)— 6 integration test:
  - `TestTrustedProxies_Empty_XFFIgnored` — 不信任任何反代,XFF 被忽略
  - `TestTrustedProxies_TrustedProxy_XFFParsed` — 信任 + RemoteAddr 匹配,XFF 被采用
  - `TestTrustedProxies_UntrustedProxy_XFFIgnored` — 信任范围外,XFF 被忽略
  - `TestTrustedProxies_MultiHop_XFFChainParsed` — 多 hop 链,gin 从右向左走,返回右起第一个非信任
  - `TestTrustedProxies_FullChainTrusted_LeftmostClientReturned` — 整链信任,返回最左端 client
  - `TestTrustedProxies_ConfigDrivenIntegration` — 端到端:从 cfg 字符串解析 → SetTrustedProxies → 3 场景 XFF 验证

## Verification

```bash
cd server

# 单测全绿
go test ./internal/config/ -count=1     # config 校验
go test ./internal/api/ -count=1        # 含新 router_trusted_proxies_test.go
go test ./... -count=1                  # 全 12 包 ALL PASS

# 类型/静态检查
go vet ./...                            # clean
go build -tags dev -o /tmp/mm-server ./cmd/server   # 编译 OK
go build -tags release -o /tmp/mm-server ./cmd/server  # release 编译 OK
```

**实测**(2026-06-19):
- config 包:新增 6 测试 + 已有测试全绿
- api 包:新增 6 测试 + 已有测试全绿
- 12 个 Go 包 ALL PASS

## 深度判定

🟢 verified-deep:
- 每条 validate() reject 路径都有负向测试
- CIDR 格式校验覆盖 CIDR / bare IP / IPv6 / 多种 invalid 格式
- integration test 用真实 httptest + gin engine,验证实际 ClientIP() 行为(非 mock)
- 表驱动测试覆盖等价类边界(空 / 单 CIDR / 多 CIDR / 混合 / trailing comma)
- 不允许 vacuous assertion

## Follow-ups

无 — P1-5 完全关闭。剩余 post-v1 候选见 [`IMPLEMENTATION_PLAN.md`](../../../IMPLEMENTATION_PLAN.md)。

## Notes

- **gin XFF 解析方向**:从右向左走 XFF chain,遇到非信任 IP 就停。这是 gin 的防伪造策略:即使 XFF 里有 client IP,也不轻易相信。要拿到最左端 client IP 必须信任整条链
- **BEHIND_REVERSE_PROXY 默认 false**:向后兼容(现有部署行为不变)。仅当用户显式 opt-in 才触发严格校验
- **CIDR 校验时机**:validate() 在 Load() 内调用,启动时 fail-fast 拒绝,不让配置错误的 server 跑起来
- **配置一致性原则**:BEHIND_REVERSE_PROXY=true + 空 TrustedProxies 是反代忘配的典型场景,reject;直接暴露 + 设了 TrustedProxies 是无害的过度配置,silent allow
