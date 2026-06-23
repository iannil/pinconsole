# 2026-06-23 · marketing/ 接入 GA4 · 设计文档

> **状态**：draft（待用户复核）
> **范围**：仅 `marketing/`（Astro + Cloudflare Pages）
> **不影响**：`landing/` / `admin/` / `visitor-sdk/` / `server/`
> **决策来源**：brainstorming 4 轮收敛（范围 → GA 产品 → 与 CF Analytics 关系 → GDPR consent）

---

## 1. 目标与范围

### 目标

为 maintainer 自有的 `marketing/` 官网接入 **Google Analytics 4**，用于：

- 营销漏斗顶部流量分析（pageview / 跳出率 / 来源）
- 转化追踪（`lead_submit` event，与 D1 `leads` 表互补——GA 做漏斗顶部，D1 做漏斗底部）
- 渠道效果对比（organic / direct / referral）

### 范围内

- `marketing/src/layouts/MarketingLayout.astro` 接入 gtag.js snippet
- `marketing/src/components/LeadForm.vue` 加 `lead_submit` conversion event
- `marketing/wrangler.toml` 新增/重命名公开 site vars
- `marketing/scripts/deploy.sh` check 加 GA/CF 配置检测
- `marketing/README.md` 加配置指引段落

### 范围外（显式排除）

- ❌ OSS `landing/` 模板不接入任何 GA（自托管用户不被插 maintainer 的 GA）
- ❌ `admin/` / `visitor-sdk/` 不接入 GA（产品本身是监控工具，无需 GA）
- ❌ Cookie banner / 显式 opt-in UI（用 Consent Mode v2 default-denied 替代）
- ❌ GTM 容器（单一 provider 不需要抽象层）
- ❌ Partytown / web worker（性能不敏感场景）
- ❌ 服务端 GA Measurement Protocol（数据流靠客户端 gtag 已足够）

---

## 2. 关键决策（brainstorming 收敛）

| 维度 | 决策 | 理由 |
|---|---|---|
| 接入位置 | 仅 `marketing/` | maintainer 自有官网，与 OSS 严格区隔；产品本身不需要 GA |
| GA 产品 | GA4（非 Universal Analytics / GTM） | UA 已下线；GTM 对单 provider 过度抽象 |
| 与 CF Web Analytics 关系 | GA4 主 + CF Web Analytics 后备 | 修现有 CF 占位 bug；CF 无 cookie 作交叉验证；国内 GA 被墙时 CF 仍可收基本流量 |
| 接入方式 | 直接 `gtag.js` snippet | 零依赖；与现有 layout 模式一致；单 provider 不需要抽象 |
| GDPR consent | Consent Mode v2 default-denied，不弹 banner | 合规 + 零摩擦；GA4 仍发 cookieless ping 做 modeling；后续可升级 banner |
| 转化事件 | 仅 `lead_submit`，参数 `purpose` + `locale` | 漏斗底部核心动作；不发 PII |
| Env 暴露路径 | `wrangler.toml [vars]` + Astro `import.meta.env.PUBLIC_*` | 公开值可 commit；Astro 构建期暴露给 client |

---

## 3. 详细设计

### 3.1 配置层（`marketing/wrangler.toml`）

新增 `PUBLIC_GA_MEASUREMENT_ID`，重命名 `CF_WEB_ANALYTICS_TOKEN` → `PUBLIC_CF_ANALYTICS_TOKEN`：

```toml
[vars]
SITE_URL = "https://pinconsole.com"
CONTACT_EMAIL = "contact@pinconsole.com"
TURNSTILE_SITE_KEY = "0x4AAAAAADpMi0AZU7xEXtzr"
PUBLIC_GA_MEASUREMENT_ID = ""           # 新增：GA4 Measurement ID（G-XXXXXXXX），空值时不渲染 snippet
PUBLIC_CF_ANALYTICS_TOKEN = ""          # 重命名自 CF_WEB_ANALYTICS_TOKEN（PUBLIC_ 前缀符合 Astro 约定）
```

