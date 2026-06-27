# 2026-06-26 · `@marketing/` SEO + GEO 规划方案

**状态**：spec（grill-me 7 轮访谈后锁定）
**开始**：2026-06-26
**关联**：
- 与 [`docs/marketing/2026-06-25-domestic-finance-gtm-plan.md`](../marketing/2026-06-25-domestic-finance-gtm-plan.md)：国内金融线 GTM —— 本方案的 SEO/GEO 框架服务于该线的中文搜索/百度 AI 需求，但不偏该线优先
- 与 [`docs/marketing/2026-06-25-english-cold-start-gtm-plan.md`](../marketing/2026-06-25-english-cold-start-gtm-plan.md)：英语冷启动 GTM —— 本方案的对比页 `/alternatives/*` 直接输出该线所需的 SEO 落地页
- 与 [`docs/marketing/2026-06-23-marketing-ga4-design.md`](../marketing/2026-06-23-marketing-ga4-design.md)：GA4 设计文档 —— 阶段 3 实施，设计与该文档一致
- 与 [`docs/marketing/2026-06-22-landing-readme-design.md`](../marketing/2026-06-22-landing-readme-design.md)：官网设计 —— 本方案不改变现有视觉/内容，只做 SEO+GEO 增强
- 产品本身仍 OSS 自托管、AGPL-3.0、不计费、不做注册流。本方案只动 maintainer 的 marketing/ 对外推广层

---

## 根级决策（7 项，grill-me 已锁定）

| # | 决策点 | 选择 | 理由 |
|---|---|---|---|
| 1 | 服务对象 | **先搭框架，不偏任何一条 GTM 战线** | 框架先行，内容随 GTM 执行状况填充；不预判哪条线先跑 |
| 2 | GEO 范围 | **中英双覆盖**（ChatGPT/Perplexity/Gemini + 百度AI/文心一言/豆包） | 两线都需要 AI 搜索可见；结构化数据和内容策略可复用 |
| 3 | 内容深度 | **最小可行内容（A），后续升级** | 关键 landing page + 对比页先上；不建完整博客，留升级空间 |
| 4 | 结构化数据 | **一次性全上** | FAQ / Breadcrumb / SoftwareApplication / HowTo / VideoObject。SoftwareApplication 不依赖内容量，填一次就行 |
| 5 | CWV 优先级 | **B（图片）→ C（脚本）→ A（GA4随analytics）** | 有序分批，不并行摊薄精力 |
| 6 | 中文 SEO | **中英都做完整 SEO**（Google + 百度基础） | 百度站点提交 + `baidu-site-verification` + 百度可读的结构化数据；不做百度竞价/SEM |
| 7 | 关键词策略 | **C 先覆盖双品类，D 行业场景词作为 long-tail 补充** | Co-browsing + session replay 品类词为主落地页；金融/合规场景词在内容页做 |

---

## 实施阶段

### 阶段 0：基础设施（先做，不依赖内容量）

**目标**：让当前唯一页面（+中英两版）的 SEO 基础做到最好，并为后续内容页铺好框架。

#### 0a · 结构化数据扩展（`MarketingLayout.astro`）

当前 JSON-LD 只有 Organization + WebSite。扩展为 `@graph` 数组包含以下类型：

**SoftwareApplication**（新增）
```json
{
  "@type": "SoftwareApplication",
  "name": "PinConsole",
  "applicationCategory": "BusinessApplication",
  "operatingSystem": "Linux, macOS, Windows (Docker)",
  "description": "Open source ToB real-time visitor monitoring + operator interaction + session replay platform. AGPL-3.0, self-hosted.",
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "USD",
    "priceValidUntil": "2030-12-31"
  },
  "license": "https://github.com/iannil/pinconsole/blob/master/LICENSE"
}
```

**BreadcrumbList**（新增，每页一个）
```json
{
  "@type": "BreadcrumbList",
  "itemListElement": [
    { "@type": "ListItem", "position": 1, "name": "PinConsole", "item": "https://pinconsole.com" }
  ]
}
```

**WebPage**（新增，随内容页动态）
```json
{
  "@type": "WebPage",
  "name": "PinConsole — Your visitors, your data.",
  "description": "...",
  "url": "https://pinconsole.com",
  "inLanguage": "zh-CN"
}
```

