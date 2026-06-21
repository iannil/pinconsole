# pinconsole 设计系统

> 版本:v1.0 · 日期:2026-06-21
> 来源:`/frontend-design` + `/grill-me` 14 轮访谈共识
> 状态:📐 **设计基线**(实现未开始)

本文件是项目前端的单一设计权威。任何 admin SPA / visitor-sdk / landing 模板的视觉与交互决策都以本文件为准。如与 `PLAN.md` 冲突,以 `PLAN.md` 为准(架构与产品层)。

---

## 1. 设计立场

**Calm Crafted** —— 软中性色 + 人文 sans + 柔影 + 8px 圆角。

### 1.1 为什么不是 Mission Control / Brutalism

| 候选 | 否决理由 |
|---|---|
| Mission Control(工业控制台) | 诚实但冷感;运营也是人,工具不必 dystopian |
| Editorial Restraint(Linear 风) | 高级但烂用度高,面目模糊 |
| Honest Brutalism(全 mono + 尖角) | 反 SaaS 美学但对运营场景偏激进 |
| Calm Crafted ✓ | 信任、可长时间使用、双语友好、避 AI slop |

### 1.2 反 AI-slop 红线

明确禁止:
- ❌ 紫色 / indigo / 蓝紫渐变(尤其 login 页)
- ❌ `system-ui` / `Inter` / `Geist` / `Roboto` / `Space Grotesk` 作为主字体
- ❌ slate + indigo 默认 SaaS 配色
- ❌ emoji(🚩 等)在 UI 中作 icon 用
- ❌ 单层 `box-shadow`(用多层堆叠)
- ❌ 首屏 staggered 揭示动画(对运营工具是噪音)
- ❌ Element Plus 默认蓝 `#409eff` + 中性灰板(`#f5f7fa` / `#ebeef5` / `#606266`)

### 1.3 适合的场景

- 运营每天用 8h+ 的专业工具 → 柔和不刺眼
- 访客看到的 SDK 表面 → 不威胁感
- 中英双语环境 → 字体配对兼容
- OSS 自托管 → 诚实、可主题化、无外部依赖

---

## 2. 设计 Token

### 2.1 字体 (Typography)

**Family**:
```
--pc-font-sans:   "IBM Plex Sans", "IBM Plex Sans SC", -apple-system, sans-serif;
--pc-font-mono:   "IBM Plex Mono", ui-monospace, "Cascadia Code", monospace;
```

加载方式:**self-host woff2**(Go embed 兼容 + AGPL 合规 + 离线可用)。预加载 400/500/600。

**Type Scale**(对齐 4px grid):

| Token | Size | Line | Weight | 用途 |
|---|---|---|---|---|
| `--pc-text-xs` | 12px | 1.4 | 400 | caption / footnote |
| `--pc-text-sm` | 13px | 1.5 | 400 | 表格 / dense 数据 / label |
| `--pc-text-base` | 14px | 1.5 | 400 | body / button / input |
| `--pc-text-md` | 15px | 1.5 | 500 | subtitle |
| `--pc-text-lg` | 17px | 1.4 | 600 | h3 / section heading |
| `--pc-text-xl` | 20px | 1.3 | 600 | h2 / page heading |
| `--pc-text-2xl` | 24px | 1.2 | 600 | h1 |
| `--pc-text-3xl` | 32px | 1.2 | 600 | display(仅 login/marketing) |

### 2.2 色彩 (Color)

#### 中性 Stone(暖灰)

| Token | Light | Dark | 用途 |
|---|---|---|---|
| `--pc-color-bg-canvas` | `#FAF8F5` | `#1C1917` | 页面底 |
| `--pc-color-bg-surface` | `#FFFFFF` | `#292524` | 卡片 / 浮层 |
| `--pc-color-bg-subtle` | `#F5F1EC` | `#44403C` | hover / inactive tab |
| `--pc-color-border-default` | `#E7E5E4` | `#57534E` | 分隔线 / input 边框 |
| `--pc-color-border-strong` | `#D6D3D1` | `#78716C` | focus 边框外圈 |
| `--pc-color-text-primary` | `#1C1917` | `#FAF8F5` | 正文 |
| `--pc-color-text-secondary` | `#57534E` | `#D6D3D1` | 次要文本 |
| `--pc-color-text-muted` | `#78716C` | `#A8A29E` | placeholder / disabled |

