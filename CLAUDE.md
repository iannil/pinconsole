# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目定位

构建竞品的**开源替代品**。不考虑客户获取与销售——不做计费、注册流程、营销页。专注技术核心：访客实时监控 + 运营互动 + 录像回放 + （v1 之后）低代码页面编辑器。

License：**AGPL-3.0**（防止云厂商 SaaS 化）。

## 项目指南

- 目标：以强类型、可测试、分层解耦为核心，保证项目健壮性与可扩展性；以清晰可读、模式统一为核心，使大模型易于理解与改写。
- 语言约定：交流与文档使用中文；生成的代码使用英文；文档放在 `docs` 且使用 Markdown。
- 发布约定：
  - 发布固定在 `/release` 文件夹，如 rust 服务固定发布在 `/release/rust` 文件夹。
  - 发布的成果物必须且始终以生产环境为标准，要包含所有发布生产所应该包含的文件或数据（包含全量发布与增量发布，首次发布与非首次发布）。
- 环境约定：
  - 对于数据库、消息队列、缓存等，尽量使用docker部署环境
  - 如果是Python项目，尽量使用venv虚拟环境
  - 尽量为项目配置独立的网络，避免与其他项目网络冲突
- 文档约定：
  - 每次修改都必须延续上一次的进展，每次修改的进展都必须保存在对应的 `docs` 文件夹下的文档中。
  - 执行修改过程中，进展随时保存文档，带上实际修改的时间，便于追溯修改历史。
  - 未完成的修改，文档保存在 `/docs/progress` 文件夹下。
  - 已完成的修改，文档保存在 `/docs/reports/completed` 文件夹下。
  - 对修改进行验收，文档保存在 `/docs/reports` 文件夹下。
  - 对重复的、冗余的、不能体现实际情况的文档或文档内容，要保持更新和调整。
  - 文档模板和命名规范可以参考 `/docs/standards` 和 `docs/templates` 文件夹下的内容。

### 面向大模型的可改写性（LLM Friendly）

- 一致的分层与目录：相同功能在各应用/包中遵循相同结构与命名，使检索与大范围重构更可控。
- 明确边界与单一职责：函数/类保持单一职责；公共模块暴露极少稳定接口；避免隐式全局状态。
- 显式类型与契约优先：导出 API 均有显式类型；运行时与编译时契约一致（zod schema 即类型源）。
- 声明式配置：将重要行为转为数据驱动（配置对象 + `as const`/`satisfies`），减少分支与条件散落。
- 可搜索性：统一命名（如 `parseXxx`、`assertNever`、`safeJsonParse`、`createXxxService`），降低 LLM 与人类的检索成本。
- 小步提交与计划：通过 `IMPLEMENTATION_PLAN.md` 和小步提交让模型理解上下文、意图与边界。
- 变更安全策略：批量程序性改动前先将原文件备份至 `/backup` 相对路径；若错误数异常上升，立即回滚备份。

### 可观测性开发（Observability Driven Development）

- 为了能够完整追踪代码的执行流，请你遵循 "全链路可观测性 (Full-Lifecycle Observability)" 模式编写代码；
- 结构化日志： 所有的日志输出必须是 JSON 格式，包含字段：timestamp, trace_id (全链路唯一ID), span_id (当前步骤ID), event_type (Function_Start/End, Branch, Error), payload (变量状态)；
- 装饰器/切面模式： 请定义一个 LifecycleTracker 装饰器或上下文管理器；
- 在函数进入时：记录输入参数 (Args/Kwargs)；
- 在函数退出时：记录返回值 (Return Value) 和耗时 (Duration)；
- 在函数异常时：记录完整的堆栈信息 (Stack Trace)；
- 关键节点埋点： 在复杂的 if/else 分支、for/while 循环内部、以及外部 API 调用前后，必须手动添加埋点（Point）；
- 执行摘要： 代码运行结束时，必须能够生成一份“执行轨迹报告 (Execution Trace Report)”；
- 请确保埋点代码与业务逻辑解耦（尽量使用装饰器），不要让日志代码淹没业务逻辑；

### 记忆系统

本项目采用基于Markdown文件的透明双层记忆架构。禁止使用复杂的嵌入检索。 所有记忆操作必须对人类可读且对Git友好。

#### 存储结构

记忆分为两个独立的层："流"（日常）层和"沉积"（长期）层。

- 第一层：每日笔记（流）
  - 路径： `./memory/daily/{YYYY-MM-DD}.md`
  - 类型： 仅追加日志。
  - 目的： 记录上下文的"流动"。今天所说的一切、做出的决定以及完成的任务。
  - 格式： 按时间顺序排列的Markdown条目。

- 第二层：长期记忆（沉积）
  - 路径： `./memory/MEMORY.md`
  - 类型： 经过整理、结构化的知识。
  - 目的： 记录上下文的"沉积"。用户偏好、关键上下文、重要决策以及"经验教训"（避免过去的错误）。
  - 格式： 分类的Markdown（例如 `## 用户偏好`、`## 项目上下文`、`## 关键决策`）。

