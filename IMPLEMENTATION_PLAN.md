# IMPLEMENTATION_PLAN — 当前正在做什么

> CLAUDE.md "项目指南"段要求:通过 IMPLEMENTATION_PLAN.md 让模型理解当前正在做什么、边界、下一步。
> 本文件 rolling 更新,每次开始新切片时改写;完成后归档到 `docs/reports/completed/`。

**当前状态**:v1 主干完全收口 + 1aa TS 测试深化 + 1ab TrustedProxies 加固 + 1ae 测试健康度加固完成(test-health-audit 9 项 P0+P1 修复)
**最后更新**:2026-06-19

## 当前焦点

**无活跃切片**。deep-audit 13 个 P0 + 关键 P1(P1-5 TrustedProxies) + test-health-audit 9 项全部闭环。等待用户决定 post-v1 路线。

> **2026-06-19 test-health-audit 结果**:对 1ac+1ad 68 闭包做 4 维度审计(D1 badge 准确性 / D2 弱断言 / D3 mutation / D4 flakiness),整体 verdict 🔴。1ae 关闭 9 项 P0+P1(R1 operatorWS 行为级 + R2 erasure CASCADE 隔离 + R3a-e 5 项源码契约升级 + R4 e2e flaky 修复 + R5 虚标修复),mutation score 71.4%→100%, e2e flaky 20%→0%。整体 verdict 升 🔴→🟡。详见 [`docs/audits/2026-06-19-test-health-audit.md`](./docs/audits/2026-06-19-test-health-audit.md)。剩余 backlog:R3 ~30 项 + R6/R7/R8 留 post-v1。

## v1 已交付切片(完整清单)

图例:🟢 verified-deep / 🟡 verified-shallow / 🔴 implemented-unverified

| 切片 | 内容 | 深度 |
|---|---|---|
| 1a | 仓库骨架 | 🟢 |
| 1b | 单向最小(SDK → WS → hub → admin 实时) | 🟢 |
| 1c | rrweb 接入 | 🟢 |
| 1d | 录像归档(MinIO + PG 元数据) | 🟢 |
| 1e | 双向通道(operator overlay → SDK 执行) | 🟢 |
| 1f | 表单代填 + 跳转接管 + 跨页面会话续接 | 🟢 |
| 1g | 弹窗 + 双向聊天 | 🟢 |
| 1h | 认证 + 多运营(后端) | 🔴 spec partial |
| 1h-ui | admin LoginView + Vue Router 守卫 | 🟢 |
| 1i | 反爬虫(rate limit + UA + behavior + fingerprint) | 🟢 |
| 1j | i18n 中英双语 + 部署 + CI | 🟢 |
| 1k | 安全阻断栈(8 个 P0 修复) | 🟢 |
| 1l | GDPR 合规(consent + erasure + IP 截断 + co-browse 横幅) | 🟢 |
| 1m | 可观测性(LifecycleTracker + WS trace_id) | 🟢 |
| 1n | 测试深度 + 文档虚标修复 | 🟢 |
| 1o | 生产硬化(TrustedProxies + WS timeout + flushSession tx + goroutine 泄漏) | 🟢 |
| 1p | LLM friendly(proto 共享 + IMPLEMENTATION_PLAN + change-safety) | 🟢 |
| 1q | 死代码 + 重复清理 | 🟢 |
| 1r | i18n + logger 迁移(admin utils + SDK 22 处 console.*) | 🟢 |
| 1s | 可观测性深化(LifecycleTracker 接入关键路径) | 🟢 |
| 1t | 测试覆盖补全(logging + storage + privacy + migrations) | 🟢 |
| 1u | god files 拆分(queries.go 771 LOC → 10 文件) | 🟢 |
| 1v | 审计后续修复(migrator 统一 + GDPR DELETE + e2e webServer) | 🟢 |
| 1w | flagged session 接入(listSessions + operatorWS + replay) | 🟢 |
| 1x | 登录暴力破解防护(Redis 计数器 + 锁定 15 分钟) | 🟢 |
| 1y | visitor WS rate limit(滑动窗口 10s/500 envelope/50 MiB) | 🟢 |
| 1z | 生产就绪度补全(i18n `@` + trace_id 端到端 + 连接池 + fail-secure) | 🟢 |
| v1-e2e | 全量 e2e acceptance + 6 regression spec + 4 production bug 修复 | 🟢 |
| v1-followups | e2e 后 5 个生产 bug fix(co-browse + visitor-sdk + v1-replay × 3) | 🟢 |
| 1aa | TS 测试深化(admin 64 + SDK 48,累计 112) | 🟢 |
| 1ab | TrustedProxies 加固(P1-5,BEHIND_REVERSE_PROXY env + validate fail-fast) | 🟢 |
| 1ae | 测试健康度加固(9 项 P0+P1:muation score 71.4%→100%, e2e flaky 20%→0%) | 🟢 |

**累计**:🟢 ×32 / 🔴 ×1(1h-backend spec partial)

完整深度判定与每切片报告见 [`docs/project-status.md`](./docs/project-status.md) §5 + [`docs/reports/completed/`](./docs/reports/completed/)。

## 下一步候选(按优先级)

### 1. post-v1 路线(详见 PLAN.md §8)

- **自定义域名**:DNS 验证 + Let's Encrypt ACME + Host-header 路由(1-2 周)
- **页面编辑器**:拖拽 / 低代码 / JSON schema → Go 模板渲染(切片 2-3)
- **Tauri 桌面端**:Win + Mac,复用 admin SPA(1 个月)
- **反爬加固**:CAPTCHA + honeypot + 动态类名/ID(2-3 周)

## 决策原则

- 任何架构层工作先读 [`PLAN.md`](./PLAN.md)(架构事实来源)
- 任何产品层冲突以 [`START.md`](./START.md) 为准
- 切片流程:grill-me → spec → impl → 验收 → spec+impl 移到 `docs/reports/completed/`
- 范围控制严格:不做多租户、不做计费、不做注册流(CLAUDE.md 硬约束)
- 安全/反爬虫/GDPR 是一等公民(START.md 明确)
- 测试深度判定遵循 [`docs/standards/verification-depth.md`](./docs/standards/verification-depth.md)(R2 rubric)

## 历史切片(已归档)

详见 [`docs/reports/completed/`](./docs/reports/completed/) — 每切片 spec + implementation 两文件 + v1-slice-plan 总览 + v1-e2e-acceptance + v1-followups。