#### 主色 Teal

| Token | Light | Dark | 用途 |
|---|---|---|---|
| `--pc-color-accent-default` | `#0F766E` | `#2DD4BF` | primary button / link |
| `--pc-color-accent-hover` | `#115E59` | `#5EEAD4` | hover |
| `--pc-color-accent-active` | `#134E4A` | `#14B8A6` | active / pressed |
| `--pc-color-accent-subtle` | `#CCFBF1` | `#134E4A` | selected row bg / badge bg |
| `--pc-color-accent-on` | `#FFFFFF` | `#1C1917` | text on accent |

#### 信号色 Signal

| Token | Light | Dark | 用途 |
|---|---|---|---|
| `--pc-color-success` | `#15803D` | `#4ADE80` | live / 成功 |
| `--pc-color-warning` | `#B45309` | `#FBBF24` | idle / 警告 |
| `--pc-color-danger` | `#B91C1C` | `#FCA5A5` | flagged / 错误 |
| `--pc-color-info` | `#0E7490` | `#67E8F9` | info badge |

**注**:danger 用 red-700 不是 pure red,降刺激;warning 用 amber-700 不是 yellow,避免警告黄。

### 2.3 间距 (Spacing)

**Calm Macro + Dense Micro** 双尺度:

```
/* Macro —— 页面 / 卡片 / 章节 */
--pc-space-page:      32px;   /* 页面 padding */
--pc-space-section:   24px;   /* 章节间距 */
--pc-space-card:      16px;   /* 卡片内 padding */

/* Micro —— 组件内 / 数据行 */
--pc-space-component: 12px;   /* 组件间 */
--pc-space-field:     8px;    /* label ↔ input */
--pc-space-row-y:     8px;    /* 数据行垂直 */
--pc-space-row-y-dense: 4px;  /* 紧凑表格 */
```

### 2.4 圆角 (Radius)

```
--pc-radius-sm:    6px;   /* input / small button */
--pc-radius-md:    8px;   /* button / tag */
--pc-radius-lg:    12px;  /* card */
--pc-radius-xl:    16px;  /* modal / drawer */
--pc-radius-pill:  9999px;/* badge / avatar */
```

### 2.5 阴影 (Shadow) —— 多层堆叠

```
--pc-shadow-xs: 0 1px 2px rgba(28,25,23,0.04);
--pc-shadow-sm: 0 2px 4px rgba(28,25,23,0.05), 0 1px 2px rgba(28,25,23,0.04);
--pc-shadow-md: 0 4px 8px rgba(28,25,23,0.06), 0 2px 4px rgba(28,25,23,0.04);
--pc-shadow-lg: 0 8px 16px rgba(28,25,23,0.08), 0 4px 8px rgba(28,25,23,0.05);
```

### 2.6 动效 (Motion) —— Gentle & Restrained

```
--pc-duration-fast: 120ms;   /* hover / focus / state-change */
--pc-duration-base: 180ms;   /* default */
--pc-duration-slow: 240ms;   /* modal / drawer open */
--pc-easing: cubic-bezier(0.4, 0, 0.2, 1);
```

**触发规则**:
- ✓ hover / focus / press / state-change / route fade / data-update pulse
- ✗ 首屏 stagger 揭示 / error shake / notification bounce / page transition slide

### 2.7 图标 (Iconography)

- 库:**Phosphor Icons**(`@phosphor-icons/vue`)
- Weight:`light`(subtle) / `regular`(default) / `bold`(emphasis) / `fill`(status active)
- 尺寸:16 / 20 / 24 / 32px(对齐 type scale)
- 替代当前所有 emoji(🚩 → `Flag` fill)和手撸 SVG