**改名影响评估**：
- 当前 `CF_WEB_ANALYTICS_TOKEN` 在 wrangler.toml 里是空字符串
- dashboard 当前没有显式设置的值
- 改名零数据丢失，仅是命名规范化

### 3.2 Layout 层（`marketing/src/layouts/MarketingLayout.astro`）

#### 头部 frontmatter 读取 env

替换现有第 17 行硬编码：

```ts
// 旧
const cfAnalyticsToken = 'PASTE_CF_WEB_ANALYTICS_TOKEN'; // overridden by env in production

// 新
const gaId = import.meta.env.PUBLIC_GA_MEASUREMENT_ID;
const cfToken = import.meta.env.PUBLIC_CF_ANALYTICS_TOKEN;
```

#### `<head>` 末尾注入（顺序敏感：Consent Mode 必须在 gtag 之前）

```astro
<!-- GA4 Consent Mode v2 — default denied（不写 cookie；cookieless ping for modeling） -->
<script is:inline>
  window.dataLayer = window.dataLayer || [];
  function gtag() { dataLayer.push(arguments); }
  gtag('consent', 'default', {
    ad_storage: 'denied',
    analytics_storage: 'denied',
    ad_user_data: 'denied',
    ad_personalization: 'denied',
    wait_for_update: 500,
  });
</script>

<!-- GA4 gtag.js — 仅在 measurement ID 配置后渲染 -->
{gaId && (
  <Fragment>
    <script async src={`https://www.googletagmanager.com/gtag/js?id=${gaId}`} />
    <script is:inline define:vars={{ gaId }}>
      gtag('js', new Date());
      gtag('config', gaId, { anonymize_ip: true });
    </script>
  </Fragment>
)}
```

**关键点**：
- Consent Mode snippet **无条件渲染**（即使没配 GA ID，也要让 consent default 状态可见，便于将来加 banner）
- gtag.js snippet **条件渲染**（gaId 为空时不渲染，避免 404）
- `is:inline` 防止 Astro 处理/打包这段脚本（必须原样输出给浏览器）
- `define:vars` 让 Astro 把 `gaId` 注入到内联脚本作用域

#### `<body>` 底部修 CF Web Analytics

```astro
<!-- Cloudflare Web Analytics — privacy-friendly, no cookies -->
{cfToken && (
  <script defer src="https://static.cloudflareinsights.com/beacon.min.js"
    data-cf-beacon={`{"token": "${cfToken}"`} />
)}
```

**关键点**：去掉旧的 `cfAnalyticsToken !== 'PASTE_CF_WEB_ANALYTICS_TOKEN'` 判断（那是 bug 的根源），改成简单的 truthy 检查。

### 3.3 转化事件（`marketing/src/components/LeadForm.vue`）

在 `onSubmit` 的 `status.value = 'success';` 之后插入：

```ts
status.value = 'success';

// Fire GA4 conversion event（无 gtag 时 no-op，不发 PII）
if (typeof window !== 'undefined' && typeof (window as any).gtag === 'function') {
  (window as any).gtag('event', 'lead_submit', {
    purpose: form.purpose,
    locale: props.locale,
  });
}

form.name = '';
// ... rest of reset
```

**关键点**：
- `purpose` + `locale` 作为 event 参数（GA4 后台可做切片）
- **不发** `name` / `contact` / `company` / `message`（PII，违反 GA4 TOS）
- `window.gtag` 存在性检查（gtag.js 加载失败时 no-op）

### 3.4 部署集成

#### `marketing/scripts/deploy.sh` 的 `check` 子命令

在现有 check 流程末尾加：

```bash
# Analytics 配置检测（warning，不 fail）
GA_ID="${PUBLIC_GA_MEASUREMENT_ID:-}"
CF_TOKEN="${PUBLIC_CF_ANALYTICS_TOKEN:-}"
if [ -z "$GA_ID" ]; then
  echo -e "${YELLOW}⚠ PUBLIC_GA_MEASUREMENT_ID not set — GA4 disabled${NC}"
else
  echo -e "${GREEN}✓ GA4 measurement ID: $GA_ID${NC}"
