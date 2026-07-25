import { createRouter, createWebHistory } from 'vue-router'

import { authSession } from './auth'
import DashboardLayout from './views/DashboardLayout.vue'
import AIConfigView from './views/AIConfigView.vue'
import AuditLogsView from './views/AuditLogsView.vue'
import LoginView from './views/LoginView.vue'
import OverviewView from './views/OverviewView.vue'
import OpsContentView from './views/OpsContentView.vue'
import FeedbackView from './views/FeedbackView.vue'
import InvitationsView from './views/InvitationsView.vue'
import SlackConfigView from './views/SlackConfigView.vue'
import SubscriptionConfigView from './views/SubscriptionConfigView.vue'
import SuspiciousRecordsView from './views/SuspiciousRecordsView.vue'
import UserDetailView from './views/UserDetailView.vue'
import UsersView from './views/UsersView.vue'

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
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
        { path: 'suspicious-records', name: 'suspicious-records', component: SuspiciousRecordsView },
        { path: 'ai-config', name: 'ai-config', component: AIConfigView },
        { path: 'subscription-config', name: 'subscription-config', component: SubscriptionConfigView },
        { path: 'ops-content', name: 'ops-content', component: OpsContentView },
        { path: 'feedback', name: 'feedback', component: FeedbackView },
        { path: 'invitations', name: 'invitations', component: InvitationsView },
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