---

## 3. App Shell

**Layout**: Top bar + Content

```
┌─────────────────────────────────────────────────────────────────┐
│ ·pin  Dash  Replay  Privacy    │  ◉ LIVE 12  ⌘K  中/EN  ¶  │ 56px
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                         Content Area                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Top bar 规格**:
- 高度:56px
- 背景:`bg-surface`
- 下边框:`1px solid border-default`
- 内容:Logo · Nav(active 有 accent 下划线) · spacer · LIVE status · ⌘K search · 中/EN toggle · Profile menu
- 内 padding:`0 24px`

**Logo**:`pinconsole` wordmark,IBM Plex Sans 600,`text-xl`,前缀 `·` 小点装饰(产品 signature)。

---

## 4. 表面设计 (Admin Surfaces)

### 4.1 Login

**Layout**: Centered Card on Cream

```
                  (warm cream bg, subtle noise texture)

                  · · ·  pinconsole  · · ·
                  open-source visitor console

              ┌────────────────────────────────┐
              │                                │
              │  Email                         │
              │  [_________________________]   │
              │                                │
              │  Password                      │
              │  [_________________________]   │
              │                                │
              │  [       Sign in          ]    │  ← accent-default
              │                                │
              └────────────────────────────────┘
                      shadow-lg, radius-xl

              By signing in you acknowledge
              visitor recording is logged.
              Privacy · GDPR · Source          ← text-xs muted
```

- 卡片宽度:400px,padding:40px
- 背景:`bg-canvas` + SVG noise overlay(`opacity: 0.02`)
- 表单字段:label 上,input 下,`radius-md`,`border-default`,focus 时 `border-strong` + ring
- 主按钮:full-width,`accent-default` bg,`accent-on` text,`radius-md`,高 44px
- Dev 模式默认账号提示:卡片下方 text-xs muted(保留现有逻辑)

### 4.2 Dashboard

**Layout**: 3-Column Smart Expand

**默认(未 claim)**:
```
┌──────────┬─────────────────────────────────────┐
│ Visitors │ ◉ a3f9c2  · 4:23     [Claim →]      │
│ ──────── │ ─────────────────────────────────── │
│ LIVE 12  │                                     │
│          │                                     │
│ ·a3f9 ●  │      [   live rrweb replay   ]     │
│ ·b71c ◐  │                                     │
│ ·c0fe    │                                     │
│ ▲ flagged│                                     │
│          │                                     │
│ (280px)  │              (flex 1)               │
└──────────┴─────────────────────────────────────┘
```

**claim 后(3 栏展开)**:
```
┌──────────┬──────────────────────┬──────────────┐
│ Visitors │ ◉ a3f9 · 4:23        │ Engaging     │
│ ──────── │ ──────────────────── │ ──────────── │
│ LIVE 11  │                      │ Chat Popup Form │
│          │                      │ ──────────── │
│ ·a3f9 ●  │   [   live replay  ] │ visitor: hi! │
│ ·b71c ◐  │                      │ operator: ...│
│ ·c0fe    │                      │              │
│          │ ──────────────────── │ [type msg..] │
│          │ [Co-browse] [Release]│              │
│          │                      │ (320px)      │
└──────────┴──────────────────────┴──────────────┘
```

**关键约束**:
- List 宽:280px(可折叠到 60px icon rail,post-v1)
- Live 列:flex 1,最小 480px
- Engagement 列:320px(仅 claim 后展开)
- Engagement 默认 tab:Chat(Popup / Forms 次要)
- Live 列底部:`Co-browse` / `Release` 按钮(claim 后才显示)
- Visitor list 项:`fingerprint`(mono 13px) + `status dot`(8px) + `event count`(text-xs) + `time`(text-xs) + `flag icon`(Phosphor Flag fill,accent-danger)

### 4.3 Replay Viewer

**Layout**: Custom Calm Chrome + API 控制

```
┌──────────────────────────────────────────────────┐
│ ‹ Back   Visitor #a3f9c2 · 4:23 · 124 events     │ ← header
│          Started 2026-06-21 14:23                │
├──────────────────────────────────────────────────┤
│                                                  │
│                                                  │
│              [   rrweb player   ]                │ ← flex 1
│              (原生控制器隐藏)                    │
│                                                  │
│                                                  │
├──────────────────────────────────────────────────┤
│ ▶  ━━━━━━━●━━━━━━━━━━━━━━  2:14 / 4:23  1×  ⚙ │ ← custom controls
└──────────────────────────────────────────────────┘
```

- 隐藏 rrweb-player 原生 UI(`.rrweb-controller` `display: none`)
- 自建 Calm 控制栏:▶/⏸(Phosphor)、进度条、time、speed(1×/2×/4×/8×)、settings
- 用 rrweb-player API:`player.play()` / `.pause()` / `.goto(time)` / `.speed(rate)`
- header 含 Back link + 会话元数据(IBM Plex Mono 等宽对齐)

### 4.4 Privacy

(若未来添加 admin 端 GDPR 视图)
- Reading-friendly 单列布局,最大宽 720px
- Sections:访客数据清单 / 录制同意状态 / 删除请求按钮

---

## 5. Visitor SDK UI

**Stance**: Consistent Brand(与 admin 同一套 token)

### 5.1 实现约束

SDK 注入第三方页面,需要:
- 所有 CSS vars 加 `--pinconsole-*` 前缀(避免宿主页面冲突)
- 所有注入元素加 `data-pinconsole-*` 属性(便于识别 / 测试 / 排查)
- 所有样式 **不继承** 宿主(`all: initial` reset + 显式 token)
- 字体 self-host(或 CDN,但需 CSP 友好)

### 5.2 Consent Banner

**Layout**: Centered Card(不是全宽 strip)

```
                  (subtle scrim backdrop)

              ┌────────────────────────────────┐
              │  · pinconsole                  │
              │                                │
              │  我们记录访客操作用于客户服务  │
              │  与产品改进。你可随时撤回。   │
              │  Privacy                       │
              │                                │
              │  [  拒绝  ]   [ 同意并继续 ]   │
              └────────────────────────────────┘
