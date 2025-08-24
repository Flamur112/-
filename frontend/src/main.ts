import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import './styles/index.scss'
import { initializePorts } from './config/ports'
import { authService } from './services/auth'

// Initialize port configuration before starting the app
const startApp = async () => {
  try {
    // Initialize ports - this will throw an error if config is invalid
    await initializePorts()
    console.log('Port configuration loaded successfully')
    
    const app = createApp(App)

    // Register Element Plus icons
    for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
      app.component(key, component)
    }

    app.use(createPinia())
    app.use(router)
    app.use(ElementPlus)

    // Check backend health before mounting
    try {
      const isBackendHealthy = await authService.checkBackendHealth()
      
      if (!isBackendHealthy) {
        // If backend is down, clear all auth state immediately
        console.warn('Backend is unavailable on startup, clearing auth state')
        localStorage.removeItem('auth_token')
        localStorage.removeItem('user_data')
        localStorage.removeItem('active_profile_id')
      }
    } catch (error) {
      console.warn('Backend health check failed on startup:', error)
      // Clear auth state on health check failure
      localStorage.removeItem('auth_token')
      localStorage.removeItem('user_data')
      localStorage.removeItem('active_profile_id')
    }

    app.mount('#app')

    // Initialize auth service and start health check
    try {
      await authService.init()
      authService.startHealthCheck()
      console.log('Auth service initialized and health check started')
    } catch (error) {
      console.warn('Auth service initialization failed:', error)
    }
  } catch (error) {
    console.error('Failed to start app due to port configuration error:', error)
    // Show error to user
    document.body.innerHTML = `
      <div style="padding: 20px; font-family: Arial, sans-serif; color: #f56c6c;">
        <h1>❌ MulisC2 Failed to Start</h1>
        <p><strong>Error:</strong> ${error instanceof Error ? error.message : 'Unknown error'}</p>
        <p>Please check your <code>frontend/config.json</code> file and ensure it contains valid port configuration.</p>
        <p>Example configuration:</p>
        <pre style="background: #f5f5f5; padding: 10px; border-radius: 4px;">
{
  "backend": {
    "api_port": 8080,
    "c2_default_port": 8081
  },
  "frontend": {
    "port": 5173
  }
}</pre>
      </div>
    `
  }
}

startApp()
