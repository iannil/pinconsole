# 官网 + README 设计（grill-me 14 轮访谈合成）

**状态**：in_progress（待用户确认）
**开始**：2026-06-22
**完成**：（未完成）
**关联**：
- 与 [`PLAN.md §1`](../../PLAN.md) "不考虑客户获取与销售"决策**冲突**——本设计是策略转向，需 PLAN.md 同步更新
- 与 [`MEMORY.md`](../../memory/MEMORY.md) "范围控制严格"约束**冲突**——同上
- 设计系统基线：[`docs/design-system.md`](../design-system.md) Calm Crafted

## Context

v1 主干完全收口（e2e 65/0/4，90+ commits，1a-1z 全切片交付 + 1aa-1ai-h 测试深化）。PLAN.md 原锁定的"不考虑客户获取与销售"约束在 v1 完成后解禁——现在到了接触潜在采用者的时机。

本设计通过 14 轮 grill-me 访谈达成共识，输出官网 + README 的完整设计蓝图。**产品本身仍 OSS 自托管、AGPL-3.0、不计费、不做注册流**；转向的是 maintainer 在官网层面接受 ToB 咨询询盘（PLG + 咨询式销售混合），类似 Posthog/Plausible 路径。

## 决策矩阵（14 项根级决策）

| # | 决策点 | 选择 | 理由 |
|---|---|---|---|
| 1 | 策略定位 | 接受咨询（PLG + 咨询式） | v1 完成、AGPL 立场不变、咨询的是部署/定制/托管服务而非订阅 |
| 2 | 目标客户 | 国内 ToB/SaaS 决策者（CEO/CTO/运营/市场） | 正在用美洽/智齿/Udesk/风车/小能/Live800 等，痛点：年费/数据/锁定/定制 |
| 3 | Hero 主钩子 | 数据主权 | 与 AGPL/OSS 立场一致，与 Posthog/Plausible 成功路径同构，涵盖合规敏感行业 |
| 4 | 主 CTA | 留资表单 + 文档次级 | 单人产能匹配的中摩擦入口，过滤 tire-kicker，技术决策者走次级文档 CTA |
| 5 | 竞品提及 | 完全不提，只讲类别 | 国内不正当竞争法 + 投诉文化风险高，格调优先于 SEO |
| 6 | Demo 形态 | Hero 视频 + 截图 | 30-60s 真实使用 MP4 + 每段配截图，可信度高、决策者读懂成本最低 |
| 7 | 语言策略 | 中英镜像（同内容互译） | 维护简单，符合 i18n from day 1；放弃 EN/ZH 异化定位以换产能 |
| 8 | 信息架构 | 单页长滚动 | 8 段漏斗，转化漏斗最短，移动端友好，决策者从头读到尾 |
| 9 | 视觉风格 | Calm Crafted 营销变体 | 复用 docs/design-system.md tokens，营销场景加大字号/留白/动效 |
| 10 | Landing 技术栈 | Astro + Vue 组件 | 静态输出、SEO 最优、Vue 组件可复用 admin tokens、i18n 原生 |
| 11 | 部署 | Cloudflare Pages + Workers + D1 | 数据在 maintainer 自己账号，与"数据主权"卖点一致 |
| 12 | 表单机制 | 自托管 Cloudflare Worker + D1 | 与数据主权一致，反垃圾用 rate limit + honeypot |
| 13 | README 范围 | 顶部加营销概述 + 保留 dev 主体 | GitHub 跳入的决策者能读懂，开发者不损失信息 |
| 14 | 章节结构 | 8 段漏斗型 | Hero → Problem → Features → 数据主权 → Self-host → Roadmap → FAQ → CTA |
| 15 | Tone | 冷静立场型 | 与 Calm Crafted 美学同构，参照 Linear/Vercel/Plausible |
| 16 | Tagline | "Your visitors, your data." / "你的访客，你的数据。" | 5 词立场，与数据主权 hook 完全对齐 |
| 17 | 域名 | 已有自有域名（用户填写） | Cloudflare Pages 绑定 |

## 仓库结构（决策）

