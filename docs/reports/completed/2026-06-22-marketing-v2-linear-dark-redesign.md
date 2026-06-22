# Marketing v2 — Linear 黑调极简重设计

**状态**：completed
**完成时间**：2026-06-22
**深度 badge**：🟡 verified-shallow
**对应 progress**：[docs/progress/2026-06-22-marketing-v2-linear-dark-redesign.md](../../progress/2026-06-22-marketing-v2-linear-dark-redesign.md)（如未独立创建，本报告即作为完成依据）

## Summary

将 marketing 站从 v1 Editorial Manifesto（Fraunces serif + Cream 纸纹 + 报刊框）整体重写为 v2 Linear 黑调极简（近黑 + Emerald + IBM Plex Sans + Noto Sans SC），修复"气质过于 indie/engineer、年费 30-100 万的 ToB 决策者觉得小作坊"的错位问题。Hero 改为 Type + Code/Install block；DataSovereignty 与 SelfHost 合并为 Why pinconsole；Problem 缩到 Hero 一句话；FAQ 裁到 6 项 + accordion；Header 加 credibility strip（commits/tests/AGPL/self-host）。

## Changes Delivered

### Tokens & global

- ✅ `marketing/src/styles/tokens.css` — 完全重写：Linear dark palette（`#08090A` canvas + `#0E1011` surface + `#141718` elevated）、Emerald accent（`#10B981`/`#34D399`）、Plex 字体变量、emerald radial spotlight 变量、退役 Cream/Fraunces/Fraunces variable 字体设定
- ✅ `marketing/src/styles/global.css` — 完全重写：IBM Plex Sans/Mono + Noto Sans SC @fontsource 导入、Linear 排版基线、退役 paper noise texture / drop cap / 4px rule

### Components

- ✅ `marketing/src/components/Header.astro` — 重写：删 "Issue 01 · 立场" 报刊框 + 4px 黑 rule；加 credibility strip（4 个信号：90+ commits · 65 e2e tests · AGPL-3.0 · self-hosted by you，thin strip + emerald dot + 1px sep）
- ✅ `marketing/src/components/Hero.astro` — 重写：Type + Code/Install block（Vercel 牌路）；emerald `$` prompt；mono 标题 `~/pinconsole`；emerald-subtle eyebrow chip；title 第二行 emerald
- ✅ `marketing/src/components/Features.astro` — 重写：5 项保留，tight grid（5fr copy / 6fr screenshot），screenshot 放进 code-window-style chrome（dot + mono title + emerald border on hover），1px divider
- ✅ `marketing/src/components/WhyPinconsole.astro` — **新建**：合并 DataSovereignty 3 pillar + SelfHost code window。左 flow window（3 nodes: visitor SDK → pinconsole-server → PostgreSQL·Redis·MinIO，server 节点 emerald 高亮，"无外部依赖" badge）；右 code window；下 3 pillars
- ✅ `marketing/src/components/Roadmap.astro` — 重写：3 col 保留，每列 status chip（shipped/coming/out-of-scope，pill 形，emerald/secondary/muted），item 用 mono marker（✓/·/×），out-of-scope 删除线
- ✅ `marketing/src/components/FAQ.astro` — 重写：accordion（默认展开第 1 项），mono numeral，emerald hover，open 时 question text 变 emerald，answer 用 fade-in 动画
- ✅ `marketing/src/components/FinalCTA.astro` — 重写：manifesto title accent + emerald 句号；form 放 surface card 里；input 改 elevated bg + emerald focus；CTA section 背后弱 emerald radial
- ✅ `marketing/src/components/Footer.astro` — 重写：精简为 brand+tagline / 3 col links / bottom meta + status dot（emerald + glow）；退役 inverse light bg
- ❌ `marketing/src/components/Problem.astro` — **删除**（内容融入 Hero h2）
- ❌ `marketing/src/components/DataSovereignty.astro` — **删除**（合并到 WhyPinconsole）
- ❌ `marketing/src/components/SelfHost.astro` — **删除**（合并到 WhyPinconsole）

### Content & i18n

- ✅ `marketing/src/content/types.ts` — 移除 `Problem` interface + `PageContent.problem` + `hero.videoSrc/videoPoster` 字段
- ✅ `marketing/src/content/zh.ts` — Hybrid voice：Hero h2 融入 SaaS 痛点（"年费 30-100 万的 SaaS 锁住你的数据..."）；FinalCTA title 改为 manifesto（"我们不卖订阅。我们讨论你的场景。"）；FAQ 从 10 项裁到 6（license/single-maintainer/customization/hosting/concurrency/compliance）；nav links 从 5 缩到 4（移除"自托管"，合并到"数据主权"）
- ✅ `marketing/src/content/en.ts` — 同步 Hybrid voice 英文版

