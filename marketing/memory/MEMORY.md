# pinconsole 项目长期记忆

> 整理后的当前真实状态。详细决策理由参见 `PLAN.md` 与 `docs/`。

## 项目上下文

- 单租户 OSS ToB 实时访客监控平台，AGPL-3.0，对标商业竞品开源替代
- v1 切片拆为 1a-1j 共 10 个子切片，按 PLAN.md §7 推进
- 仓库结构：admin/ + visitor-sdk/ + server/ + marketing/ + landing/
- marketing 是 **maintainer 独立基础设施**（不为 OSS 用户分发），部署在 Cloudflare，目标是 ToB 咨询 lead gen

## 关键决策

### Marketing v2 设计 DNA（2026-06-22 锁定）

**驱动**：v1 Editorial Manifesto（Fraunces serif + Cream 纸纹 + 报刊框）气质过于 indie/engineer，年费 30-100 万的 ToB 决策者（CTO/运营 VP）会觉得"小作坊"。

**v2 设计方向**：Linear 黑调极简（modern tool brand 流派）
- **背景**：近黑 `#08090A` + 顶部弱 emerald radial spotlight
- **强调色**：Emerald `#10B981` / `#34D399`（与 admin Teal 小偏差，marketing 独立可接受）
- **字体**：IBM Plex Sans / Mono + Noto Sans SC（OFL、与 admin 连贯、非 Inter/Geist/Space Grotesk）
- **Hero**：Type + Code/Install block（Vercel 牌路，与 OSS / self-host DNA 同源）
- **视觉语言**：Pure Linear——1px subtle border、几乎无 shadow、极简动效（200ms opacity + 1px shift）、无 grain/无 ASCII
- **Voice**：Hybrid——Hero/CTA 保留 manifesto 措辞，Features/Roadmap/FAQ 走事实型
- **IA 调整**：合并 DataSovereignty + SelfHost → Why pinconsole；Problem 缩一句进 Hero subhead；FAQ 从 10 项裁到 6；Header 下加 credibility strip（commits/tests/AGPL/self-host）
- **FAQ**：Accordion（默认展开第 1 项）

**退役**：Cream 纸纹 / Fraunces serif / Noto Serif SC / drop cap / "Issue 01 · 立场" 报刊框 / 4px 黑色 rule / heavy editorial ornaments

**不变项**：admin 设计系统不动（Stone+Teal+Amber + Phosphor + IBM Plex）；AGPL-3.0 license；self-host 单二进制部署；i18n 中英双语 from day 1

### 反 AI-slop 红线（继承 frontend-design skill）

- ❌ 紫渐变 / Inter/Roboto/Arial / slate+indigo / emoji-icon
- ❌ Space Grotesk / Geist（泛滥字体）
- ❌ "标准 SaaS" 圆角 + 单层 box-shadow + 居中 hero + 三卡片 grid
- ❌ Cream 纸纹 / serif / drop cap（v1 Editorial Manifesto 全部退役）

## 用户偏好

- 中文交流、英文代码、Markdown 文档放 `docs/`
- 文档：每次修改都更新 `docs/progress/`（未完成）或 `docs/reports/completed/`（已完成）
- 测试深度三级：🟢 verified-deep / 🟡 verified-shallow / 🔴 implemented-unverified
- 批量改动前先将原文件备份至 `/backup`；异常错误数立即回滚
- 键盘监听类功能必须主动提示 GDPR/CCPA 合规风险

## 外部资源引用

- PLAN.md — 架构、产品定位、技术栈、切片拆分、决策理由
- docs/design-system.md — admin 设计基线（Calm Crafted · Stone+Teal+Amber · Phosphor）
- docs/standards/verification-depth.md — 测试深度判定
