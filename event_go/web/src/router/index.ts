import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/EventList.vue'),
    },
    {
      path: '/organizers',
      name: 'organizers',
      component: () => import('@/views/OrganizerList.vue'),
    },
    {
      path: '/organizers/:id',
      name: 'organizer-detail',
      component: () => import('@/views/OrganizerDetail.vue'),
    },
    {
      path: '/events/:id',
      name: 'event-detail',
      component: () => import('@/views/EventDetail.vue'),
    },
    {
      path: '/events/:id/discussion',
      name: 'discussion',
      component: () => import('@/views/Discussion.vue'),
    },
    {
      path: '/events/:id/posts/:postId',
      name: 'post-detail',
      component: () => import('@/views/PostDetail.vue'),
    },
    {
      path: '/admin',
      name: 'admin-login',
      component: () => import('@/views/admin/Login.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/Login.vue'),
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/auth/Register.vue'),
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('@/views/auth/ForgotPassword.vue'),
    },
    {
      path: '/reset-password',
      name: 'reset-password',
      component: () => import('@/views/auth/ResetPassword.vue'),
    },
    {
      path: '/verify-recovery-email',
      name: 'verify-recovery-email',
      component: () => import('@/views/auth/VerifyRecoveryEmail.vue'),
    },
    {
      path: '/me/registrations',
      name: 'my-registrations',
      component: () => import('@/views/MyRegistrations.vue'),
      meta: { requiresUser: true },
    },
    {
      path: '/me/security',
      name: 'account-security',
      component: () => import('@/views/AccountSecurity.vue'),
      meta: { requiresUser: true },
    },
    {
      path: '/me/notifications',
      name: 'notifications',
      component: () => import('@/views/Notifications.vue'),
      meta: { requiresUser: true },
    },
    {
      path: '/workspace',
      name: 'workspace-list',
      component: () => import('@/views/workspace/WorkspaceList.vue'),
      meta: { requiresUser: true },
    },
    {
      path: '/workspace/new',
      name: 'workspace-new',
      component: () => import('@/views/workspace/WorkspaceCreate.vue'),
      meta: { requiresUser: true },
    },
    {
      path: '/workspace/:organizationId',
      name: 'workspace-dashboard',
      component: () => import('@/views/workspace/WorkspaceDashboard.vue'),
      meta: { requiresUser: true, requiresOrganization: true },
    },
    {
      path: '/workspace/:organizationId/members',
      name: 'workspace-members',
      component: () => import('@/views/workspace/WorkspaceMembers.vue'),
      meta: { requiresUser: true, requiresOrganization: true },
    },
    {
      path: '/workspace/:organizationId/events/new',
      name: 'workspace-event-new',
      component: () => import('@/views/workspace/WorkspaceEventForm.vue'),
      meta: { requiresUser: true, requiresOrganization: true },
    },
    {
      path: '/workspace/:organizationId/events/:id/edit',
      name: 'workspace-event-edit',
      component: () => import('@/views/workspace/WorkspaceEventForm.vue'),
      meta: { requiresUser: true, requiresOrganization: true },
    },
    {
      path: '/workspace/:organizationId/events/:id/tickets',
      name: 'workspace-tickets',
      component: () => import('@/views/workspace/WorkspaceTickets.vue'),
      meta: { requiresUser: true, requiresOrganization: true },
    },
    {
      path: '/workspace/:organizationId/events/:id/operations',
      name: 'workspace-operations',
      component: () => import('@/views/workspace/WorkspaceOperations.vue'),
      meta: { requiresUser: true, requiresOrganization: true },
    },
    {
      path: '/organization-invitations/accept',
      name: 'organization-invitation-accept',
      component: () => import('@/views/workspace/InvitationAccept.vue'),
      meta: { requiresUser: true },
    },
    {
      path: '/admin/events',
      name: 'admin-events',
      component: () => import('@/views/admin/EventManage.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/organizers',
      name: 'admin-organizers',
      component: () => import('@/views/admin/OrganizerManage.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/content',
      name: 'admin-content',
      component: () => import('@/views/admin/ContentModeration.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/organizers/new',
      name: 'admin-organizer-new',
      component: () => import('@/views/admin/OrganizerForm.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/organizers/:id/edit',
      name: 'admin-organizer-edit',
      component: () => import('@/views/admin/OrganizerForm.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/events/new',
      name: 'admin-event-new',
      component: () => import('@/views/admin/EventForm.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/events/:id/edit',
      name: 'admin-event-edit',
      component: () => import('@/views/admin/EventForm.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/events/:id/registrations',
      name: 'admin-registrations',
      component: () => import('@/views/admin/Registrations.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/admin/events/:id/tickets',
      name: 'admin-tickets',
      component: () => import('@/views/admin/TicketManage.vue'),
      meta: { requiresAdmin: true },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFound.vue'),
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.requiresUser && !localStorage.getItem('user_token')) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAdmin && !localStorage.getItem('admin_token')) {
    return { path: '/admin', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAdmin) {
    try {
      const { getAdminSession, isPlatformAdminSession } = await import('@/api/admin')
      const response = await getAdminSession()
      if (!isPlatformAdminSession(response.data)) throw new Error('platform admin session was not confirmed')
    } catch {
      useAuthStore().logout()
      return { path: '/admin', query: { redirect: to.fullPath } }
    }
  }
  if (to.meta.requiresOrganization) {
    const organizationId = Number(to.params.organizationId)
    if (!Number.isInteger(organizationId) || organizationId <= 0) return { path: '/workspace' }
    try {
      const { useWorkspaceStore } = await import('@/stores/workspace')
      await useWorkspaceStore().loadSession(organizationId)
    } catch {
      return { path: '/workspace' }
    }
  }
})

export default router
