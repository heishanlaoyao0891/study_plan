import { createRouter, createWebHistory } from 'vue-router'

import { authSession } from './auth'
import DashboardLayout from './views/DashboardLayout.vue'
import AIConfigView from './views/AIConfigView.vue'
import AuditLogsView from './views/AuditLogsView.vue'
import LoginView from './views/LoginView.vue'
import OverviewView from './views/OverviewView.vue'
import SlackConfigView from './views/SlackConfigView.vue'
import SubscriptionConfigView from './views/SubscriptionConfigView.vue'
import UserDetailView from './views/UserDetailView.vue'
import UsersView from './views/UsersView.vue'

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
        { path: 'users', name: 'users', component: UsersView },
        { path: 'users/:id', name: 'user-detail', component: UserDetailView },
        { path: 'slack-config', name: 'slack-config', component: SlackConfigView },
        { path: 'ai-config', name: 'ai-config', component: AIConfigView },
        { path: 'subscription-config', name: 'subscription-config', component: SubscriptionConfigView },
        { path: 'audit-logs', name: 'audit-logs', component: AuditLogsView },
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
