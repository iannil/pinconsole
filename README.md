# marketing-monitor

> 构建一款 ToB 实时访客监控 + 运营互动 + 录像回放工具的**开源替代品**，对标某商业竞品。
> 不考虑客户获取与销售（不做计费、注册、营销页），专注技术核心。

## 当前状态

**Pre-code（规划完成，代码未启动）**

- ✅ 产品需求：[`START.md`](./START.md)
- ✅ 架构设计：[`PLAN.md`](./PLAN.md)
- ✅ 工作指南：[`CLAUDE.md`](./CLAUDE.md)
- ✅ 项目状态快照：[`docs/project-status.md`](./docs/project-status.md)
- ⏳ 代码：待启动切片 1a（仓库骨架）

License：**AGPL-3.0**（详见 [`LICENSE`](./LICENSE)）

## 快速开始

> ⚠️ Pre-code 阶段，无 build / test / run 命令。以下占位待切片 1a 启动后填充。

```bash
# 待切片 1a 完成：
# git clone https://github.com/<owner>/marketing-monitor
# cd marketing-monitor
# docker compose up -d           # 起 PG + Redis + MinIO
# make dev                      # 起后端 + admin SPA + SDK dev server
```

## 文档导航

### 事实来源（仓库根）

| 文档 | 角色 |
|---|---|
| [`CLAUDE.md`](./CLAUDE.md) | Claude 工作指南（含文档/记忆/可观测性约定） |
| [`PLAN.md`](./PLAN.md) | v1 架构与切片事实来源 |
| [`START.md`](./START.md) | 产品需求与竞品分析 |

### 工作文档（`docs/`）

| 路径 | 用途 |
|---|---|
| [`docs/project-status.md`](./docs/project-status.md) | **必读**——项目当前状态、决策清单、下一步动作 |
| [`docs/README.md`](./docs/README.md) | 文档索引 |
| [`docs/progress/`](./docs/progress/) | 进行中的修改单元 |
| [`docs/reports/completed/`](./docs/reports/completed/) | 已完成的修改报告 |
| [`docs/audits/`](./docs/audits/) | 审计发现 |
| [`docs/standards/`](./docs/standards/) | 文档/命名规范 |
| [`docs/templates/`](./docs/templates/) | 文档模板 |

### 记忆（`memory/`）

| 路径 | 用途 |
|---|---|
| [`memory/MEMORY.md`](./memory/MEMORY.md) | 长期记忆（沉积层） |
| [`memory/daily/`](./memory/daily/) | 每日笔记（流层） |

## v1 切片路线（详见 [`PLAN.md`](./PLAN.md) §7）

```
1a 仓库骨架 → 1b 单向最小 → 1c rrweb 接入 → 1d 录像归档
→ 1e 双向通道 → 1f 表单+跳转 → 1g 弹窗+聊天 → 1h 认证+多运营
→ 1i 反爬虫 → 1j i18n+部署+CI
```

v1 估时：solo 全职 14-17 周（3.5-4 个月）；业余 9-12 个月。

v1 之后：页面编辑器、Tauri、自定义域名、反爬加固、SSO、分析仪表盘。

## 给 LLM 的提示

第一次进入此项目时，**先读 [`docs/project-status.md`](./docs/project-status.md)**——它会在 60 秒内告诉你项目做什么、当前进展、下一步、哪些决策不能动。

## License

Copyright (C) 2026 marketing-monitor contributors.

本程序是自由软件：你可以依据 Free Software Foundation 发布的 GNU Affero General Public License（v3 或更新版本）条款重新分发或修改它。

我们分发此程序的目的是希望它有用，但**不提供任何担保**；甚至没有**适销性**或**特定用途适用性**的隐含担保。详见 [`LICENSE`](./LICENSE)。
