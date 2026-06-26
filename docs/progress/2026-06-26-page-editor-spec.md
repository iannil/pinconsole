# 切片 page-editor — Spec（草案）

**切片编号**：page-editor（post-v1 #2）
**类型**：新功能（运营端页面/Widget 编辑器）
**创建时间**：2026-06-26
**状态**：✅ completed — pe-1(proto+PG+Go+API) + pe-2(admin UI) + pe-3(SDK 配置驱动) 全部实现并合并
**实现 commits**: `19eca36`(pe-1) `42a5625`(pe-2) `1dcda74`(pe-3)
**关联**：[PLAN.md §8 post-v1 候选](../../PLAN.md)、[design-system.md 前端基线](../design-system.md)

## Context

为什么做页面编辑器？

- **触发原因**：v1 已完成端到端访客监控 + 互动。运营目前只能通过硬编码修改弹窗/聊天 widget 的文案和样式（在 visitor-sdk/src/ui/ 中改代码），每次改动需重新构建 SDK + 部署。
- **业务/技术价值**：让运营在 admin 后台直接编辑落地页和互动组件的内容/样式，零代码改动。这是从"监控工具"到"运营平台"的关键跃迁。
- **不做的代价**：任何文案/样式变更都需要开发介入，运营效率低下；竞品标配可视化编辑。

## 范围边界

**本切片做（MVP 范围）**：
- Admin 后台 Widget 编辑器：管理弹窗(popup)、聊天(chat)、共浏览(co-browse banner)、同意书(consent banner) 4 组件的文案和基础样式（颜色、按钮文字）
- JSON 配置 schema + admin UI 表单 → 存 PG → SDK 读取渲染
- SDK 侧的配置驱动渲染（从硬编码切换为配置注入）

**本切片不做（避免范围爬升）**：
- 完整拖拽式落地页编辑器（post-v1 backlog，独立切片）
- 自定义 CSS/JS 注入（安全风险，需后续评估）
- 多语言内容管理（SDK i18n 已有，编辑器只编辑当前 locale）
- A/B 测试或多版本发布（后续切片）

## 决策表

| # | 决策点 | 选项 | 选定 | 理由 |
|---|---|---|---|---|
| 1 | 编辑器接入方式 | A: admin SPA 内 Vue 组件 / B: 独立页面 | **A** | 复用 admin 认证/布局/设计系统 |
| 2 | 配置存储 | A: PG JSON 列 / B: 独立 KV 表 | **A: 单表 + type 列** | 4 类 widget 共用 `widget_configs` 表，`tenant_id + widget_type` 联合唯一键；Go `[]byte` 处理 JSONB（与 `visitors.meta` 同模式） |
| 3 | SDK 获取配置时机 | A: init 时 HTTP GET / B: WS 推送 | **A: init 时 GET** | 简单可靠；`start()` 时 `GET /api/widget-config` 返回 4 类 JSON |
| 4 | Widget 类型 v1 范围 | A: 全部 4 类 / B: 仅 popup+chat | **全部 4 类** | MVP 一次覆盖 popup/chat/banner/consent |
| 5 | 配置 schema 定义 | A: proto 包新文件 / B: 独立包 | **A: proto 包** | `packages/proto/src/` 新增 `widget-config.ts` |
| 6 | 编辑权限 | A: 仅 admin / B: operator 也可 | **仅 admin** | 安全保守 |
| 7 | API 路由 | A: 公开 GET + 保护写 / B: 全公开 | **公开 GET + 保护写** | SDK 读无认证；admin 写需 AuthMiddleware |
| 8 | Admin UI 位置 | A: 独立路由 /widgets / B: 嵌入现有 | **独立页面** | 新增 `WidgetsView.vue` + router entry |

## 提议的切片拆分（顺序推进）

| 切片 | 内容 | 完成门 |
|---|---|---|
| **pe-1** | proto 包新增 `widget-config.ts` 类型 + PG migration `000006_widget_configs.up.sql` + Go repo 层 CRUD + API handler（公开 GET + protected PUT）| 类型检查 + 集成测试 |
| **pe-2** | Admin 侧：`WidgetsView.vue` + router `/widgets` + API 调用 + Pinia store | admin 测试绿 + 手测编辑保存 |
| **pe-3** | SDK 侧：`start()` 时 `GET /api/widget-config` → 各 UI widget 读取配置替换硬编码 | SDK 测试绿 + e2e 验证不同渲染 |

## Acceptance（总纲级）

- [ ] pe-1: PG `widget_configs` 表创建 + Go CRUD 通过集成测试（含 4 种 widget_type）
- [ ] pe-2: admin `/widgets` 页面可编辑 4 类 widget 的文案/颜色/按钮文字，保存后 PG 可查到
- [ ] pe-3: SDK 根据配置渲染不同文案，不修改 SDK 代码即可换文案；配置缺失时 fallback 到默认
- [ ] 全程：admin 测试绿 + visitor-sdk 测试绿 + Go 测试绿 + e2e 不回归

## 深度目标

- 🟢 verified-deep：pe-3 需负向测试（配置缺失 → SDK fallback 到默认文案不崩溃）；pe-1 需边界测试（超长文案截断、特殊字符转义）
- 🟡 verified-shallow：pe-2 编辑器 UI 用验收测试而非全量交互测试

## 估时（粗）

- pe-1：2-3 天（schema + PG + Go）
- pe-2：3-5 天（Vue 编辑器组件 + 表单 + 预览）
- pe-3：2-3 天（SDK 读取 + 渲染切换）
- **累计**：7-11 天 solo

## 关联

- PLAN.md §8 post-v1 #2 — 页面编辑器
- docs/design-system.md — 编辑器 UI 遵循 Calm Crafted 设计基线
- visitor-sdk/src/ui/ — 4 个 UI widget 的现有硬编码实现

## 未解决问题

1. **配置 vs 代码的边界**：SDK 现有硬编码内容包括文案、颜色 token、布局。v1 编辑器只覆盖"运营需频繁改动的部分"，其他保持硬编码。需确认哪些字段进配置。
2. **管理员权限**：编辑 widget 配置是否需 admin role（当前只有删除访客要求 admin），还是 operator 也可编辑？
3. **预览机制**：保存前是否提供预览弹窗？

---

> 本 spec 为草案，关键决策点标记"待确认"。请审阅并确认各决策方向后进入 pe-1 实施。