```

- 触发:SDK 加载 + opt-in 模式
- 位置:居中,距顶 ~30%
- 卡片:400px 宽,`bg-surface`,`radius-xl`,`shadow-lg`
- Backdrop:`rgba(28,25,23,0.4)` scrim + blur(4px)
- 动画:fade-in 180ms
- 按钮:[Reject] 用 `text-secondary`(弱化),[Accept] 用 `accent-default`(主操作)

### 5.3 Co-browse Banner

**Layout**: 顶栏 full-width,cream + teal(不是警告黄 + 红)

```
┌──────────────────────────────────────────────────────────────────┐
│  ◉ 运营正在协助你 · 可见页面操作              [ 退出协助 ]       │ 40px
└──────────────────────────────────────────────────────────────────┘
```

- 高度:40px
- 背景:`bg-subtle`(cream)
- 下边框:`1px solid border-default`
- 内容:◉ live 圆点(success color)+ 文本(text-secondary)+ 右侧退出按钮(text-secondary,无背景,hover 时 `bg-subtle`)
- **NO** 🔔 emoji(用 Phosphor Eye / Headset icon)
- **NO** 红色退出按钮(避免威胁感,用 text link style)

### 5.4 Chat Widget

```
                                       ┌────┐
                                       │ ·· │  ← 56px circle
                                       └────┘    accent-default bg
                                                 shadow-md
                                                 bottom-right 20px
```

展开后:
```
                                       ┌────────────────┐
                                       │ ◉ Operator     │
                                       │ ─────────────  │
                                       │ visitor: 你好  │
                                       │ operator: 欢迎 │
                                       │                │
                                       │ ─────────────  │
                                       │ [输入消息] [➤] │
                                       └────────────────┘
                                       360×480
                                       bg-surface
                                       radius-lg
                                       shadow-lg
