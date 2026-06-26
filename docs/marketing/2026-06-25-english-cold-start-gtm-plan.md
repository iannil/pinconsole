# 英语市场冷启动推广方案（grill-me 7 轮访谈合成）

**状态**：in_progress（待用户确认执行）
**开始**：2026-06-25
**关联**：
- 与 [`docs/progress/2026-06-22-landing-readme-design.md`](./2026-06-22-landing-readme-design.md) **并行不冲突**——那份是中文站（国内决策者）；本份是英语冷启动（工程布道者）。双轨并行，互不污染。
- 产品本身仍 OSS 自托管、AGPL-3.0、不计费、不做注册流。本方案只动 maintainer 的对外推广层。

## TL;DR

把对外楔子从"三合一"收窄成 **co-browsing**，对标 Upscope/Cobrowse.io；冷启动先打**合规约束行业（金融科技/保险/医疗/政务类 SaaS）里会自己 self-host 的工程布道者**，地域 EU/UK + US；用一个**公开双端 Demo 沙盒**做引流锚点，**Show HN 单点引爆**，英语 CTA 反转为 Demo/Star 优先、咨询降为三级；AGPL 加一句"商业许可可谈"把采购禁区反转成线索；单人单线作战，靠 90 天指标闸门决定加码或转向。

## 7 项根级决策

| # | 决策点 | 选择 | 理由 |
|---|---|---|---|
| 1 | 对外楔子品类 | **Co-browsing**（"open-source Upscope/Cobrowse.io alternative"），录像/监控降级为附带能力 | OSS 空地（session replay 有 OpenReplay 占位，正面打吃亏）；自托管动机最硬（屏幕共享合规）；付费意愿高 |
| 2 | 冷启动滩头人群 | 合规约束行业的**工程布道者**（adopter），EU/UK + US；buyer 层后置靠咨询转化 | 对他们自托管是刚需非锦上添花（Schrems II/GDPR/HIPAA 不让屏幕数据送美国 SaaS）；愿忍 self-host 摩擦；天然内部 champion |
| 3 | 试用资产 | **公开双端 Demo 沙盒**列为前置必做项 | co-browse 的啊哈时刻必须亲手摸；是所有英语渠道的引流锚点；没它冷启动天花板被锁死 |
| 4 | 英语站主 CTA | **反转**：Try live demo / Star / Self-host 为主，咨询降为三级"talk to maintainer" | 工程布道者反感 lead-gen 墙；先采用再变现；star 是英语冷启动硬通货 |
| 5 | 渠道与节奏 | 三波节奏 + **Show HN 单点引爆**；英语侧**点名对标**做 SEO/对比页 | 一发首映子弹集中打；英语世界"X 的开源替代品"是标准打法（与中文站"不提竞品"双轨相反） |
| 6 | AGPL 措辞 | 公开姿态不变 + 一句 **"commercial license available, talk to maintainer"** | 把企业采购 AGPL 禁令从劝退反转成最强咨询触发；不搭真双 license 基建，零增量负担 |
| 7 | 成功定义 | 90 天指标及格线 + 三道决策闸门 + **单线作战护栏** | 单人最大风险是精力撒在没回报的渠道；先打透一波拿信号再决定 |

## 执行计划

### 第 0 波 · 起跑线（launch 前 2-3 周，攒弹药）

- [ ] **Demo 沙盒上线 + 加固**（前置项）：
  - 访客打开 `demo.pinconsole.com` → 自动出现在公开只读运营台
  - 用户同时点开运营台侧，实时看到自己的鼠标/点击/滚动被镜像 —— 一个人体验双端魔法时刻
  - 预置一段录像回放样本，避免冷场
  - 防滥用：rate limit + 会话 TTL + 沙盒隔离（复用现成 ratelimit/fingerprint 栈）
- [ ] **GitHub repo 打磨**：README 顶部一句话定位（co-browse 楔子）、demo gif、`good first issue` 标签、5 分钟 quickstart 可复现
- [ ] **预埋 SEO 落地页**（英语站，点名对标）：`/alternatives/upscope`、`/alternatives/cobrowse-io`、`/co-browsing/self-hosted`
- [ ] **提 PR 进 awesome-lists**：awesome-selfhosted、awesome-privacy、awesome-customer-support

### 第 1 波 · 首映日（集中火力，单点引爆）

- [ ] **Show HN**（中心引爆点）：标题 `Show HN: pinconsole – self-hosted, open-source co-browsing (Upscope alternative)`；正文讲"为什么合规团队不能用 SaaS co-browse"的真实痛点故事，附 Demo 链接
- [ ] 同日发 **r/selfhosted、r/opensource、Lobsters**
- [ ] Reddit 垂直：**r/devops、r/sysadmin**（合规自托管动机最强）

### 第 2 波 · 持续渗透（launch 后 4-8 周，复利）

- [ ] 二级垂直社区：r/CustomerSuccess、CX/客服论坛、合规/GDPR 圈子（讲数据驻留故事，不硬推）
- [ ] 技术博客：`Why we can't send screen-share data to US SaaS (and what we built instead)` → 同步 dev.to / HN
- [ ] 持续养 SEO 对比页 + 收集早期 self-host case 回流官网

### 英语站文案改动（相对现有 marketing/）

- 主 CTA 三件套：**Try the live demo** / **Star on GitHub** / **Self-host in 5 min**
- 咨询入口降为三级，措辞：`Need it deployed / customized / audited? → Talk to the maintainer`
- AGPL 措辞：`AGPL-3.0 by default. Need a commercial license for closed-source/internal use? → Talk to the maintainer.`
- Hero 楔子从"三合一"收窄到 co-browse：`Self-hosted, open-source co-browsing. Your screen-share data never leaves your infra.`

## 90 天成功指标与决策闸门

| 层 | 指标 | 及格线 |
|---|---|---|
| 关注度 | GitHub star | Show HN 当天 +300，90 天 1k |
| 激活 | Demo 沙盒独立体验数 | 90 天 2k+ |
| 真采用 | self-host 信号 / "我跑起来了" issue & discussion | 90 天 20-30 个真实自部署 |
| 变现线索 | 咨询入口 inbound（含 deploy/license 邮件） | 90 天 ≥5 条合格询盘 |

**三道闸门：**
1. Show HN 当天进不了首页前 30 → 定位措辞问题，改标题/hero，两周后 r/selfhosted 二次引爆，不否定方向
2. Demo 体验高但 self-host issue ≈ 0 → 摩擦太大，全压降低部署门槛（一键 docker/托管试用），停止铺渠道
3. self-host 起来但 0 询盘 → 采用-变现链路断，强化"商业许可/部署服务可谈"可见度，主动去采用者公司找 buyer

**产能护栏**：单人同一时间只维护一条主战线；中文站维持现状不动；英语冷启动单独跑，先打透第 1 波拿信号再决定下一步，不并行铺 5 个渠道。

## 与现有架构/文档的衔接

- Demo 沙盒复用现成栈：ratelimit（Redis 滑动窗口）、fingerprint/antiscrape、GDPR consent —— 增量主要是公开实例托管 + 双端演示编排
- 英语 SEO 落地页落在 `marketing/`（Astro content collections + i18n），与中文内容镜像但 CTA 反转、且英语侧点名对标
- 不触碰 OSS `landing/`（用户部署的纯演示模板）
