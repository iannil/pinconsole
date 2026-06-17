# v1 切片规划（grill-me 访谈 → PLAN.md）

**状态**：completed
**开始**：2026-06-17
**完成**：2026-06-17
**关联**：[`PLAN.md`](../../PLAN.md)（v1 架构事实来源）、[`START.md`](../../START.md)（产品事实来源）

## Context

仓库初始仅有 `START.md`（竞品招募帖原文 + 技术分析）。用户希望构建竞品的**开源替代品**（不考虑客户与销售），需要先把模糊的产品需求转化为可执行的架构决策与切片拆分。

采用 `grill-me` 访谈法：从最上游决策往下，逐项确认推荐方案。28 轮问答覆盖架构骨架、技术栈、co-browsing 技术路径、反爬虫、license 等关键决策。

## Changes

- ✅ 用 `grill-me` 完成架构访谈，逐项确认 16 项主决策 + 12 项次决策
- ✅ 写入 [`PLAN.md`](../../PLAN.md)（事实来源，详见 §3-§10）
- ✅ 在 [`CLAUDE.md`](../../CLAUDE.md) 增加"已锁定的架构决策"清单与"给后续 Claude 的工作提示"
- ✅ 用户进一步扩展 CLAUDE.md（加入文档约定、记忆系统、可观测性、LLM Friendly 等通用规范）

## Status

规划完成。PLAN.md 是 v1 切片的事实来源。

实施尚未启动——切片 1a 待启动（见 PLAN.md §7）。

## Next

下一步是启动切片 1a（仓库骨架）：

1. 初始化 Go module + Vue3 项目 + TypeScript SDK 项目（monorepo）
2. 创建 `docker-compose.yml`（Go + PG + Redis + MinIO）
3. 打通 hello world 端到端：SDK 鼠标事件 → WebSocket → admin 实时显示
4. 完成 1a 后，新建 `docs/progress/{date}-slice-1a-skeleton.md` 记录

详见 [`docs/project-status.md`](../project-status.md) §下一步动作。

## Blockers

无。

## Notes

### 决策时间线（grill-me 28 问的依赖顺序）

按依赖树组织，先问最上游：

1. **范围层**：v1 切片范围（完整对标）→ 切片策略（纵向切片）→ 租户模型（单租户+预留）
2. **采集层**：SDK 采集范围（全量）→ 实时管道（hub-and-spoke）
3. **结构层**：仓库结构（Monorepo Go embed）→ 运营端形态（Web admin）→ 运营动作范围（完整互动）
4. **持久化层**：录制范围（实时+完整）→ 对象存储（MinIO）→ License（AGPL-3.0）
5. **传输层**：WebSocket 实现（coder/websocket + 自定义 hub）→ 多运营并发（1:1 锁）
6. **质量层**：反爬深度（中等）→ 域名（v1 平台域名）→ 后端框架（Gin）
7. **co-browsing 层**：技术路径（rrweb 双向）→ 选择器（rrweb 节点 ID）→ 代填粒度（防抖 300ms）
8. **细节层**：截图策略（选择性）→ 保留策略（30 天可配）→ Replayer（rrweb-player + overlay）→ i18n（中英 day 1）→ 浏览器矩阵（Desktop + Mobile 访客）→ CI/CD（GitHub Actions）→ 可观测（仅日志）→ 认证（Email/password + Cookie）→ SDK 分发（/sdk.js endpoint）

每个问题都用 `AskUserQuestion` 提出，含推荐答案与理由。用户大多数选了推荐，少数选了"max scope"（如"完整对标"、"全量采集"、"完整互动"）。

### 估时（参考）

- v1 切片（1a-1j）：solo 全职约 14-17 周，业余约 9-12 个月
- 完整对标竞品（再加页面编辑器、Tauri、自定义域名、反爬加固）：v1 之后再加 4-6 个月全职