```

- Bubble:56px 圆,`accent-default` bg,Phosphor `chats-circle` icon 24px(`accent-on` color),`shadow-md`
- 未读 badge:pill,`accent-danger` bg,白色文本,右上角
- 展开 panel:360×480,`bg-surface`,`radius-lg`,`shadow-lg`
- panel 头:avatar(40px 圆)+ operator name(`text-md` 500)
- 消息列表:flex-end,operator 右(`accent-subtle` bg),visitor 左(`bg-surface` + border)
- 输入行:flex,input + 发送按钮(`accent-default` icon button)

### 5.5 Popup(运营推送)

**Layout**: Centered Modal

```
                  (subtle scrim backdrop)

              ┌────────────────────────────────┐
              │  Title                     ×   │
              │  ────────────────────────      │
              │  Body text...                  │
              │                                │
              │  [ Action button ]             │
              └────────────────────────────────┘
```

- Modal:400px 宽,`bg-surface`,`radius-xl`,`shadow-lg`
- Backdrop:`rgba(28,25,23,0.4)` scrim
- 关闭:右上 × 图标 / backdrop 点击 / ESC(若 `dismissible: true`)
- Action 按钮:`accent-default`,右下
- 动画:fade + scale 240ms

---

## 6. Token 架构

### 6.1 文件结构

```
admin/
├── public/
│   └── fonts/                      # IBM Plex woff2 self-host
│       ├── IBMPlexSans-Regular.woff2
│       ├── IBMPlexSans-Medium.woff2
│       ├── IBMPlexSans-SemiBold.woff2
│       ├── IBMPlexSansSC-Regular.woff2
│       ├── IBMPlexSansSC-Medium.woff2
│       ├── IBMPlexSansSC-SemiBold.woff2
│       └── IBMPlexMono-Regular.woff2
└── src/
    └── styles/
        ├── tokens.css              # :root vars
        ├── reset.css               # global reset
        ├── fonts.css               # @font-face
        └── base.css                # 元素基础样式

visitor-sdk/
├── src/
│   ├── fonts/                      # 同上 IBM Plex woff2
│   └── styles/
│       └── tokens.ts               # token constants(JS 对象)
```

### 6.2 命名规则

**CSS 变量**:`--pc-{category}-{role}-{state?}`

```css
/* 颜色 */
--pc-color-bg-canvas
--pc-color-bg-surface
--pc-color-accent-default
--pc-color-accent-hover
--pc-color-text-primary

/* 字体 */
--pc-font-sans
--pc-font-mono
--pc-text-base

/* 间距 */
--pc-space-page
--pc-space-section
--pc-space-card

/* 圆角 */
--pc-radius-md

/* 阴影 */
--pc-shadow-md

/* 动效 */
--pc-duration-base
--pc-easing
```

**SDK 变量**:`--pinconsole-{...}`(镜像 admin token,prefix 避免宿主冲突)

### 6.3 Light/Dark 双模

```css
:root {
  --pc-color-bg-canvas: #FAF8F5;
  --pc-color-text-primary: #1C1917;
  /* ... light values */
}

