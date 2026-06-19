# 验证深度判定标准

> 本文件落地 [`CLAUDE.md`](../../CLAUDE.md) 中"工作提示 → 测试深度判定"小节。
> 每个切片(以及未来每个交付单元)按以下三级标定验证深度。深度决定 LLM
> 与人类对该切片实现的可信度假设。

## 1. 三级定义

### 🟢 verified-deep(深度验证)

必须满足以下全部:

1. **正向 e2e 覆盖**:Playwright 或同级测试覆盖**用户可见的真实行为**
   - 不只是 `request.get/post` 调 API endpoint
   - admin / visitor 都通过真实浏览器交互
   - 测试时序模拟真实使用场景(打开 admin → 访客访问 → 订阅 → 触发交互)

2. **断言强度**:测试断言必须验证实际行为,而不是只 `expect(resp.ok()).toBeTruthy()`
   - 例子:鼠标坐标到了、rrweb-player 真渲染了、replay 控制器真响应了、chat 面板真出现消息了

3. **切片特性要求**:
   - **展示/采集类切片**(如 1a 骨架、1b 单向最小、1c rrweb、1d 录像):1 + 2 即可
   - **安全/边界类切片**(如 1e 紧急退出、1f navigate 跨域拒绝、1g 离线消息、1h 认证、1i 反爬):**额外要求至少 1 个负向测试**(401/403/429/拒绝/失败模式)

4. **不允许静默跳过**:`if (!x.length) return;` 类让测试在空状态下静默 pass 的模式自动降级为 🟡

### 🟡 verified-shallow(浅验证)

至少有 e2e/集成测试触达该切片,但存在以下任一:

- 测试只验证 API endpoint(`request.post(...).ok()`),没有真实 UI 流
- 断言强度不足(只 `resp.ok()` 或只检查 console 文本)
- 含静默跳过模式(`if (!x.length) return;`)
- 切片属安全/边界类但缺负向测试

### 🔴 implemented-unverified(已实现未验证)

代码已交付、能编译、可能跑过 smoke,但**无切片级 e2e/集成测试**。

## 2. 升级路径

| 当前 → 目标 | 需补 |
|---|---|
| 🔴 → 🟡 | 加任意触达该切片的 e2e(即使是 API 级 smoke) |
| 🟡 → 🟢 | (a) 替换浅断言为真实行为断言;(b) 若是安全/边界类,补负向测试;(c) 移除静默跳过模式 |
| 🟢 维持 | 任何重构后必须重新确认上述四项仍满足 |

## 2.5 Spec 对照分级(🟢 内部细分,2026-06-19 测试信心审计引入)

🟢 verified-deep 内部按 **spec 对照源** 细分三级,反映判定的可信度:

| Grade | 含义 | 对照源 |
|---|---|---|
| 🟢 **strict** | 所有 spec 决策点都有 strong 测试 | slice spec 文档逐项 |
| 🟢 **aligned** | START/PLAN 决策有 strong 测试,切片目标段有 touched 测试 | START §安全 / PLAN §10 |
| 🟢 **touched** | 切片目标段有测试,但断言强度未严格验证 | impl 报告"目标"段 |

**为什么细分**:

- strict 切片 badge 最稳(spec 决策编号 → 测试 file:line 可追溯)
- aligned 中等(顶层决策有保护,细节未深查)
- touched 最容易降级(下次 reality check 可能发现 gap)

**降级关系**:

- T0/T1 gap 出现 → 🟢(任意级)→ 🔴/🟡
- T2 gap 累积 ≥ 5 → 🟢 touched 持续监控
- T3 gap → 🟡(无论原 strict/aligned/touched)

## 2.6 测试 gap 严重度尺度 T0~T3(2026-06-19 测试信心审计引入)

与 deep-audit 的 P0~P3 **正交**(那里是代码 bug,这里是测试 gap)。两套尺度并存,不互相替代。

| 级别 | 描述 | 典型例子 |
|---|---|---|
| **T0** | critical 路径无测试 or 测试无效 | "1h HttpOnly cookie 未测"、"1k 非 owner command → 403 未测" |
| **T1** | 重要路径无测试 or 仅 happy path | "1l GDPR DELETE 仅测 PG,未测 R2/Redis"、"1d GC worker 未测" |
| **T2** | 次要路径无测试 | "1m slog 格式未测"、"1r SDK i18n keys 未测" |
| **T3** | 测试存在但断言弱 | "1k rate limit 测了 resp.ok 但未测 429"、"v1-e2e 13 处 indirect 覆盖" |

**对应 grade**:

- 🟢 strict/aligned:0 T0/T1/T2/T3
- 🟢 touched:0 T0/T1,可能 T2
- 🟡 verified-shallow:有 T3,无 T0/T1/T2
- 🔴 implemented-unverified:有 T0/T1/T2

## 3. 当前分布(2026-06-19 测试信心审计 + 1ac + 1ac-final + 1ad 后)

详见 [`docs/project-status.md`](../project-status.md) §5。snapshot:

