# 切片 wv-1 — Widget SDK 消费端补齐（Spec）

**切片编号**：wv-1（Widget v2）
**类型**：现有功能补齐
**创建时间**：2026-06-26
**状态**：completed（2026-06-26 完成）
**关联**：[page-editor spec](../reports/completed/2026-06-26-page-editor-spec.md)、[PLAN.md](../../PLAN.md)

## Context

widget_configs 的 pe-1/2/3 已完成，4 类 widget 的编辑 UI 和存储层都已就绪。但 SDK 端只有 chat 和 consent_banner 消费了 widgetConfig 配置；popup 和 cobrowse_banner 虽然可编辑保存到 PG，SDK 端仍在用硬编码/WS 实时下发。

## 范围

### 做

1. **popup 接入 widgetConfig**：SDK Popup 组件在 `show()` 时读取 widgetConfig 的 popup 配置作为默认内容（title/body/action_label/action_url/dismissible/primary_color），WS 命令 `show_popup` 的字段可以覆盖默认值
2. **cobrowse_banner 接入 widgetConfig**：SDK CoBrowseBanner 从 widgetConfig 读取 operator_label/assist_hint/exit_label 替换当前 i18n 硬编码
3. **测试**：负向测试（配置缺失 → fallback 到默认不崩溃）

### 不做

- 后端/存储层改动（已有 pe-1/2/3 完整实现）
- admin 编辑器 UI 改动（已有 WidgetsView.vue）
- chat/consent_banner 现有实现的修改

## 决策表

| # | 决策点 | 选择 | 理由 |
|---|---|---|---|
| 1 | popup 配置覆盖策略 | WS 命令字段优先，widgetConfig 为默认值 | 兼容现有 WS 实时下发逻辑，无需改后端 |
| 2 | cobrowse_banner 替换策略 | widgetConfig 有值则用，无值 fallback 到当前 i18n hardcode | 向后兼容已部署的实例 |

## Acceptance

- [ ] Admin 编辑 popup 文案 → 访客侧 SDK popup 使用该默认文案
- [ ] WS 下发 `show_popup` 带自定义字段 → 覆盖 widgetConfig 默认值
- [ ] Admin 编辑 cobrowse_banner 文案 → 访客侧 SDK banner 使用新文案
- [ ] widgetConfig 缺失任何配置 → SDK fallback 到默认值，不崩溃
- [ ] 现有 chat / consent_banner 行为不回归

## 估时

- popup 接入：~1 天
- cobrowse_banner 接入：~0.5 天
- 测试：~0.5 天
- **累计**：~2 天 solo