```
pinconsole/
├── admin/                 # OSS 运营端 SPA（用户部署）
├── visitor-sdk/           # OSS 访客 SDK（用户部署）
├── landing/               # OSS 落地页模板（用户部署，仅 demo）
├── marketing/             # 【新增】maintainer 营销站（Astro + Vue，Cloudflare 部署）
│   ├── src/
│   │   ├── pages/         # /, /zh, /en, /api/leads (Worker)
│   │   ├── components/    # Hero / FeatureShowcase / Roadmap / FAQ / LeadForm
│   │   ├── styles/        # Calm Crafted 营销变体 tokens
│   │   └── content/       # 中英双语内容（Astro content collections）
│   ├── workers/           # Cloudflare Worker（form submission）
│   ├── migrations/        # D1 schema（leads 表）
│   ├── public/            # 截图 / demo.mp4 / og-image
│   ├── astro.config.mjs
│   ├── wrangler.toml      # Cloudflare 配置
│   └── package.json
├── server/                # OSS Go monolith（用户部署）
├── docs/                  # 文档
└── README.md              # 顶部加营销概述 + dev 主体
```

**关键区隔**：
- `landing/` = OSS 用户部署的落地页模板（极简、纯演示、不带营销代码）
- `marketing/` = maintainer 营销站（独立 Astro 项目，部署在自有域名，含表单 backend）

这避免营销代码污染 OSS repo，也避免用户部署时把 maintainer 的留资表单一起跑起来。

## 8 段漏斗型单页结构（详细）

### Section 1: Hero（Awareness 阶段）
- **大字 H1（64-96px）**：
  - ZH: "你的访客，你的数据。"
  - EN: "Your visitors, your data."
- **副标题 H2（24-32px）**：
  - ZH: "开源 ToB 实时访客监控 + 运营互动 + 录像回放平台。自托管，AGPL-3.0，数据从不出门。"
  - EN: "Open source ToB real-time visitor monitoring + operator interaction + session replay. Self-hosted, AGPL-3.0, data never leaves your infrastructure."
- **CTA 组**：
  - Primary: "预约咨询" / "Request consultation"（滚动到 Section 8 表单）
  - Secondary: "5 分钟自部署" / "Self-host in 5 min"（跳转 GitHub README quickstart）
  - Tertiary: "GitHub ★"（链接到 repo）
- **右侧/背景**：30-60s demo MP4（自动播放 muted loop，点击放大有声）
- **背景视觉**：Stone 暖中性 + 柔影 + 微妙的 Teal 渐变（**禁止紫渐变**）

### Section 2: 核心问题（Awareness → Interest 过渡）
- **小标题**："为什么 we built this" / "Why we built this"
- **3 个痛点卡片**（横向 grid）：
  1. **数据在第三方手里**：客户行为、运营对话、录像会话——全部在 SaaS 厂商服务器。合规审查时你只能祈祷。
  2. **功能被锁定**：想加一个字段、改一个流程、对接一个内部系统——都要等厂商排期，加钱，加时间。
  3. **年年涨价，迁移困难**：年费 30-100 万，每年续约涨 15%。想换？数据导不出来，团队重新培训。
- **视觉**：每张卡片一个 Phosphor icon（不用 emoji）+ 简洁排版

### Section 3: 核心能力（Interest 阶段）
- **小标题**："v1 已交付的能力" / "What v1 delivers"
- **5 个能力卡片**（2-3 列 grid）：
  1. **实时访客监控**：rrweb 全量采集 DOM/鼠标/点击/滚动/表单失焦值。1fps WebP 选择性截图（canvas/WebGL/iframe）。
  2. **双向协同（co-browsing）**：运营可高亮/点击/滚动/代填/跳转。防抖 300ms 平衡体验与流量。
  3. **录像回放**：MinIO 归档事件流 + 截图，30 天可配，rrweb-player 标准 replayer。
  4. **弹窗 + 聊天**：双向消息通道，持久化到 PostgreSQL。
  5. **反爬虫 + GDPR**：rate limit + UA + 行为 + fingerprint；consent opt-in + 被遗忘权 + IP 截断 + co-browse 横幅。
- 每卡片配一张高质截图（admin 真实界面）

