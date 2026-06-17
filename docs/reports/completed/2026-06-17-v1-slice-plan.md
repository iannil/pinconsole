# v1 切片规划完成

**状态**：completed
**完成时间**：2026-06-17
**对应 progress**：[`docs/progress/2026-06-17-v1-slice-plan.md`](../../progress/2026-06-17-v1-slice-plan.md)

## Summary

通过 grill-me 28 轮访谈完成 v1 切片规划，产出 `PLAN.md` 作为架构事实来源；CLAUDE.md 增加架构决策索引与工作提示。规划覆盖 16 项主决策与 12 项次决策，含 v1 切片 10 个子任务的实施顺序与估时。

## Changes Delivered

- ✅ 写入 [`PLAN.md`](../../../PLAN.md)（8KB，10 节）
  - §1-§2 项目定位与 v1 切片范围
  - §3 架构骨架（5 项核心决策）
  - §4 技术栈（后端 / 前端 / SDK）
  - §5 关键技术决策（11 项）
  - §6 仓库布局提案
  - §7 v1 切片实施顺序（10 个子切片 1a-1j）
  - §8 v1 之后的切片路线图
  - §9 未敲定的实施层细节
  - §10 已识别的风险
- ✅ 更新 [`CLAUDE.md`](../../../CLAUDE.md)
  - 事实来源优先级（PLAN > START > CLAUDE.md）
  - 已锁定的架构决策清单（14 项，禁止重新讨论）
  - 实施顺序指引（始终从下一个子切片开始）
  - 8 条工作提示（不扩大范围 / 不建多租户 / 不引用不存在的命令 等）

## Verification

```bash
# 1. PLAN.md 存在且非空
test -s PLAN.md && echo "PLAN.md OK"

# 2. PLAN.md 含必要章节
grep -E '^## (1|2|3|4|5|6|7|8|9|10)\.' PLAN.md | wc -l   # 期望 10

# 3. CLAUDE.md 含锁定决策清单
grep -c '已锁定的架构决策' CLAUDE.md   # 期望 ≥1

# 4. CLAUDE.md 含事实来源优先级
grep -c '事实来源优先级' CLAUDE.md   # 期望 ≥1
```

**预期结果**：所有命令均输出预期值，无错误。

## Follow-ups

- 启动切片 1a（仓库骨架）→ 新建 `docs/progress/{date}-slice-1a-skeleton.md`
- 切片 1a 完成后，更新 [`docs/project-status.md`](../../project-status.md) 的 v1 切片状态表
- 在 [`docs/project-status.md`](../../project-status.md) §当前阶段 标注进展

## Notes

### 关键事实快照（供后续 LLM 快速回顾）

- **License**：AGPL-3.0（防止云厂商 SaaS 化）
- **租户模型**：单租户部署 + schema 预留 `tenant_id`（不做多租户 SaaS）
- **后端**：Go + Gin + coder/websocket + 自定义 hub（不用 Go-Zero / gorilla / Centrifugo / melody）
- **存储**：PostgreSQL（元数据）+ Redis（presence/限流）+ MinIO（rrweb 事件 + 选择性截图）
- **co-browsing**：rrweb 双向 + 节点 ID 选择器 + 防抖 300ms 代填
- **截图**：选择性（仅 canvas/WebGL/跨域 iframe 触发，1fps WebP q70）
- **认证**：Email/password + bcrypt + HttpOnly cookie + MFA 可选
- **多运营**：1:1 锁定（claim/release）
- **可观测**：仅 slog 结构化日志（暂不加 metrics/tracing/Sentry）
- **域名**：v1 仅平台域名
- **浏览器**：Modern evergreen desktop + mobile 访客，运营仅桌面

完整决策与理由见 [`PLAN.md`](../../../PLAN.md)。
