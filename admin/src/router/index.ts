// Vue Router 配置(1b dashboard + 1d replay + 1h-ui login)
//
// 2026-06-18 v1-e2e-acceptance 修复:
// router.beforeEach 异步等待 fetchMe 完成。原实现在 App.vue onMounted 调 fetchMe,
// 但 router 守卫在 mount 之前已执行,所以**任何直接 URL 访问或刷新** /admin/*
// 都会被守卫误判为未登录,跳转到 /login。这是 production bug:用户刷新 dashboard
// 会被强制退出。
//
// 修复方式:第一次 navigation 触发时 lazy 调 fetchMe,await 之后再判断 auth 状态。
//
// design-system Phase 2:嵌套路由重构 —— /login 独立 public 路由,
// 其他认证页面挂在 AppShell(顶栏 + 内容)的 children 下。

import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('../layouts/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/dashboard' },
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('../views/Dashboard.vue'),
      },
      {
        path: 'replay',
        name: 'replay-list',
        component: () => import('../views/ReplayList.vue'),
      },
      {
        path: 'replay/:session_id',
        name: 'replay-viewer',
        component: () => import('../views/ReplayViewer.vue'),
        props: true,
      },
      {
        path: 'privacy',
        name: 'privacy',
        component: () => import('../views/Privacy.vue'),
      },
      {
        path: 'widgets',
        name: 'widgets',
        component: () => import('../views/WidgetsView.vue'),
      },
      {
        path: 'pages',
        name: 'pages',
        component: () => import('../views/PagesView.vue'),
      },
      {
        path: 'pages/:slug/edit',
        name: 'page-editor',
        component: () => import('../views/PageEditorView.vue'),
        props: true,
      },
      {
        path: 'pages/:slug/leads',
        name: 'page-leads',
        component: () => import('../views/PageLeadsView.vue'),
        props: true,
      },
    ],
  },
];

export const router = createRouter({
  history: createWebHistory('/admin/'),
  routes,
});

// 首次 navigation 时触发 fetchMe,后续复用同一 Promise。
// fetchMe 失败(cookie 失效等)不抛错,内部会把 user.value 置 null。
let initPromise: Promise<void> | null = null;
function ensureAuthInit(): Promise<void> {
  if (!initPromise) {
    const auth = useAuthStore();
    initPromise = auth.fetchMe().catch(() => {
      // fetchMe 内部已处理错误(user = null),此处兜底防止 Promise reject 阻塞 router
    });
  }
  return initPromise;
}

router.beforeEach(async (to, _from, next) => {
  // 等 fetchMe 完成,避免首次 navigation 时 user 还未加载就被判定未登录
  await ensureAuthInit();

  const auth = useAuthStore();

  if (to.name === 'login' && auth.isAuthenticated) {
    next({ path: '/dashboard' });
    return;
  }

  if (to.meta.public) {
    next();
    return;
  }

  // 用 matched.some 检查父级(AppShell)的 requiresAuth meta。
  // to.meta 会浅 merge 父子,但 matched.some 是 Vue Router 推荐的 canonical 写法。
  if (to.matched.some((record) => record.meta.requiresAuth) && !auth.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  next();
});