### Section 4: 数据主权架构（Consideration 阶段 - 信任峰值）
- **小标题**："为什么你的数据真的在你手里" / "Why your data is actually yours"
- **架构图**（Mermaid 或 SVG）：访客 → pinconsole-server（你的服务器）→ PostgreSQL/Redis/MinIO（你的存储）。**无外部依赖**。
- **3 个子区块**：
  1. **AGPL-3.0**：copyleft 强保护。任何修改必须开源。**云厂商不能拿去做 SaaS**。
  2. **标准栈**：PostgreSQL 16 / Redis 7 / MinIO / Go 1.22 / Vue 3。**无锁定**，Schema 在你手里。
  3. **合规就绪**：GDPR consent + erasure + IP 截断；HttpOnly cookie + bcrypt；命令授权 + popup URL 白名单；WS trace_id 端到端。
- **视觉**：架构图 + 3 个 Phosphor 图标卡片

### Section 5: 自托管快速开始（Consideration → Decision 过渡）
- **小标题**："5 分钟跑起来" / "Running in 5 minutes"
- **代码块**（typography 突出，深色背景）：
  ```bash
  git clone https://github.com/iannil/pinconsole
  cd pinconsole
  cp .env.example .env
  make docker-up build-frontend build
  ./server/bin/pinconsole-server
  ```
- **次级链接**：完整文档、生产部署、API 参考
- **CTA**："技术评估？看文档" / "Technical evaluation? Read docs"

### Section 6: Roadmap（Consideration - 透明度）
- **小标题**："v1 完成。下一步去哪。" / "v1 done. What's next."
- **3 列 grid**：
  1. **✅ Shipped (v1)**：实时监控 / 双向协同 / 录像 / 弹窗聊天 / 认证 / 反爬 / GDPR / 可观测 / i18n / Docker 部署
  2. **🚧 Coming (post-v1)**：自定义域名 / 页面编辑器 / Tauri 桌面端 / SSO/SAML / 反爬加固 / 分析仪表盘
  3. **❌ Out of scope (ever)**：多租户 SaaS / 计费 / 注册流 / 云托管——**立场不变**。
- **视觉**：3 列卡片，shipped 用 Forest 绿，coming 用 Amber，out-of-scope 用 Stone 灰 + 删除线

### Section 7: FAQ（Decision 阶段 - 异议处理）
- **小标题**："常见疑问" / "Common questions"
- **8-10 个 FAQ（折叠式 accordion）**：
  1. **AGPL-3.0 商用合规吗？**——AGPL 要求修改开源，但"内部使用"不触发。SaaS 用 AGPL 项目对外服务需开源自己的修改——这正是为什么云厂商不能拿走。
  2. **单人开发能撑多久？**——v1 90+ commits 已完成端到端切片，post-v1 路线公开在 PLAN.md。咨询收入支撑持续维护。
  3. **能不能定制开发？**——能，这是咨询的核心。表单里写明需求，48h 内回复。
  4. **能不能托管我们部署？**——v1 不做托管，但可推荐合作的部署伙伴。长期立场是不做托管（避免与 OSS 用户竞争）。
  5. **怎么从 X 迁移？**——v1 不提供迁移工具，但咨询可包含数据迁移（按场景评估）。
  6. **500 并发不够怎么办？**——v1 是单实例 hub。多实例需要 Redis Pub/Sub 总线，是 post-v1 切片。
  7. **能跑在 k8s / 内网 / 私有云 吗？**——能，docker-compose 是参考部署，k8s/裸机/反代都行。
  8. **能不能过等保/ISO27001？**——产品层合规就绪（GDPR/bcrypt/审计日志），等保/ISO 需结合部署环境评估，咨询可协助。
  9. **支持移动端访客吗？**——支持。运营端仅桌面（v1）。
  10. **AGPL 和我们内部 GPL 不兼容代码怎么办？**——咨询，case-by-case。

### Section 8: Final CTA（Decision 阶段）
- **大字标题**："聊聊你的场景" / "Let's talk about your use case"
- **留资表单**（4 必填 + 1 可选 + honeypot）：
  - 姓名（必填）
  - 公司（必填）
  - 联系方式（手机或邮箱，必填）
  - 用途 dropdown（必填）：评估替代 / 自托管咨询 / 定制开发 / 合规咨询 / 其他
  - 留言（可选 textarea）
  - honeypot field（隐藏，反垃圾）
