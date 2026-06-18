// Vue Router 配置(1b dashboard + 1d replay + 1h-ui login)

import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
    meta: { public: true },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/replay',
    name: 'replay-list',
    component: () => import('../views/ReplayList.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/replay/:session_id',
    name: 'replay-viewer',
    component: () => import('../views/ReplayViewer.vue'),
    props: true,
    meta: { requiresAuth: true },
  },
  {
    path: '/privacy',
    name: 'privacy',
    component: () => import('../views/Privacy.vue'),
    meta: { requiresAuth: true },
  },
];

export const router = createRouter({
  history: createWebHistory('/admin/'),
  routes,
});

// 1h-ui:全局 beforeEach 守卫
router.beforeEach((to, _from, next) => {
  const auth = useAuthStore();

  // 已认证访问 /login → 跳 dashboard
  if (to.name === 'login' && auth.isAuthenticated) {
    next({ path: '/dashboard' });
    return;
  }

  // 公共路由(/login)直接放行
  if (to.meta.public) {
    next();
    return;
  }

  // 需要 auth 但未认证 → 跳 login 带 redirect
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  next();
});