**FAQPage**（随 FAQ.astro 组件动态注入）
```json
{
  "@type": "FAQPage",
  "mainEntity": [
    {
      "@type": "Question",
      "name": "AGPL-3.0 我们公司商用合规吗？",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "AGPL 要求...（精简到 100 字以内）"
      }
    }
  ]
}
```

**HowTo**（可选，在 SelfHost/Deploy section 注入部署流程）

#### 0b · 中文 SEO 基础

- 在 layout 的 `<head>` 新增 `baidu-site-verification` meta tag（占位，token 为空时不渲染）
- sitemap：当前 Astro sitemap 输出 `sitemap-index.xml` 已覆盖双语言。额外在 robots.txt 中加百度 sitemap 引用，或产出 `baidusitemap.xml` 镜像。建议：直接提交现有 sitemap 到百度站长平台，不需额外文件。
- 百度站长平台提交方案说明写入 `marketing/README.md`

#### 0c · 图片优化（CWV B 项）

- OG image（当前 62KB）→ WebP 压缩 ≤ 30KB，保持 1200×630 尺寸
- 所有 screenshot `<img>` 加 `loading="lazy"` + `width`/`height` 属性消除 CLS
- 检查 HTML 输出的 LCP 元素（当前可能是 Hero 文字或 background，不需要图片）
- 验证：Lighthouse 图片优化得分 ≥ 90

#### 0d · Turnstile script lazy load（CWV C 项）

当前：`MarketingLayout.astro` 第 77 行无条件加载 Turnstile api.js：
```astro
<script defer src="https://challenges.cloudflare.com/turnstile/v0/api.js" async />
```

改为：只在 `LeadForm.vue` 按需加载（Vue mounted 或 onFocus 时动态插入 script 标签），不在全局 layout 加载。参考：

```ts
// LeadForm.vue — onMounted 中懒加载 Turnstile
if (!document.querySelector('script[src*="turnstile"]')) {
  const script = document.createElement('script');
  script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js';
  script.async = true;
  script.defer = true;
  document.head.appendChild(script);
}
```

#### 0e · GA4 + CF Web Analytics 接入准备（阶段 3 先备）

按 GA4 设计文档（`docs/marketing/2026-06-23-marketing-ga4-design.md`）：
- 在 layout 的 frontmatter 改为从 env 读取 token（与 GA4 设计 §3.2 一致）
- 修掉 `PASTE_CF_WEB_ANALYTICS_TOKEN` 硬编码 bug（当前第 17 行 + 第 90 行，`cfAnalyticsToken !== 'PASTE_CF_WEB_ANALYTICS_TOKEN'` 这个判断就是 bug 根源——`PASTE_CF_WEB_ANALYTICS_TOKEN` 作为占位符在 prod 也不等于自己）
- CF token 配置检查改为 truthy 判断：`{cfToken && ( ... )}`
- GA4 gtag snippet 按条件渲染（gaId 为空时不渲染），Consent Mode snippet 始终渲染
- 本次先修 CF bug + 加 env 结构，不配置 measurement ID（等 GTM 执行时配置）

---

### 阶段 1：内容落地页（最小可行集）

**目标**：英语 GTM 所需的对比页 + 品类落地页先上线，中文版镜像同步。

#### 新增页面路由

| URL (EN) | URL (ZH) | 类型 | 关键词目标 |
|---|---|---|---|
| `/en/alternatives/upscope` | `/alternatives/upscope` | 对比页 | "open source upscope alternative", "upscope 开源替代" |
| `/en/alternatives/cobrowse-io` | `/alternatives/cobrowse-io` | 对比页 | "cobrowse.io alternative", "cobrowse.io 开源替代" |
| `/en/co-browsing/self-hosted` | `/co-browsing/self-hosted` | 品类页 | "self-hosted co-browsing", "私有化共浏览" |
| `/en/session-replay/self-hosted` | `/session-replay/self-hosted` | 品类页 | "self-hosted session replay", "私有化录像回放" |

#### 内容结构

**对比页结构**（`/alternatives/*`）：
- H1: "PinConsole vs [竞品名]: Open Source Alternative"
- Feature comparison table（5 行对比：data sovereignty / self-hosted / pricing / features / license）
- Migration motivation（why switch）
- CTA: Try live demo / Self-host in 5 min / Talk to maintainer
- FAQ section with schema

