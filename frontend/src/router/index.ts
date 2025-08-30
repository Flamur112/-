import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from '@/services/auth'
import Dashboard from '../views/Dashboard.vue'
import ProfileSelection from '../views/ProfileSelection.vue'

const routes = [
  {
    path: '/',
    name: 'Dashboard',
    component: Dashboard,
    meta: { title: 'Dashboard', icon: 'DataBoard', requiresAuth: true }
  },
  {
    path: '/profile-selection',
    name: 'ProfileSelection',
    component: ProfileSelection,
    meta: { title: 'Profile Selection', requiresAuth: true, hideFromMenu: true }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

// Navigation guard for SPA
router.beforeEach((to, from, next) => {
  const { isAuthenticated } = useAuth()
  
  // Allow navigation to proceed - let the components handle auth state
  // The App.vue will handle showing auth forms vs main app based on auth state
  next()
})

export default router

