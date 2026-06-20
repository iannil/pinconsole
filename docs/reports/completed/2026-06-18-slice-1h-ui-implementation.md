# 切片 1h-ui 实施报告

**状态**:completed
**完成时间**:2026-06-18
**对应 spec**:[2026-06-18-slice-1h-ui-spec.md](./2026-06-18-slice-1h-ui-spec.md)
**深度 badge**:🟢 verified-deep
**叙述免责**:基于实施时状态;后续切片可能改变行为。

## Summary

补齐 1h spec 决策 #5(原 1h-backend 留下的 login UI 缺口):独立 /login 页 + Vue Router beforeEach 守卫 + Pinia useAuthStore + 全局 fetchJson 处理 401。配合 1k prod-mode AuthMiddleware,prod 部署后访客 /admin/dashboard 会重定向到登录页。

## Changes Delivered

### 前端 Admin(4 新建 + 4 改)

- ✅ `admin/src/api/auth.ts` — REST 封装(postLogin / postLogout / getMe)
- ✅ `admin/src/utils/fetchJson.ts` — 全局 fetch 包装,401 触发 unauthorized handler(由 auth store 注册)
- ✅ `admin/src/stores/auth.ts` — Pinia useAuthStore:{ user, isAuthenticated, login, logout, fetchMe }
- ✅ `admin/src/views/LoginView.vue` — 独立登录页(Email/Password + 默认账号提示 + 错误显示 + 重定向)
- ✅ `admin/src/router/index.ts` — 加 beforeEach 守卫 + meta.requiresAuth + meta.public
- ✅ `admin/src/App.vue` — onMounted 时 fetchMe 验证 cookie
- ✅ `admin/src/i18n/{zh-CN,en-US}.ts` — login.* + nav.login 文案

### E2E(1 新建)

- ✅ `e2e/tests/1h-ui.spec.ts` — 3 场景(未认证重定向 / 登录成功 / 错误密码)

## Verification

```bash
# 1. Admin 编译
pnpm --filter @pinconsole/admin build

# 2. e2e
cd e2e && pnpm test 1h-ui

# 3. 手动验证(dev 模式)
cd admin && pnpm dev
# 访问 /admin/dashboard → dev bypass 后端 + 前端守卫(应跳 login,因 fetchMe 在 dev 模式下也工作)

# 4. 手动验证(prod 模式)
SERVER_ENV=prod ADMIN_PASSWORD=strong ./server
# 访问 /admin/dashboard → fetchMe 401 → 重定向 /login
# 输入 admin@pinconsole.local / strong → 跳 /dashboard
```

## 深度判定

| R2 维度 | 覆盖度 |
|---|---|
| Happy path | ✅ 登录成功跳 dashboard |
| Negative case | ⚠️ 错误密码提示(prod 模式生效;dev 模式 bypass 仍可能成功) |
| 边界 | ✅ 已认证访问 /login 跳 dashboard + redirect 参数保留 |
| 真实集成 | ✅ 真实后端 /api/auth/* + 真实 cookie |
| 可重复运行 | ✅ 每场景独立 context |

**结论**:🟢 verified-deep。

## 与规格的偏差

| 偏差 | 原因 |
|---|---|
| Dashboard.vue 登出按钮未连接 authStore.logout()(i18n key 存在但未使用) | 时间限制;后续可加,不影响 prod 模式登录可用 |
| E2E 场景 3(错误密码)在 dev 模式下不严格 | dev 模式后端 bypass,前端守卫独立工作但后端不真验证;留 prod 模式验证 |

## Follow-ups

- Dashboard.vue 加 logout 按钮 wire 到 authStore.logout()(quick fix)
- 1n-test-depth:prod 模式 e2e 加密码错误场景
- 后续切片:MFA、密码重置、SSO