fi
if [ -z "$CF_TOKEN" ]; then
  echo -e "${YELLOW}⚠ PUBLIC_CF_ANALYTICS_TOKEN not set — CF Web Analytics disabled${NC}"
else
  echo -e "${GREEN}✓ CF Web Analytics token: configured${NC}"
fi
```

（具体语法跟随现有 deploy.sh 的颜色变量约定）

#### `marketing/README.md`

新增段落 "## Analytics"，覆盖：
- 如何拿 GA4 Measurement ID（GA 后台 → 创建 property → Data Stream → Web → 拿 `G-XXXXXXXX`）
- 在哪填（`wrangler.toml [vars]` 的 `PUBLIC_GA_MEASUREMENT_ID`，或 dashboard env vars）
- 如何拿 CF Web Analytics token（CF dashboard → Web Analytics → 创建 site → 拿 token）
- 验证方法（GA4 DebugView + Tag Assistant 浏览器扩展）

---

## 4. 数据流

```
访客打开 pinconsole.com/
    ↓
prerendered HTML 含 gtag snippet（measurement ID 内联）
    ↓
浏览器加载 gtag.js → Consent Mode default-denied 生效
    ↓ （不写 cookie，发 cookieless ping for modeling）
GA4 后台收到 pageview
    ↓
访客提交 lead → POST /api/leads → 200 OK
    ↓
客户端 LeadForm.vue → gtag('event', 'lead_submit', {purpose, locale})
    ↓
GA4 后台漏斗：pageview → lead_submit