**品类页结构**（`/co-browsing/self-hosted` 等）：
- H1: "What is self-hosted co-browsing" / "什么是私有化共浏览"
- Problem definition + solution overview
- Feature list（与 Landing page Features section 一致，但更聚焦品类）
- Why self-hosted matters for compliance（GDPR / HIPAA / 数据不出域）
- CTA + FAQ

#### 新增 Content Collections 架构

当前 `src/content/` 只有两个文件（`zh.ts` / `en.ts`）导出 `PageContent` 对象。为支持多页面，扩展为：

```
src/content/
├── types.ts              # PageContent 保持不变（Landing 页面内容类型）
├── zh.ts                 # Landing 中文内容（不变）
├── en.ts                 # Landing 英文内容（不变）
└── pages/                # 新增：多页面内容
    ├── alternatives/
    │   ├── upscope-zh.ts
    │   ├── upscope-en.ts
    │   ├── cobrowse-io-zh.ts
    │   └── cobrowse-io-en.ts
    └── self-hosted/
        ├── co-browsing-zh.ts
        ├── co-browsing-en.ts
        ├── session-replay-zh.ts
        └── session-replay-en.ts
```

每个 ts 文件导出简单的 `{ meta, hero, content, faq, cta }` 结构（简化版 PageContent，不需要 Landing 的所有 section）。

或者直接用 Astro Content Collections（MD/MDX）：
```
src/content/pages/
├── en/
│   ├── alternatives-upscope.md
│   ├── alternatives-cobrowse-io.md
│   ├── self-hosted-co-browsing.md
│   └── self-hosted-session-replay.md
└── zh/
    ├── alternatives-upscope.md
    ├── alternatives-cobrowse-io.md
    ├── self-hosted-co-browsing.md
    └── self-hosted-session-replay.md
```

**推荐**：用 Astro Content Collections（MD 文件），与 Astro 生态一致，维护成本最低，SEO meta 可在 frontmatter 定义。

---

### 阶段 2：GEO 优化（AI 搜索可见）

**目标**：让 PinConsole 在 AI 搜索回答 "what is the best open source co-browsing" / "开源共浏览推荐" 时被引用。

#### 2a · 内容摘录优化

AI 搜索（ChatGPT/Perplexity）倾向于引用页面首段的事实陈述。关键落地页的 `<p>` 首段应写清晰、自包含的陈述：

> **Hero subtitle**（EN）: "PinConsole is an open-source, self-hosted alternative to Upscope and Cobrowse.io for real-time co-browsing, session replay, and visitor monitoring. AGPL-3.0 licensed, data never leaves your infrastructure."
>
> **Hero subtitle**（ZH）: "PinConsole 是开源、自托管的实时共浏览（co-browsing）和录像回放平台，可私有化部署，数据不出域，AGPL-3.0 许可。"

#### 2b · FAQ schema

FAQ 是 AI 搜索最常引用的结构化数据类型。在 FAQ.astro 组件中注入 `FAQPage` JSON-LD，确保：
- 每个 Q&A 对含 `@type: Question` + `acceptedAnswer: { @type: Answer, text: "..." }`
- text 长度控制在 100-300 字（AI 搜索倾向于中长度回答）

#### 2c · 百度 AI 搜索优化

百度 "AI 搜索" 同样依赖结构化数据和清晰的事实陈述。当前中文 FAQ 内容已对齐。额外补充：
- 中文品类页首段包含"开源"、"私有化"、"自托管"等百度爬虫高频索引词
- 百度站长平台提交 sitemap + 结构化数据测试

---

### 阶段 3：Analytics + 监测

**条件执行**：配置 GA4 的 measurement ID 需要维护者手动获取，本阶段只准备代码框架，实际激活等语义决定执行哪条 GTM 线后由维护者配 env var。

按 `docs/marketing/2026-06-23-marketing-ga4-design.md` 执行。

---

## 实施顺序（建议切片）

