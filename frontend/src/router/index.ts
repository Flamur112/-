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

