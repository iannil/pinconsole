# 切片 pe-3 — SDK 集成 + 表单提交（Spec）

**切片编号**：pe-3（Page Editor v3）
**类型**：前端 + SDK 集成
**创建时间**：2026-06-26
**状态**：in_progress — spec 阶段
**关联**：[pe-1 spec](./2026-06-26-slice-pe1-spec.md)、[PLAN.md](../../PLAN.md)

## Context

pe-1 已完成表单提交后端（POST /api/pages/:slug/form + honeypot）、pe-2 已完成拖拽编辑器 UI。pe-3 收尾：在 Admin 中查看表单提交记录，确保 SDK 自动加载到 SSR 页面。

## 范围

### 做
1. **SDK 自动加载** — SSR 渲染的页面已注入 `<script src="/sdk.js">`（pe-1 已实现），确认 e2e 生效
2. **PageLeadsView.vue** — 表单提交记录页面 `/pages/:slug/leads`
3. **路由注册** — `/pages/:slug/leads` 子路由
4. **列表页跳转** — PagesView 中每行的"leads"按钮
5. **i18n 文案** — leads 相关中英双语 keys

### 不做
- 表单提交通知（后续切片）
- 表单导出 CSV（后续切片）

## Acceptance
- [ ] `/pages/:slug/leads` 显示该页面的所有表单提交
- [ ] 提交时间为可读格式，字段以表格展示
- [ ] PagesView 有跳转到 leads 的入口
- [ ] Go 测试 + admin build 通过，不回归

## 估时
- PageLeadsView + 路由 + 导航：~1 天 solo