- **隐私声明**："提交后数据存于 maintainer 自托管 Cloudflare D1（欧洲/亚洲 region），48h 内回复，不分享第三方。"
- **次级**："不需要销售？直接看文档 / GitHub ★"

## 视觉设计（Calm Crafted 营销变体）

### Tokens 扩展（基于 docs/design-system.md）

```css
/* marketing variant — 比 admin 的 density 更宽 */
--pinconsole-marketing-section-y: 96px;    /* admin 是 32-48px */
--pinconsole-marketing-container: 1200px;  /* 比 admin 更宽 */

/* Hero 字号阶梯 */
--pinconsole-hero-h1: clamp(48px, 8vw, 96px);
--pinconsole-hero-h2: clamp(20px, 2.5vw, 32px);
--pinconsole-section-h2: clamp(32px, 4vw, 48px);
--pinconsole-section-h3: clamp(20px, 2vw, 24px);
--pinconsole-body: 16px;
--pinconsole-body-lg: 18px;

/* 复用 admin tokens */
--pinconsole-color-stone-*, --pinconsole-color-teal-*, --pinconsole-color-amber-*
--pinconsole-radius-*, --pinconsole-shadow-*
/* IBM Plex Sans / Sans SC / Mono */

/* 禁忌（继承 design-system.md §1.2） */
/* ❌ 紫渐变 / Inter/Geist/Roboto / slate+indigo / emoji-as-icon / Element Plus 蓝默认 #409eff / 单层 box-shadow / 首屏 stagger */
```

### 动效（Gentle & Restrained 营销变体）

- Hero 大字 fade-in + slight rise（120ms，单次，无 stagger）
- Scroll-triggered reveal：section 进入视口时 fade + 8px rise（160ms ease-out）
- Hover：feature 卡片 subtle lift（translateY -2px + shadow 增强，160ms）
- CTA hover：背景色过渡 200ms
- **禁止**：stagger 揭示、视差滚动、自动轮播、loading spinner > 240ms

## 技术架构

### Cloudflare 部署拓扑

```
your-domain.com (Cloudflare Pages)
├── Static HTML/CSS/JS (Astro build output)
├── /api/leads → Cloudflare Worker → D1 (leads table)
└── (可选) /demo → 真实 pinconsole-server 部署（独立子域，后期）

Worker 反垃圾：
- Rate limit：10 req/min/IP（Cloudflare native）
- Honeypot：hidden field 非空 = spam
- reCAPTCHA（可选，后期）
- 邮箱/手机格式校验

D1 schema：
CREATE TABLE leads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  company TEXT NOT NULL,
  contact TEXT NOT NULL,  -- phone or email
  purpose TEXT NOT NULL,  -- enum
  message TEXT,
  locale TEXT,            -- zh / en
  ip TEXT,                -- 截断 /24
  ua TEXT,
  created_at INTEGER NOT NULL,
  handled_at INTEGER,
  status TEXT DEFAULT 'new'  -- new / contacted / qualified / closed
);
```

### Astro 项目结构

```
marketing/
├── astro.config.mjs          # i18n routing: /zh, /en, / default
├── wrangler.toml             # Cloudflare 配置
├── package.json
├── public/
│   ├── demo.mp4              # 30-60s hero video
│   ├── og-image-zh.png
│   ├── og-image-en.png
│   └── screenshots/          # 5-8 张 section 配图
├── src/
│   ├── layouts/
│   │   └── MarketingLayout.astro
│   ├── components/
│   │   ├── Hero.astro
│   │   ├── Problem.astro
│   │   ├── Features.astro
│   │   ├── DataSovereignty.astro
│   │   ├── SelfHost.astro
│   │   ├── Roadmap.astro
│   │   ├── FAQ.astro
│   │   ├── FinalCTA.astro
│   │   ├── LeadForm.vue       # Vue 组件（需 interactivity）
│   │   ├── Header.astro
│   │   └── Footer.astro
│   ├── content/
│   │   ├── zh/                # 中文文案
│   │   └── en/                # 英文文案
│   ├── pages/
│   │   ├── index.astro        # 默认 zh-CN（geo-based redirect 后期）
│   │   ├── en.astro
│   │   └── api/
│   │       └── leads.ts       # Worker endpoint
│   ├── styles/
│   │   ├── tokens.css         # Calm Crafted 营销变体
│   │   └── global.css
│   └── workers/
│       └── leads.ts           # Cloudflare Worker
├── migrations/
│   └── 0001-create-leads.sql
└── README.md                  # marketing/ 自身说明
```

