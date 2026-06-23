# Marketing v2.1 — IA 重排 + 内容/排版优化 + 截图重跑

**状态**：completed
**完成时间**：2026-06-22
**深度 badge**：🟡 verified-shallow
**对应 v2 报告**：[2026-06-22-marketing-v2-linear-dark-redesign.md](./2026-06-22-marketing-v2-linear-dark-redesign.md)

## Summary

v2 落地后用户反馈「风格 OK，但内容和排版要重排」。通过 grilling 收敛 7 个决策：trust-first 叙事弧（Why 提前到 Features 前）、Features 3 大+2 小 bento 压缩 29%、Hero h2 收敛到纯痛点一句话、WhyPinconsole 改为 Pillars→Flow→Code 三段全宽、Roadmap 改为 2 col + 底部 manifesto callout、FAQ 改为 2 列 grid 全展开、FinalCTA 改为垂直居中三段。同时重新截取了 5 张 admin UI 高保真截图（dashboard / cobrowse-active / replay / chat / privacy），文件大小 2-3x 提升。

## Changes Delivered

### IA 重排（trust-first）

- ✅ `marketing/src/pages/index.astro` — 重排为 Hero → **WhyPinconsole** → Features → Roadmap → FAQ → FinalCTA
- ✅ `marketing/src/pages/en/index.astro` — 同步重排

### Hero tightening

- ✅ `marketing/src/content/zh.ts` + `en.ts` — Hero h2 收敛到纯痛点一句话。产品描述与 proof 全下提到 Features / WhyPinconsole
- ✅ `marketing/src/components/Header.astro` — credibility strip 信号从 `90+ commits / 65 e2e tests / AGPL-3.0 / self-hosted` 改为更面向 ToB 决策者的 `AGPL-3.0 / self-hosted / GDPR-ready / 48h 人回复`

### Section padding

- ✅ `marketing/src/styles/tokens.css` — `--pc-space-section-y` 120px → 96px（节奏更紧）

### WhyPinconsole rebuild（Pillars → Flow → Code）

- ✅ `marketing/src/components/WhyPinconsole.astro` — 重排为三段全宽：
  - Top：3 pillars 作为 trust anchors（1 行紧凑）
  - Middle：flow diagram 全宽（"看，数据不出你的基础设施"）
  - Bottom：code window 全宽（"验证：5 分钟跑起来"）

### Features bento（3 大 + 2 小）

- ✅ `marketing/src/components/Features.astro` — 重写为 bento：
  - Top 3（实时监控 / 双向协同 / 录像回放）：full-width copy + screenshot
  - Bottom 2（弹窗+聊天 / 反爬虫+GDPR）：2-col 紧凑卡，小截图 + 紧凑 copy
  - 总高 2750px → 1958px（-29%）

### Roadmap（2 col + manifesto callout）

- ✅ `marketing/src/components/Roadmap.astro` — shipped + coming 作为 2-col grid；out-of-scope 改为底部 manifesto callout（emerald 左边框 + x-circle icon + 项目 join with · + italic 立场不变）

### FAQ（2 列 grid 全展开）

- ✅ `marketing/src/components/FAQ.astro` — 从 accordion 改为 2 列 grid × 3 行 = 6 项全展开；hover 时 emerald dot + question text emerald

### FinalCTA（垂直居中三段）

- ✅ `marketing/src/components/FinalCTA.astro` — 重排为：
  - Band 1：全宽 manifesto title 居中（accent period）
  - Band 2：3 信任 badge 一行（clock-countdown / lock-key / shield-check）
  - Band 3：居中表单卡（max-width 560px）
- ✅ `marketing/src/components/PhosphorIcon.astro` — 加 `clock-countdown` + `lock-key` 图标

### Screenshots 重跑

- ✅ `marketing/public/screenshots/dashboard.png` — 78KB → 129KB（实时监控面板 + 1 visitor CONNECTED）
- ✅ `marketing/public/screenshots/cobrowse-active.png` — 82KB → 203KB（co-browsing 已启用 + chat tab + 控制模式提示）
- ✅ `marketing/public/screenshots/replay.png` — 65KB → 231KB（session 详情 + rrweb-player iframe + Play/seek/速度控制）
- ✅ `marketing/public/screenshots/chat.png` — 86KB → 218KB（chat 输入 + 已发送消息）
- ✅ `marketing/public/screenshots/privacy.png` — 57KB → 222KB（GDPR 级联删除表单 + fingerprint 输入示例）

### a11y 修复

- ✅ `marketing/src/components/Features.astro` — 移除 `.feature-screenshot` + `.compact-screenshot` 的 `aria-label`（避免 label-content-name-mismatch：button 内已有可见 chrome-title + chrome-action 文本作为 accessible name）

## Verification

### Dev server

```bash
./dev.sh start    # go on :8080, admin on :5173, sdk on :5174
pnpm --filter @pinconsole/marketing dev   # marketing on :4321
```

**预期**：
- IA 顺序：Hero(98) → WhyPinconsole(826) → Features(2482) → Roadmap(4440) → FAQ(5399) → FinalCTA(6601) → Footer(7850)
- Features 高度从 v2 的 2750px → v2.1 的 1958px（-29%）
- 全部 5 张新截图在 Features 区加载（1440x900，HMR 自动 pickup）
- Lighthouse：Accessibility 100 / Best Practices 100 / SEO 100 / Agentic Browsing 100（33 passed / 0 failed）

### 类型检查 + 生产 build

预存错误（与本次重排无关）依旧 2 个，在 `pages/api/leads.ts`。`pnpm --filter @pinconsole/marketing build` 仍可完成（prerender `/` + `/en/`）。

## Follow-ups

- 预存类型错误（leads.ts）→ 独立 PR
- `@fontsource-variable/fraunces` + `@fontsource/noto-serif-sc` 依赖清理 → 独立 PR
- screenshots 拍摄时 DB 中只有 1 active session + 15 historical；如果未来需要「多访客在线」的更丰富 dashboard 截图，可在 fixture 里加更多 demo visitors

## Notes

### 7 项 grilling 决策（用户确认）

1. 叙事弧：trust-first（Why 提前到 Features 前）
2. Features：3 大 + 2 小 bento
3. Hero h2：纯痛点一句话
4. WhyPinconsole：Pillars → Flow → Code 三段全宽
5. Roadmap：2 col + 底部 manifesto callout
6. FAQ：2 列 grid 全展开
7. FinalCTA：垂直居中三段

### 不变项

- 设计 DNA（Linear-dark + Emerald + IBM Plex/Noto Sans SC + Hybrid voice）保留
- v2 已退役的 Editorial Manifesto 元素（Fraunces / Cream / drop cap / Issue 01）不再回来
- admin 设计系统、AGPL license、self-host 单二进制、i18n 双语 全部不变
