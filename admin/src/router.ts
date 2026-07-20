import { createRouter, createWebHistory } from 'vue-router'

import { authSession } from './auth'
import DashboardLayout from './views/DashboardLayout.vue'
import LoginView from './views/LoginView.vue'
import OverviewView from './views/OverviewView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView },
    {
      path: '/',
      component: DashboardLayout,
      meta: { requiresAdmin: true },
      children: [
        { path: '', name: 'overview', component: OverviewView },
      ],
    },
  ],
})

router.beforeEach(to => {
  if (to.meta.requiresAdmin && !authSession.isAdmin) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && authSession.isAdmin) {
    return { name: 'overview' }
  }
  return true
})
