import { createRouter, createWebHistory } from 'vue-router'

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
      path: '/admin/events',
      name: 'admin-events',
      component: () => import('@/views/admin/EventManage.vue'),
    },
    {
      path: '/admin/events/new',
      name: 'admin-event-new',
      component: () => import('@/views/admin/EventForm.vue'),
    },
    {
      path: '/admin/events/:id/edit',
      name: 'admin-event-edit',
      component: () => import('@/views/admin/EventForm.vue'),
    },
    {
      path: '/admin/events/:id/registrations',
      name: 'admin-registrations',
      component: () => import('@/views/admin/Registrations.vue'),
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFound.vue'),
    },
  ],
})

export default router
