import { createRouter, createWebHistory } from 'vue-router'
import { useAuth } from '@/services/auth'

// Import components using relative paths
import Dashboard from '../views/Dashboard.vue'
import ProfileSelection from '../views/ProfileSelection.vue'

// Debug imports
console.log('🔍 Router imports debug:')
console.log('Dashboard component:', Dashboard)
console.log('ProfileSelection component:', ProfileSelection)
console.log('Dashboard type:', typeof Dashboard)
console.log('ProfileSelection type:', typeof ProfileSelection)

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

// Verify route components are valid
console.log('🔍 Route component verification:')
routes.forEach((route, index) => {
  if (route.component) {
    console.log(`✅ Route ${index} (${route.path}): Component is valid`)
  } else {
    console.log(`❌ Route ${index} (${route.path}): Component is invalid/undefined`)
  }
})

// Debug routes
console.log('🔍 Routes debug:')
console.log('Routes array:', routes)
routes.forEach((route, index) => {
  console.log(`Route ${index}:`, {
    path: route.path,
    name: route.name,
    component: route.component,
    componentType: typeof route.component
  })
})

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

// Navigation guard for SPA
router.beforeEach((to, from, next) => {
  console.log('🚦 Router navigation:', { from: from.path, to: to.path, toName: to.name })
  const { isAuthenticated } = useAuth()
  
  console.log('🔐 Auth status:', isAuthenticated.value)
  console.log('🔐 Route requires auth:', to.meta.requiresAuth)
  
  // Allow navigation to proceed - let the components handle auth state
  // The App.vue will handle showing auth forms vs main app based on auth state
  console.log('✅ Access granted - proceeding to route (auth handled by components)')
  next()
})

export default router