| 切片 | 深度 | 备注 |
|---|---|---|
| 1a 骨架 | 🟢 touched | 结构类,低风险 |
| 1b 单向最小 | 🟡 | 4 T1 留 1ad 续集(TS) |
| 1c rrweb | 🟡 | 1/5 T1 closed(1ad snapshot TTL),4 T1 留续集 |
| 1d 录像归档 | 🔴 | 留 1ad 续集 PG 集成 |
| 1e 双向通道 | 🟢 touched | **1ad 升 🟡 → 🟢**(3/3 T1 + 1e-4 by 1ac) |
| 1f 表单 + 跳转 | 🟡 | 1/2 T1(1ad navigated 结构),1 T1 留续集 |
| 1g 弹窗 + 聊天 | 🔴 | 留 1ad 续集 |
| 1h 认证 + 多运营(后端) | 🟢 touched | **1ac-final 6/6 T0** |
| 1h-ui LoginView + 守卫 | 🟢 touched | 1ac 3/3 T0 |
| 1i 反爬虫 | 🟢 touched | 1ac 1/1 T0 |
| 1j i18n + 部署 + CI | 🟢 aligned | CI 测试覆盖好 |
| 1k 安全阻断栈 | 🟡 | 1ac 8/9 T0 |
| 1l GDPR 合规 | 🟡 | 1ac 6/6 T0 + admin role bug fix |
| 1m 可观测性 | 🟢 touched | **1ad 升 🟡 → 🟢**(3/3 T1 trace 接线) |
| 1n 测试深度 + 文档 | 🟢 touched | 本身是补测切片 |
| 1o 生产硬化 | 🟢 touched | **1ad 升 🟡 → 🟢**(2/2 T1 per-sub cancel) |
| 1p LLM friendly | 🟢 touched | 结构类 |
| 1q 死代码清理 | 🟢 touched | 清理类 |
| 1r i18n + logger 迁移 | 🟡 | SDK i18n keys 留续集(TS) |
| 1s 可观测性深化 | 🟡 | **1ad 升 🔴 → 🟡**(13/13 T1 lifecycle 接线),T0 deep integration 仍开 |
| 1t 测试覆盖补全 | 🟢 strict | 自洽 |
| 1u god files 拆分 | 🟢 touched | 编译期校验 |
| 1v 审计后续修复 | 🟢 touched | 多为 doc-only |
| 1w flagged session | 🟢 touched | **1ad 升 🟡 → 🟢**(4/4 T1 warn+is_flagged) |
| 1x 登录暴力破解 | 🟢 touched | 1ac 1/1 T0 |
| 1y visitor WS rate limit | 🟢 touched | 1ac 2/2 T0 |
| 1z 生产就绪度补全 | 🟢 strict | 自洽 |
| 1aa TS 测试深化 | 🟢 strict | 自洽 |
| 1ab TrustedProxies 加固 | 🟢 strict | 自洽 |
| v1-e2e 全量 e2e | 🟡 | 多数 indirect(留续集) |
| v1-followups 5 个生产 bug fix | 🟡 | 3 fix 无回归测试(留续集) |

> **分布会变化**:本节 snapshot 可能滞后。永远以 `docs/project-status.md` §5 为准。

> **2026-06-19 测试信心审计 + 1ac + 1ac-final + 1ad 进展**:
> - 审计:31 切片 badge 系统性虚标,20 个应降级
> - 1ac + 1ac-final:28/28 T0 关闭 + 2 代码 bug 修复(deleteVisitor admin + operatorWS auth)
> - 1ad:30/40 T1 关闭(13/13 1s + 3/3 1m + 3/3 1e + 4/4 1w + 2/2 1o + 1/2 1f + 1/5 1c)
> - 累计 badge:🟢 ×19 / 🟡 ×6 / 🔴 ×3(1d/1g/1s)
> - 1ad 续集(~10 小时)留 TS + Vue 组件 T1。
>
> 详见 [`audits/2026-06-19-test-confidence-audit.md`](../audits/2026-06-19-test-confidence-audit.md) + [`reports/completed/2026-06-19-slice-1ac-implementation.md`](../reports/completed/2026-06-19-slice-1ac-implementation.md) + [`reports/completed/2026-06-19-slice-1ad-implementation.md`](../reports/completed/2026-06-19-slice-1ad-implementation.md)。

## 4. 与完成报告的关系

每个 `docs/reports/completed/slice-X-implementation.md` 报告顶部都有:

```markdown
> **Verification Depth**: 🟢/🟡/🔴(以 YYYY-MM-DD reality check 为准)
> **报告叙述免责**:[详见模板](../templates/report.md)
```

报告内容是"实施者自述",**不是**"已验证事实"。报告叙述的准确性在 A 阶段补深测时一并 audit。

## 5. 触发重新评级的场景

- 加新 e2e 测试 → 评估是否升级
- 重构某切片代码 → 评估是否需要重新跑全 e2e 守底
- 发现 bug → 评估是否需要降级(说明测试不够)
- reality check → 每次完整审计后更新分布表
