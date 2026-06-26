# 切片 pe-1 — Page Editor 后端基础设施（Spec）

**切片编号**：pe-1（Page Editor v1）
**类型**：新功能后端层（proto + migration + repo + handler + renderer）
**创建时间**：2026-06-26
**状态**：completed（2026-06-26 经 commit `19eca36` 完成）
**关联**：[PLAN.md §8 post-v1 #1](../../PLAN.md)、[page-editor dev 日志](../reports/completed/2026-06-26-page-editor-dev.md)

## Context

在 wv-1（widget SDK 消费端补齐）之后，需要为拖拽式落地页编辑器搭建完整的后端基础设施。这是三个 pe 切片中的第一个，只涉及后端代码，不涉及 admin UI 和 SDK 集成。

## 范围

### 做

1. **`packages/proto/src/` 新增 `page-schema.ts`** — PageSchema + Section 类型定义
2. **PG migration `000007_pages.up.sql`** — pages 表（单表 + JSONB）
3. **PG migration `000008_page_leads.up.sql`** — page_leads 表（表单提交）
4. **`server/internal/storage/page_repo.go`** — CRUD（Create/List/GetBySlug/Update/Delete/Publish）
5. **`server/internal/storage/page_lead_repo.go`** — 表单提交存储（Insert/ListByPageID）
6. **`server/internal/api/page_handler.go`** — REST API handler:
   - `GET /api/pages` — 列表
   - `POST /api/pages` — 创建（自动生成 slug）
   - `GET /api/pages/:slug` — 详情
   - `PUT /api/pages/:slug` — 更新 schema
   - `DELETE /api/pages/:slug` — 删除
   - `POST /api/pages/:slug/publish` — 发布/取消发布
   - `GET /api/pages/:slug/form` — 列出表单提交
   - `POST /api/pages/:slug/form` — 接收表单提交（公开，honeypot）
   - `GET /p/:slug` — 公开 SSR 渲染（Go template）
7. **`server/internal/pages/renderer.go` + templates/** — Go SSR 渲染引擎
8. **路由注册** — `server/internal/api/router.go` 注册新路由
9. **测试** — 集成测试覆盖 handler CRUD + SSR

### 不做

- Admin UI（pe-2 做）
- SDK 集成（pe-3 做）
- 表单提交记录在 admin 中的展示（pe-2 做）

## 决策表

| # | 决策点 | 选择 | 理由 |
|---|---|---|---|
| 1 | 存储模型 | 单表 `pages` + JSONB schema | 与 widget_configs 同模式，整页读写 |
| 2 | slug 生成 | 标题拼音化 + 随机后缀 | 手工输入易撞，自动生成友好 URL |
| 3 | SSR 渲染 | Go html/template + embed | 安全(自动转义)、可维护、与现有架构一致 |
| 4 | 表单防 spam | honeypot 隐藏字段 + rate limit | 轻量防垃圾，无 CAPTCHA 依赖 |
| 5 | 权限 | pages CRUD 需 auth；public SSR + form submit 公开 | 与 widget_config 一致 |

## Acceptance

- [ ] `page-schema.ts` 类型检查通过
- [ ] PG migrations 在集成测试中运行成功
- [ ] page_repo CRUD 集成测试通过（含重复 slug 拒绝）
- [ ] page_lead_repo 集成测试通过
- [ ] handler CRUD API 集成测试通过
- [ ] `GET /p/:slug` 返回正确 SSR HTML
- [ ] 表单提交 `POST /api/pages/:slug/form` 存储成功 + honeypot 过滤
- [ ] admin 测试绿 + Go 测试绿 + e2e 不回归

## 估时（solo）

- proto 类型定义：~0.5 天
- PG migrations：~0.5 天
- page_repo + page_lead_repo：~1 天
- page handler（CRUD + SSR）：~1.5 天
- renderer + templates：~1 天
- 测试：~1 天
- **累计**：~5.5 天 solo
