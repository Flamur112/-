import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from '@/services/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'Dashboard',
      component: () => import('../views/Dashboard.vue'),
      meta: { title: 'Dashboard', icon: 'DataBoard', requiresAuth: true }
    },
    {
      path: '/profile-selection',
      name: 'ProfileSelection',
      component: () => import('../views/ProfileSelection.vue'),
      meta: { title: 'Profile Selection', requiresAuth: true, hideFromMenu: true }
    },
    {
      path: '/agents',
      name: 'Agents',
      component: () => import('../views/Agents.vue'),
      meta: { title: 'Agents', icon: 'Monitor', requiresAuth: true }
    },
    {
      path: '/tasks',
      name: 'Tasks',
      component: () => import('../views/Tasks.vue'),
      meta: { title: 'Tasks', icon: 'List', requiresAuth: true }
    },
    {
      path: '/logs',
      name: 'Logs',
      component: () => import('../views/Logs.vue'),
      meta: { title: 'Logs', icon: 'Document', requiresAuth: true }
    },
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('../views/Settings.vue'),
      meta: { title: 'Settings', icon: 'Setting', requiresAuth: true }
    },
    {
      path: '/terminal',
      name: 'Terminal',
      component: () => import('../views/Terminal.vue'),
      meta: { title: 'Terminal', icon: 'Terminal', requiresAuth: true }
    },

  ]
})

// Navigation guard for SPA
router.beforeEach((to, from, next) => {
  const { isAuthenticated } = useAuth()
  
  // If user is not authenticated and trying to access protected routes, stay on same page
  // The App.vue will handle showing auth forms vs main app
  if (to.meta.requiresAuth && !isAuthenticated.value) {
    // Stay on current page, App.vue will show auth forms
    next(false)
  } else {
    next()
  }
})

export default router

