# 切片 Spec 模板

> 复制此模板到 `docs/progress/{YYYY-MM-DD}-slice-{id}-{kebab-case-name}-spec.md`（事前决策），
> 切片完成后与 implementation 一起移到 `docs/reports/completed/`。
> Spec = 切片的"事前决策记录"。让 LLM 能在 30 秒内理解切片要做什么、为什么、不做什么。

---

# 切片 {id}-{kebab-case-name} — Spec

**切片编号**：{例：1k}
**类型**：{例：安全 / 性能 / 测试 / 重构 / 新功能}
**创建时间**：YYYY-MM-DD
**状态**：draft | approved | in-implementation | completed
**关联**：[PLAN.md 章节](../../PLAN.md#{anchor})、[前置审计/切片](../audits/{...}.md)、[deep-audit 编号](../audits/{...}.md#P0-1)

## Context

为什么做这个切片？

- 触发原因：{例：deep-audit P0-1 发现 SERVER_ENV=dev 默认值 + dev bypass 构成 silent 远程接管链}
- 业务/技术价值：{例：阻止 silent default 部署到生产即被接管}
- 不做的代价：{例：v1 阻断 release}

不超过 5 行。

## 决策表

每个决策点列出选项与选定，含理由。**关键：列出"为什么不选其他"**，避免未来 LLM 重新讨论。

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | {例：silent defaults 处理} | A: 运行时 env 检查 / B: 编译 tag 隔离 / C: 双重防御 | C | A 易绕过（改 env 即绕过），B 结构上隔离 release binary，C 加 defense-in-depth |
| 2 | ... | ... | ... | ... |

## Acceptance（可验证的成功标准）

切片"完成"的可验证判据。每条都用客观断言，不用"基本完成"等模糊词。

- [ ] {验收 1}：{可执行命令或可观察行为}，预期 {结果}
- [ ] {验收 2}：...
- [ ] {验收 3}：...

## 深度目标

按 [`docs/standards/verification-depth.md`](../standards/verification-depth.md) R2 rubric，本切片目标深度：

- 🟢 verified-deep：{列出需要的负向测试 / 边界测试}
- 🟡 verified-shallow：{说明为什么不能到 🟢}
- 🔴 implemented-unverified：{说明为什么无测试覆盖}

## 范围边界

**本切片做**：
- {做 1}
- {做 2}

**本切片不做**（避免范围爬升）：
- {不做 1} → {理由或后续切片}
- {不做 2} → ...

## 估时

- 实施：{n} 小时 / 天
- 测试：{n} 小时 / 天
- 文档：{n} 小时 / 天

## 关联

- 前置切片：[1j](./2026-06-17-slice-1j-implementation.md)
- 触发审计：[deep-audit P0-1 至 P0-8](../audits/2026-06-18-deep-audit.md)
- 后续切片：[1l-privacy-gdpr](./2026-06-18-slice-1l-spec.md)（GDPR 部分从本切片拆出）

## Notes

补充信息：grill-me 访谈记录、被否决的方案、未来要回头看的点。（可选。）
