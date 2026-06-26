# 切片 pe-2 — Admin 拖拽编辑器 UI（Spec）

**切片编号**：pe-2（Page Editor v2）
**类型**：新功能前端层（Vue 3 拖拽编辑器）
**创建时间**：2026-06-26
**状态**：in_progress — spec 阶段
**关联**：[pe-1 spec](./2026-06-26-slice-pe1-spec.md)、[PLAN.md](../../PLAN.md)

## Context

pe-1 完成了后端基础设施（proto 类型 + PG 表 + storage repos + API handler + Go SSR 渲染引擎）。pe-2 在 admin SPA 中实现拖拽编辑器 UI，让运营能在浏览器中通过拖拽组件搭建落地页。

## 范围

### 做

1. **`PagesView.vue`** — 落地页列表 `/pages`
   - 表格：slug / title / status / updated_at
   - 新建按钮（输入标题，自动生成 slug）
   - 编辑按钮 → 跳转编辑器
   - 删除确认
   - 发布/取消发布切换
2. **`PageEditorView.vue`** — 拖拽编辑器 `/pages/:slug/edit`
   - 左侧组件面板（9 种预设组件）
   - 中间画布（vuedraggable 拖拽排序 + 从面板拖入）
   - 右侧属性面板（选中组件后编辑 props）
3. **Pinia store `usePagesStore`** — 页面列表和编辑器状态
4. **API 客户端 `admin/src/api/pages.ts`** — 对接 pe-1 handler
5. **路由注册** — `router/index.ts` 新增 `/pages` 和 `/pages/:slug/edit`
6. **侧边栏导航** — 新增"落地页"入口
7. **i18n 文案** — 中英双语 keys

### 不做

- 表单提交记录列表页面（pe-3 做）
- 实时预览（后续迭代）
- 响应式/移动端适配编辑器（仅桌面运营）

## 决策表

| # | 决策点 | 选择 | 理由 |
|---|---|---|---|
| 1 | 拖拽库 | vue-draggable-plus | 社区成熟，排序/面板拖入支持好 |
| 2 | 编辑器布局 | 左侧面板 + 中间画布 + 右侧属性（3 列） | 主流模式，用户习惯 |
| 3 | 属性编辑 | 右侧面板即时编辑，选中同步 | 无需弹窗，操作流畅 |
| 4 | 组件预设 | 新建时插入默认 props 的空组件 | 用户拖入后立即编辑 |

## Acceptance

- [ ] `/pages` 列表页显示已有页面，支持新建/编辑/删除/发布
- [ ] 编辑器 `/pages/:slug/edit` 加载对应页面 schema
- [ ] 从左侧面板拖拽组件到画布
- [ ] 画布内拖拽调整排列顺序
- [ ] 点击组件 → 右侧面板显示属性编辑器
- [ ] 修改属性 → 画布即时更新显示
- [ ] 保存 → schema 存入 PG
- [ ] 发布/取消发布
- [ ] admin 测试绿 + e2e 不回归

## 估时（solo）

- PagesList + API client + store：~1 天
- 拖拽编辑器（画布 + 面板 + 属性）：~3 天
- 路由 + i18n + 导航：~0.5 天
- **累计**：~4.5 天 solo