## 实施计划（4 phase / 10-15 天 solo 业余）

### Phase 1: 骨架（2-3 天）

- [ ] `marketing/` 目录 + Astro + Vue init
- [ ] Calm Crafted 营销变体 tokens.css
- [ ] i18n 路由（/ + /en，默认中文）
- [ ] Header / Footer / 基础 layout
- [ ] 空 section 占位

### Phase 2: 8 段内容 + 视觉资产（5-7 天）

- [ ] 中英镜像文案撰写（Hero / Problem / Features / FAQ / Final CTA）
- [ ] 录制 30-60s demo MP4（admin 实时监控 → co-browse → replay 流程）
- [ ] 5-8 张高质截图（dashboard / replay / chat / antiscrape / GDPR consent）
- [ ] 8 段组件实现
- [ ] Mermaid 架构图
- [ ] Calm Crafted 营销变体动效接入

### Phase 3: 表单 + Worker（2-3 天）

- [ ] LeadForm.vue 组件
- [ ] Cloudflare Worker + D1 schema + migration
- [ ] wrangler.toml + 本地 dev / 远端 deploy 验证
- [ ] 反垃圾：rate limit + honeypot + 格式校验
- [ ] 提交成功/失败 UX

### Phase 4: README + 部署（1-2 天）

- [ ] README 顶部加营销概述（tagline + 1-2 张截图 + 官网链接）
- [ ] Cloudflare Pages 绑定自有域名 + HTTPS
- [ ] OG meta + sitemap + robots.txt
- [ ] Analytics（Cloudflare Web Analytics 或 Plausible，不用 GA——立场一致）
- [ ] PLAN.md §1 + MEMORY.md 同步更新（去掉"不做营销页"约束，新增"咨询式 OSS"段落）

## 验证深度（按 docs/standards/verification-depth.md）

| 切片 | 深度 | 说明 |
|---|---|---|
| Phase 1 | 🔴 implemented-unverified | Astro 骨架跑起来即可 |
| Phase 2 | 🟡 verified-shallow | 8 段渲染正确 + 视觉资产齐 + 中英镜像 |
| Phase 3 | 🟢 verified-deep | 表单 e2e（submit → D1 → admin/CLI 读）+ 反垃圾负向测试 |
| Phase 4 | 🟢 verified-deep | 部署 + 域名 + HTTPS + analytics 真实访问 |

## Open Questions（用户确认时一起定）

1. **域名**：用户说"已有自有域名"，需告知具体域名以配置 DNS + wrangler.toml
2. **Worker 部署 region**：默认 auto，但若主要客户在国内，可指定 APJ region
3. **Lead notification 渠道**：仅存 D1 / 同时 SMTP 转发邮箱 / 同时企业微信 webhook——v1 建议仅 D1 + 邮件，后期加 webhook
4. **是否提供 demo 子域**（demo.your-domain.com 跑真实 pinconsole-server）：风险高（公开 demo 易被滥用），建议 v1 不做
5. **Analytics 选型**：Cloudflare Web Analytics（免费、隐私友好、自家产品）/ Plausible self-hosted / 不上 analytics——推荐 Cloudflare Web Analytics（与部署平台一致）
6. **PLAN.md/MEMORY.md 约束更新时机**：Phase 4 部署前必须同步，否则违反"事实来源优先级"原则

## Notes

- **MEMORY.md "范围控制严格"约束**：本设计**部分违反**——PLAN.md §1 "不考虑客户获取与销售——不做计费、注册流程、营销页"被改写为"产品本身不做计费/注册流，但 maintainer 自有官网做咨询转化"。需在 Phase 4 同步更新 PLAN.md + MEMORY.md。
- **CLAUDE.md "i18n from day 1"原则**：本设计遵循——中英镜像 from day 1。
- **CLAUDE.md "AGPL-3.0 一等公民"原则**：本设计强化——Hero 数据主权 hook 完全基于 AGPL 立场。
- **CLAUDE.md "前端设计基线"原则**：本设计复用 docs/design-system.md Calm Crafted，营销变体不引入新设计语言。
