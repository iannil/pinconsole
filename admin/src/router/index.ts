// Vue Router 配置（1b dashboard + 1d replay）

import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('../views/Dashboard.vue'),
  },
  {
    path: '/replay',
    name: 'replay-list',
    component: () => import('../views/ReplayList.vue'),
  },
  {
    path: '/replay/:session_id',
    name: 'replay-viewer',
    component: () => import('../views/ReplayViewer.vue'),
    props: true,
  },
];

export const router = createRouter({
  history: createWebHistory('/admin/'),
  routes,
});
