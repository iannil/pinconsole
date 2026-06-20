# 切片 1h-ui 规格(admin LoginView + 路由守卫)

**状态**:in-progress
**开始**:2026-06-18
**对应 1h spec**:[2026-06-17-slice-1h-spec.md](./2026-06-17-slice-1h-spec.md) 决策 #5(原切片未实施部分)

## Context

1h-backend 已实施认证 API(1h-implementation 报告),但 LoginView 和 Vue Router 守卫一直未做(1h spec partial)。1k 已将 SERVER_ENV 默认改 prod,prod 模式下未认证访问 protected 端点返回 401,但 admin SPA 无登录入口,用户无法登录。本切片补齐 1h spec 决策 #5。

## 范围

### In scope

- `admin/src/views/LoginView.vue` — 独立 /login 页,Email/Password 表单,中英双语,默认账号提示
- `admin/src/stores/auth.ts` — Pinia useAuthStore:{ user, isAuthenticated, login(), logout(), fetchMe() }
- `admin/src/api/auth.ts` — REST 封装(login / logout / me)
- `admin/src/utils/fetchJson.ts` — 全局 fetch 包装:401 → 调 authStore.logout() + redirect /login
- `admin/src/router/index.ts` — 加 beforeEach 守卫:`requiresAuth` meta;未认证跳 /login?redirect=...
- LoginView.vue 中英双语 i18n key
- App.vue 加载时 fetchMe 验证 cookie
- Dashboard.vue 已有登出按钮,改用 authStore.logout()
- E2E 3 场景(未认证重定向 / 登录成功 / 错误密码)

### Out of scope

- MFA(留给后续)
- 注册/邀请流(项目明确不做)
- 密码强度校验(后端层处理)
- session 自动刷新(token rotation)

## 锁定决策

| # | 决策 |
|---|---|
| 1 | 独立 /login 页 + Vue Router beforeEach 守卫 |
| 2 | Pinia useAuthStore 管理认证状态 |
| 3 | 全局 fetchJson 包装统一处理 401 |
| 4 | App.vue 加载时 fetchMe 验证 cookie;过期 redirect /login |
| 5 | E2E 3 场景 |
| 6 | 中英双语 i18n key |
| 7 | 目标 🟢 |

## Verification

```bash
# dev 模式(SERVER_ENV=dev)
cd admin && pnpm dev
# 访问 /admin/dashboard → dev bypass,无需登录可见

# prod 模式(SERVER_ENV=prod)
SERVER_ENV=prod ADMIN_PASSWORD=strong ./server
# 访问 /admin/dashboard → 重定向 /admin/login
# 输入 admin@pinconsole.local / strong → 跳回 /dashboard

# e2e
cd e2e && pnpm test 1h-ui
```

## 估时

solo 全职:0.5-1 天