| # | 步骤 | 文件 | 阶段 |
|---|---|---|---|
| 1 | 结构化数据扩展（SoftwareApplication / Breadcrumb / FAQPage JSON-LD） | `MarketingLayout.astro`, `FAQ.astro` | 0a |
| 2 | baidu-site-verification + robots.txt 更新 | `MarketingLayout.astro`, `public/robots.txt` | 0b |
| 3 | 图片优化（OG image 压缩 + img loading/width/height） | `public/og-image.png`, 各 screenshot 引用处 | 0c |
| 4 | Turnstile api.js lazy load 移入 LeadForm.vue | `MarketingLayout.astro`, `LeadForm.vue` | 0d |
| 5 | CF Web Analytics token bug 修复 + env 重构 | `MarketingLayout.astro`, `wrangler.toml` | 0e |
| 6 | Astro Content Collections 配置（MD pages） | `astro.config.mjs`, `src/content/config.ts` | 1 |
| 7 | 对比页 × 4（中英各 2 个竞品） | `src/content/pages/{en,zh}/alternatives-*.md` + `.astro` | 1 |
| 8 | 品类页 × 4（中英各 2 个品类） | `src/content/pages/{en,zh}/self-hosted-*.md` + `.astro` | 1 |
| 9 | GEO 摘录措辞优化 + 全文检查 | `src/content/en.ts`, `zh.ts` + 新页面 MD | 2 |
| 10 | GA4 框架 + Consent Mode snippet（不配 measurement ID） | `MarketingLayout.astro`, `LeadForm.vue` | 3 |
| 11 | 构建验证 + Lighthouse 基线 | CLI | all |
| 12 | 方案文档 + 记忆更新 | `docs/progress/` | all |

---

## 不做项（显式排除）

- ❌ 完整博客 / 内容营销系统（等 GTM 确认内容线后再建）
- ❌ 百度竞价 / SEM / 付费搜索
- ❌ Google Ads
- ❌ TikTok / 小红书 SEO
- ❌ 多语言扩展（当前只做中英，不增加日/韩/德等）
- ❌ 修改 OSS `landing/` / `admin/` / `visitor-sdk/` / `server/` 任何代码
- ❌ Demo 沙盒（`demo.pinconsole.com`）—— 那是英语冷启动 GTM 的独立动作，不属于 SEO/GEO 框架
- ❌ 视频 demo（录制/托管/VideoObject schema）—— 虽然列入结构化数据计划，但视频资产本身依赖于内容线启动后才录制

---

## 验收标准

- [ ] `pnpm --filter @pinconsole/marketing check` 通过
- [ ] `pnpm --filter @pinconsole/marketing build` 通过
- [ ] HTML `<head>` 含 SoftwareApplication JSON-LD（双语言环境正确）
- [ ] HTML 含 BreadcrumbList JSON-LD
- [ ] FAQ section 含 `FAQPage` JSON-LD（内容与可见 accordion 一致）
- [ ] 所有 screenshot `<img>` 含 `loading="lazy"` + `width`/`height`
- [ ] OG image 压缩 ≤ 30KB
- [ ] Turnstile api.js 不在全局 layout 无条件加载（LeadForm.vue 按需加载）
- [ ] `PASTE_CF_WEB_ANALYTICS_TOKEN` 硬编码 bug 修掉（改为 env 条件判断）
- [ ] `baidu-site-verification` meta tag 占位（空 token 时不渲染）
- [ ] `/en/alternatives/upscope` 页面可渲染且含结构化数据
- [ ] `/en/alternatives/cobrowse-io` 页面可渲染
- [ ] `/en/co-browsing/self-hosted` 页面可渲染
- [ ] 中英对应路由正常工作（hreflang 正确）
- [ ] sitemap 包含新增页面
- [ ] Lighthouse Performance ≥ 90 / SEO ≥ 90
- [ ] 验证深度：🟡 verified-shallow（构建+结构化数据+图片+路由）
- [ ] 阶段 3 完成后升到 🟢 verified-deep（含 GA4 DebugView 验证+CF token 真实流量）

---

## 注意事项

- **英语冷启动 GTM 的对标页推荐直接点名竞品**（与中文站"不提竞品"策略相反）。本方案中的英语对比页点名 Upscope/Cobrowse.io，中文对比页同样点名但措辞更温和（"相比 Upscope..." 而非 "Upscope alternative"）。
- **GA4 设计文档关于 `CF_WEB_ANALYTICS_TOKEN` → `PUBLIC_CF_ANALYTICS_TOKEN` 的改名**：本方案暂时不改名，避免影响 env var 命名一致性。只在阶段 3 随 GA4 接入一起做。
- **图片优化分两轮**：阶段 0c 只做属性加载优化（lazy/width/height），实际压缩 OG image 需用工具（Sharp/ImageOptim/GUI）手动处理，不包含在代码改动中。