#### 操作规则

##### 上下文加载（读取）

当初始化会话或生成响应时，通过组合以下内容来构建系统提示：

1. 长期上下文： 读取 `MEMORY.md` 的全部内容。
2. 近期上下文： 读取当前（以及可选的之前）一天的每日笔记内容。

##### 记忆持久化（写入）

- 即时操作（日常）：
  - 每一次交互都需要确认当日的记忆存在，如果不存在，应先初始化当日记忆
  - 将每一次重要的交互、工具输出或决策追加到当天的每日笔记中。
  - 不要覆盖或删除每日笔记中的内容；将其视为不可变的日志。
- 整合操作（长期）：
  - 触发条件： 当检测到有意义的信息时（例如，用户陈述了偏好、发现了特定的错误修复模式、建立了项目规则）。
  - 操作： 更新 `MEMORY.md`。
  - 方法： 智能地将新信息合并到现有类别中。如果信息已过时，则移除或更新它。此文件代表*当前*的真实状态。

#### 维护与调试

- 透明度： 所有记忆文件都是标准的Markdown文件。如果代理因错误的上下文而行为异常，修复方法是手动编辑 `.md` 文件。
- 版本控制： 所有记忆文件都受Git跟踪。

## 事实来源优先级

1. `PLAN.md` — 架构、技术栈、切片拆分、决策理由（架构层冲突以此为准）
2. `START.md` — 产品需求、竞品能力、业务上下文（产品层冲突以此为准）
3. 本文件 — 给 Claude 的工作提示

## 已锁定的架构决策（详见 PLAN.md）

不要重新讨论这些——除非用户明确要求重审：

- **范围**：v1 是端到端最小切片（不含页面编辑器、Tauri、自定义域名）；完整对标竞品是终局
- **租户**：单租户部署，schema 预留 `tenant_id`；**不做多租户 SaaS**（用户明确"不考虑客户和销售"）
- **管道**：中心化 hub-and-spoke，所有流量经后端（不引入 WebRTC、P2P）
- **仓库**：Monorepo，Go embed 所有静态资源（admin SPA、SDK、落地页），单二进制部署
- **后端**：Go + Gin（**不用 Go-Zero**）+ coder/websocket（**不用 gorilla、Centrifugo、melody**）+ 自定义 hub
- **存储**：PostgreSQL（元数据）+ Redis（presence/限流/hot）+ MinIO（rrweb 事件流 + 选择性截图）
- **前端**：Vue 3 + TypeScript + Vite + Pinia + Element Plus + Vue I18n（中英双语 from day 1）
- **SDK**：TypeScript + Vite，构建产物 Go embed 至 `/sdk.js` 同源分发
- **co-browsing**：rrweb 双向；元素选择器用 rrweb 节点 ID（不用 CSS/XPath）；代填防抖动 300ms
- **截图**：选择性（仅 canvas/WebGL/跨域 iframe 触发，1fps WebP q70），不做持续全量
- **认证**：Email/password + bcrypt + HttpOnly cookie；WebSocket 同源依赖 cookie；MFA 可选
- **多运营**：1:1 锁定（claim/release）
- **可观测**：仅 slog 结构化 JSON 日志到 stdout（暂不加 metrics/tracing/Sentry）
- **域名**：v1 仅平台域名（`app.host/page/:id`）；自定义域名是后续切片
- **浏览器**：Modern evergreen desktop + mobile 访客；运营仅桌面

## 实施顺序（PLAN.md §7）

v1 切片拆为 1a-1j 共 10 个子切片，按顺序推进。**始终从下一个子切片开始，不要跳步**。每个子切片完成后才进入下一个。

## 给后续 Claude 实例的工作提示

- **任何架构层的工作开始前先读 `PLAN.md`**。如果当前任务与 PLAN.md 决策冲突，停下来跟用户确认，不要擅自改变方向。
- **不要扩大范围**。START.md 描述的是竞品的完整能力图谱，但 v1 是切片。如果用户要求实现 v1 之外的能力（页面编辑器、Tauri、自定义域名、多租户），先确认是否在调整 PLAN.md，再动手。
- **不要建多租户**。schema 预留 `tenant_id` 但 v1 不激活。"不考虑客户和销售"是用户明确指令。
- **安全和防爬虫是一等公民**（START.md 明确）。任何对外接口默认 rate limit + UA 黑名单 + 行为分析 + fingerprint 纵深防御。
- **WebSocket 并发目标（500）是单租户/单房间**，不是系统全局。不要为系统级广播过度设计 socket 层。
- **不要为不存在的服务写代码**。docker-compose、Makefile、CI 配置都不存在；不要在文档或代码里引用它们直到真的创建。
- **i18n from day 1**。所有用户可见文案走 Vue I18n key，不要硬编码中英文。
- **AGPL-3.0 一等公民**。任何引入第三方代码前检查 license 兼容性。
- 提交前按键监听在 GDPR/CCPA 下属敏感处理——涉及此功能时主动提示用户合规风险（详见 PLAN.md §10）。
