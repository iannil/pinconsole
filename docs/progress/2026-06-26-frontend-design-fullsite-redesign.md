# 全站前端视觉一致性补齐

**时间**: 2026-06-26
**任务**: 使用 `/frontend-design` 对项目所有页面重新设计，风格贴近 marketing /grill-me

## 完成内容

### 1. Admin SPA Calm Crafted 样式补齐（5 页）

| 页面 | 修改内容 |
|---|---|
| `PagesView.vue` | `.btn/.btn-primary` → `.pc-btn--primary`；`.btn-text` → `.pc-btn--ghost`；`.input` → `.pc-input`；`.badge` → `.pc-badge--success/--accent`；dialog 加 `.pc-btn--icon`；scoped CSS 清理 |
| `DomainsView.vue` | `.card` → `.pc-card`；`.btn-primary/--danger` → `.pc-btn--primary/--danger`；input → `.pc-input`；scoped CSS 清理 |
| `WidgetsView.vue` | 全部 input → `.pc-input`；label → `.pc-label`；`.card` → `.pc-card`；`.save-btn` → `.pc-btn--primary`；checkbox 用 `.pc-checkbox`；scoped CSS 清理 |
| `PageEditorView.vue` | topbar `.btn-text` → `.pc-btn--ghost`；`.btn-primary` → `.pc-btn--primary`；`.status-badge` → `.pc-badge`；scoped CSS 清理 |
| `PageLeadsView.vue` | `.btn-text` → `.pc-btn--ghost`；`.lead-card` → `.pc-card`；scoped CSS 清理 |

**基建**：
- `base.css` 新增 `.pc-btn--icon`（icon-only 按钮原子类）
- `i18n/en-US.ts` / `zh-CN.ts` 新增 `pages.cancel` key

### 2. Landing 页重设计

替换 `landing/demo/index.html` + `server/cmd/server/embedded/landing/demo/index.html`
- **风格**: Marketing Linear dark（近黑 #08090A + Emerald #10B981 + IBM Plex Sans/Mono）
- **结构**: Header → Hero + CTA → Terminal/Install block → SDK Notice → Features grid → Demo Form → Footer
- **响应式**: Mobile 640px breakpoint，`prefers-reduced-motion` 支持
- **功能**: 保留 SDK script 标签，表单 alert demo

### 3. Visitor SDK

已验证：SDK 已正确使用 `data-pinconsole-*` 属性、`--pinconsole-*` CSS 变量前缀，四组件均 Calm Crafted 配色。未做修改。

## 未改动的文件

- `admin/src/styles/tokens.css`、`base.css`、`fonts.css`、`reset.css` — 已正确实现
- `admin/src/views/LoginView.vue`、`Dashboard.vue`、`ReplayList.vue`、`ReplayViewer.vue` — 已正确实现
- `admin/src/components/` — 已正确实现
- `visitor-sdk/` — 已验证

## 耗时

约 1.5 小时
