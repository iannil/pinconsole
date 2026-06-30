# pinconsole

开源的实时访客监控 + 运营互动 + 录像回放工具,AGPL-3.0,对标商业竞品的技术核心。仓库包含两层**界限严格**的代码:**OSS Core**(用户部署)与 **Marketing Layer**(maintainer 专属)。

## 项目分层

**OSS Core**:
本仓库内所有部署给最终用户的代码 — `server/` `admin/` `visitor-sdk/` `landing/` `packages/` `e2e/`。License: AGPL-3.0-or-later。用户自部署的实例**不**回传询盘或分析数据给 maintainer。
_Avoid_: product, app, main repo

**Marketing Layer**:
`marketing/` 目录,独立 Astro + Cloudflare 站点,由 maintainer 维护,用于 ToB 咨询询盘与品牌曝光。License: **UNLICENSED / All Rights Reserved**(非 OSS,与 AGPL 主仓严格隔离)。包含 blog / use-cases / alternatives / lead 表单 / GA。
_Avoid_: website, landing site, landing page(`landing/` 是 OSS 落地页模板)

**Lead**:
Marketing Layer 的 ToB 咨询询盘,通过 marketing 站提交(purpose = evaluate | self-host | custom | compliance | other),存 Cloudflare D1 + Resend 邮件通知 maintainer。**不进入** OSS Core 数据流。
_Avoid_: signup, registration, customer(本项目不做注册流)

## 产品概念

**Visitor**:
被监控的网站访客,SDK 注入其浏览器采集 DOM/事件流。
_Avoid_: user, customer(那些指运营端)

**Operator**:
登录 admin SPA 的运营人员,可 1:1 claim 一个 visitor 进行实时互动。
_Avoid_: agent, admin

**Session**:
一次访客浏览会话,从 SDK 初始化到关闭,可跨多个 page navigation 续接。
_Avoid_: visit, conversation

**Claim**:
Operator 对一个 visitor 的 1:1 排他锁定(claim/release)。仅用于写/control 端点,只读端点不要求 claim。
_Avoid_: lock, assign

## 运营内容编辑

**Widget Config**:
4 类运营 widget(popup / chat / co-browse banner / consent banner)的文案与基础样式 JSON 配置,存 PG `widget_configs` 表,admin 编辑、SDK init 时 GET 渲染。由切片 pe-1/2/3 实现。
_Avoid_: page config, content config

**Widget Config Editor**:
admin SPA 内 `/widgets` 路由的**表单式**编辑器,编辑 Widget Config。**不是** Page Editor。
_Avoid_: widget editor, content editor

**Page Editor**:
未来的**拖拽式** landing page 可视化编辑器(PLAN.md §8 post-v1 backlog,**未启动**)。与 Widget Config Editor 是**两个独立**功能,勿混。
_Avoid_: landing editor, visual editor(在本项目语境中 Page Editor 是专有名词)