@media (prefers-color-scheme: dark) {
  :root {
    --pc-color-bg-canvas: #1C1917;
    --pc-color-text-primary: #FAF8F5;
    /* ... dark values */
  }
}
```

**无 toggle UI**——v1 仅跟随系统。Token 系统从第一天支持双模,后期加 toggle 时只改 `:root` class。

### 6.4 字体加载策略

```css
/* fonts.css */
@font-face {
  font-family: 'IBM Plex Sans';
  src: url('/fonts/IBMPlexSans-Regular.woff2') format('woff2');
  font-weight: 400;
  font-style: normal;
  font-display: swap;  /* 加载时用 fallback,加载完切换 */
  unicode-range: U+0000-00FF, U+0131, U+0152-0153, ...; /* 拉丁 */
}
/* + Medium / SemiBold / SC 变体 */
```

预加载 critical weights(Regular + Medium):
```html
<link rel="preload" href="/fonts/IBMPlexSans-Regular.woff2" as="font" type="font/woff2" crossorigin>
```

Fallback 字体链:
```
-apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif
```

---

## 7. 组件清单

需要重写的现有组件 / 新增组件:

### Admin SPA

| 类型 | 组件 | 状态 |
|---|---|---|
| **现有重写** | `LoginView.vue` | 紫渐变 → Centered Card on Cream |
| **现有重写** | `Dashboard.vue` | 顶栏 + 2 列 → App Shell + 3-Column Smart Expand |
| **现有重写** | `VisitorList.vue` | 加 Phosphor Flag icon、状态色点 |
| **现有重写** | `VisitorPanel.vue` | 拆分:Live column + Engagement panel |
| **现有重写** | `ReplayPlayer.vue` | 加 sandbox observer + 自定义控制层(实时模式) |
| **现有重写** | `ChatPanel.vue` | 移到 EngagementPanel |
| **现有重写** | `CoBrowseOverlay.vue` | Calm 样式重写 |
| **现有重写** | `FloatingInput.vue` | Calm 样式重写 |
| **现有重写** | `ReplayList.vue` | Calm 表格 + Phosphor icons |
| **现有重写** | `ReplayViewer.vue` | Custom Calm Chrome + rrweb API |
| **现有重写** | `Privacy.vue` | Reading-friendly layout |
| **新增** | `AppTopBar.vue` | App Shell 顶栏 |
| **新增** | `AppNav.vue` | Nav 项(Dashboard / Replay / Privacy) |
| **新增** | `LiveColumn.vue` | Dashboard 中间列(Header + Replay + Controls) |
| **新增** | `EngagementPanel.vue` | Dashboard 右列(Tabs: Chat / Popup / Forms) |
| **新增** | `ReplayControls.vue` | Replay viewer 自建控制栏 |
| **新增** | `LangToggle.vue` | 中/EN 切换 |
| **新增** | `ProfileMenu.vue` | 用户菜单 + logout |
| **新增** | `StatusBadge.vue` | LIVE / WS status 通用 badge |

### Visitor SDK

| 类型 | 模块 | 状态 |
|---|---|---|
| **现有重写** | `consentBanner.ts` | 全宽底栏 → Centered Card |
| **现有重写** | `coBrowseBanner.ts` | 警告黄 → cream + teal,去 emoji |
| **现有重写** | `chatWidget.ts` | system-ui + 默认蓝 → Calm Crafted + Teal |
| **现有重写** | `popup.ts` | Calm Modal |

---

## 8. 迁移计划

### Phase 1: Foundation(1-2 天)

- [ ] 添加 IBM Plex 字体 woff2 到 `admin/public/fonts/` + `visitor-sdk/src/fonts/`
- [ ] 添加 `@phosphor-icons/vue` 依赖
- [ ] 创建 `admin/src/styles/{tokens,reset,fonts,base}.css`
- [ ] 创建 `visitor-sdk/src/styles/tokens.ts`
- [ ] 在 `main.ts` 接入全局样式

### Phase 2: App Shell + Login(1-2 天)

- [ ] 重写 `LoginView.vue`(Centered Card on Cream)
- [ ] 新增 `AppTopBar.vue` + `AppNav.vue` + `LangToggle.vue` + `ProfileMenu.vue`
- [ ] `App.vue` 接入 App Shell(顶栏 + router-view + 内容区)

### Phase 3: Dashboard 重构(3-5 天)

- [ ] 拆分 `Dashboard.vue` 为 3-Column Smart Expand 容器
- [ ] 重写 `VisitorList.vue`(Calm + Phosphor Flag + 状态色)
- [ ] 新增 `LiveColumn.vue`(Header + Replay + Controls)
- [ ] 新增 `EngagementPanel.vue`(Tabs: Chat / Popup / Forms)
- [ ] 重写 `ChatPanel.vue` + `CoBrowseOverlay.vue` + `FloatingInput.vue`

### Phase 4: Replay Viewer(1-2 天)

- [ ] 新增 `ReplayControls.vue`(Calm 自建控制栏)
- [ ] 重写 `ReplayViewer.vue`(Custom Calm Chrome + API 控制)
- [ ] 重写 `ReplayList.vue`(Calm 表格 + Phosphor icons)

### Phase 5: Visitor SDK(2-3 天)

- [ ] 重写 `consentBanner.ts`(Centered Card)
- [ ] 重写 `coBrowseBanner.ts`(cream + teal + 去 emoji)
- [ ] 重写 `chatWidget.ts`(Calm Crafted bubble + panel)
- [ ] 重写 `popup.ts`(Calm Modal)

### Phase 6: 验证(1 天)

- [ ] 5 admin views + 6 admin components + 4 SDK surfaces 视觉确认
- [ ] Light/Dark 双模自动切换验证
- [ ] 中英文双语字体回退验证
- [ ] Lighthouse audit(a11y / performance)
- [ ] Playwright e2e:登录 → Dashboard → claim → chat → cobrowse → replay 全链路

**预计总工作量**:solo 全职 10-15 天 / 业余 4-6 周。

---

## 9. 决策追溯

14 个核心决策(2026-06-21 `/grill-me` 访谈):

| # | 决策 | 选择 | 备选(被否决) |
|---|---|---|---|
| 1 | 美学方向 | Calm Crafted | Mission Control / Editorial Restraint / Honest Brutalism |
| 2 | 字体家族 | IBM Plex 全家族 | Hanken+Noto / Manrope+MiSans / Fraunces+DM Sans |
| 3 | 色彩系统 | Stone + Teal + Amber | Zinc+Plum / Greige+Sage / Slate+Indigo |
| 4 | 密度哲学 | Calm Macro + Dense Micro | 全面 Spacious / 全面 Dense / Toggle |
| 5 | 明暗模式 | Light + Dark 跟随系统 | 仅 Light / 仅 Dark / 手动 Toggle |
| 6 | 圆角阴影 | 8/12/16 + 多层柔影 | 4/8/12 / 12/16/20 / 全 Pill |
| 7 | 图标系统 | Phosphor | Lucide / Tabler / 全定制 |
| 8 | 动效语言 | Gentle & Restrained | Soft Choreography / Playful / Minimal |
| 9 | App Shell | Top bar + Content | Sidebar+Content / Top+Sidebar 双栏 |
| 10 | Login 页面 | Centered Card on Cream | Split Screen / Full-bleed / Editorial |
| 11 | Dashboard 布局 | 3-Column Smart Expand | Always 3-col / Drawer Overlay / Bottom Panel |
| 12 | SDK 设计立场 | Consistent Brand | Neutral Guest / Calm Distinctive / Mixed |
| 13 | Token 架构 | CSS Variables + Scoped CSS | Tailwind / UnoCSS / 现状正规化 |
| 14 | Replay 外壳 | Custom Calm Chrome + API 控制 | Player+Sidebar / 保留原生 / Fork |

---

## 10. 未决议题(留待实现阶段)

- 实际字体子集化策略(中文 woff2 体积大,需 unicode-range 切片)
- Phosphor 图标的具体子集选择(避免 bundle 全部 6000+ icons)
- App Shell 顶栏在窄屏(<768px)的响应式策略(v1 桌面优先,但 SDK 必须移动端)
- 自定义 logo / wordmark 的具体形式(目前用 `·pinconsole·` 文字签名)
- 自定义滚动条样式(Calm 配色下需要重写默认浏览器滚动条)
- 字体 self-host 的 license 合规(IBM Plex 是 SIL OFL,可商用可重分发)