（并行走）CF Web Analytics beacon → 无 cookie 基本 traffic ping
```

---

## 5. 错误处理

| 场景 | 行为 |
|---|---|
| `PUBLIC_GA_MEASUREMENT_ID` 为空 | gtag.js snippet 不渲染（Consent Mode snippet 仍渲染，便于将来加 banner） |
| `PUBLIC_GA_MEASUREMENT_ID` 格式错（非 `G-` 开头） | dev mode `console.warn`；prod 静默跳过（不阻塞页面） |
| `gtag.js` 网络加载失败（含国内被墙） | 自然 no-op；CF Web Analytics 后备仍收基本流量 |
| `window.gtag` 未定义时 `gtag('event', ...)` 调用 | wrapper 检查 `typeof === 'function'` 后跳过 |
| `PUBLIC_CF_ANALYTICS_TOKEN` 为空 | CF beacon 不渲染 |
| GA4 在国内被墙 | 不阻塞页面渲染；转化数据漏报可接受（ToB 决策者多在境外或用代理） |

---

## 6. 测试策略

### 自动化

- **`pnpm --filter @pinconsole/marketing check`** 通过（astro check / TypeScript）
- **`pnpm --filter @pinconsole/marketing build`** 通过

### 手动验证

| 测试用例 | 步骤 | 期望结果 |
|---|---|---|
| GA ID 配置 | 设 `PUBLIC_GA_MEASUREMENT_ID=G-TESTTEST`，`pnpm build` | 构建产物 `dist/index.html` 含 gtag snippet |
| GA ID 未配置 | 留空，`pnpm build` | 构建产物不含 gtag snippet；Consent Mode snippet 仍在 |
| CF token 配置 | 设 `PUBLIC_CF_ANALYTICS_TOKEN=abc`，`pnpm build` | 构建产物含 CF beacon |
| Pageview | GA4 DebugView + Tag Assistant 打开页面 | DebugView 收到 pageview 事件 |
| Conversion | 提交 lead 表单成功 | DebugView 收到 `lead_submit` 事件，参数含 purpose/locale |
| Consent Mode | 打开页面，查看 Application → Cookies | 无 `_ga` cookie（default-denied 生效） |
| 国内被墙 fallback | 国内网络打开页面 | 页面正常渲染；CF beacon 仍发出（无墙问题） |

### 不写自动化测试的理由

marketing 现状没有 vitest；GA snippet 行为依赖浏览器环境（dataLayer / Consent Mode），自动化 ROI 低；手动 + DebugView 足够覆盖。

---

## 7. OSS 隔离确认

- 本次修改只动 `marketing/`
- `landing/` / `admin/` / `visitor-sdk/` / `server/` 不引入任何 GA 代码
- OSS 用户部署的实例默认不含 maintainer 的 GA（用户部署时不强制启用 `marketing/`，即使启用也是用户自己的 GA property）

---

## 8. 风险与缓解

| 风险 | 严重度 | 缓解 |
|---|---|---|
| GA4 在国内被墙，国内访客 gtag 加载失败 | 中 | 不阻塞页面；CF Web Analytics 作后备；ToB 决策者多在境外 |
| Consent Mode default-denied 导致 GA4 数据偏低（cookieless modeling 误差） | 中 | GDPR 合规必要代价；接受数据偏低换合规 |
| wrangler.toml `[vars]` 在 Astro 静态构建期能否被 `import.meta.env` 读到 | 低 | Astro Cloudflare adapter 在构建期注入 `[vars]` 到 `process.env`，`PUBLIC_` 前缀暴露给 client；实现时实测一次验证 |
| 改名 `CF_WEB_ANALYTICS_TOKEN` → `PUBLIC_CF_ANALYTICS_TOKEN` 影响 dashboard 已配值 | 低 | dashboard 当前为空，改名零影响；deploy.sh `secrets` 会提示重设 |
| GA4 TOS 违规（误发 PII） | 低 | conversion event 显式不发 name/contact/company；代码注释明确 |
| 后续维护者误以为可以加 cookie banner | 低 | README 文档说明 Consent Mode v2 是当前方案，加 banner 是升级路径不是替换 |
| 外部脚本（`gtag.js` / `beacon.min.js` / `turnstile api.js`）未加 SRI | 低 | **显式决策**：Google/Cloudflare 频繁轮换脚本内容，固定 SRI hash 会过时导致分析全断；业界标准做法是不加 SRI；残余风险（CDN 被攻陷）接受，因分析脚本权限有限（无敏感数据访问，仅发 ping） |

---

## 9. 实施顺序（建议切片）

| # | 步骤 | 文件 |
|---|---|---|
| 1 | wrangler.toml 配置 | `marketing/wrangler.toml` |
| 2 | Layout 接入 gtag + Consent Mode + 修 CF token bug | `marketing/src/layouts/MarketingLayout.astro` |
| 3 | LeadForm 加 conversion event | `marketing/src/components/LeadForm.vue` |
| 4 | deploy.sh check 加配置检测 | `marketing/scripts/deploy.sh` |
| 5 | README 加 Analytics 段落 | `marketing/README.md` |
| 6 | 本地 build 验证（GA ID 配/不配两种） | manual |
| 7 | GA4 DebugView 手动验证 | manual |
| 8 | 进展文档 + 完成报告 | `docs/reports/completed/` |

---

## 10. 验收标准

- [ ] `pnpm --filter @pinconsole/marketing check` 通过
- [ ] `pnpm --filter @pinconsole/marketing build` 通过
- [ ] GA ID 配置时构建产物含 gtag snippet
- [ ] GA ID 未配置时构建产物不含 gtag snippet
- [ ] Consent Mode default-denied 在 HTML 中可见
- [ ] LeadForm 提交成功后触发 `lead_submit` event
- [ ] CF Web Analytics token 读取 bug 修复（从 env 读，非硬编码）
- [ ] OSS 目录（`landing/` / `admin/` / `visitor-sdk/` / `server/`）无 GA 代码引入
- [ ] deploy.sh check 在 GA/CF 未配置时显示 warning（不 fail）
- [ ] README 含 Analytics 配置指引

---

## 11. 后续可选项（不在本切片）

- Cookie banner（用户可点 opt-in 后 `gtag('consent', 'update', { analytics_storage: 'granted' })`）
- 更多 conversion events（如 `cta_click` / `docs_view` / `github_click`）
- GA4 Measurement Protocol 服务端事件（如 D1 lead 入库后服务端也发一份）
- Partytown 集成（gtag 跑 web worker 优化页面加载性能）
- Plausible / Umami 作为 GA4 替代（如果未来想完全去 Google 化）