### Pages

- ✅ `marketing/src/pages/index.astro` — 用 WhyPinconsole 替换 Problem + DataSovereignty + SelfHost
- ✅ `marketing/src/pages/en/index.astro` — 同步

### Dependencies

- ✅ `marketing/package.json` — 加 `@fontsource/ibm-plex-sans`、`@fontsource/ibm-plex-mono`、`@fontsource/noto-sans-sc`（保留 fraunces / noto-serif-sc 暂未移除，避免破坏暂未迁移的引用——可在后续清理）

## Verification

### Dev server + 视觉

```bash
pnpm --filter @pinconsole/marketing dev
# → http://localhost:4323/
```

**预期**：
- 首屏 hero：dark canvas + emerald spotlight + manifesto H1（第二行 emerald）+ 三 CTA + 右侧 code window（emerald $ prompt）
- Header 下 credibility strip 渲染 4 信号
- Features 5 项编号化，每项有 code-window-style 截图 chrome
- WhyPinconsole 显示 flow window + code window + 3 pillars
- Roadmap 3 col with status chips
- FAQ accordion 默认展开第 1 项
- FinalCTA：emerald radial 背景 + form card + manifesto title
- Footer：brand + 3 col + bottom status dot

### 类型检查

```bash
pnpm --filter @pinconsole/marketing check
```

**预期**：18 files checked。**已知预存错误 2 个**（在 `src/pages/api/leads.ts`，与本次重设计无关）：
- `Property 'runtime' does not exist on type 'Locals'` — Cloudflare adapter 类型问题
- `LeadPurpose | undefined not assignable to LeadPurpose` — strict mode narrowing

### 生产构建

```bash
pnpm --filter @pinconsole/marketing build
```

**预期**：build success，prerender `/` + `/en/`，dist 约 26MB（含 Noto Sans SC 多 subset）。

### Lighthouse（desktop snapshot）

```
Accessibility: 100
Best Practices: 100
SEO: 100
Agentic Browsing: 100
```

### Mobile（390x844 / Chrome dev tools 500x844 实测）

- docWidth < viewportWidth（无横向滚动）
- Header / Credibility strip / Hero / Features / WhyPinconsole / Roadmap / FAQ / FinalCTA / Footer 全部 stack 正常

## Follow-ups

- 预存类型错误（leads.ts 的 runtime/payload）→ 建议独立 PR 修复，scope 不在本次重设计内
- `@fontsource-variable/fraunces` + `@fontsource/noto-serif-sc` 可以从 package.json 移除（目前保留以避免破坏潜在引用）→ 建议后续清理 PR
- screenshot 资产（6 张 PNG，sub-100KB）若以后有高保真版本可替换 → 不在本次范围
- 最终用户验收（实际部署到 Cloudflare 后真实 ToB 决策者填表率）→ 上线后跟踪

## Notes

### 关键设计决策（grilling 收敛的 11 项）

1. 驱动：气质错位（v1 太 indie，需企业级）
2. 原型：Modern tool brand（Stripe/Linear/Vercel）
3. 流派：Linear 黑调极简
4. 强调色：Emerald `#10B981`/`#34D399`（与 admin Teal 小偏差）
5. 字体：IBM Plex Sans/Mono + Noto Sans SC（OFL、与 admin 连贯、非 Inter/Geist/Space Grotesk）
6. Hero：Type + Code/Install block
7. 视觉语言：Pure Linear（1px subtle border、极简动效、无装饰）
8. Voice：Hybrid（Hero/CTA manifesto + 内容事实型）
9. IA：合并 DataSovereignty + SelfHost；Problem 缩到 Hero；FAQ 裁到 6；加 credibility strip
10. Credibility strip：commits/tests/AGPL/self-host（4 信号）
11. FAQ 交互：Accordion

### 反 AI-slop 红线（贯穿全程）

❌ 紫渐变 / Inter/Roboto/Arial / slate+indigo / emoji-icon / Space Grotesk / Geist / Cream 纸纹 / serif / drop cap（全部退役）

### 不变项

admin 设计系统不动（Stone+Teal+Amber + Phosphor + IBM Plex）；AGPL-3.0 license；self-host 单二进制部署；i18n 中英双语 from day 1。
