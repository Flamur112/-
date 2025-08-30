<template>
  <div class="app-wrapper">
    <!-- Header with Tabs -->
    <div class="header-container">
      <div class="header">
        <div class="logo-section">
          <img src="/favicon.ico" alt="Logo" class="logo-img">
          <span class="logo-text">MulisC2</span>
        </div>
        
        <div class="header-right">
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-avatar :size="32" icon="UserFilled" />
              <span class="username">{{ currentUser?.username || 'User' }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="settings">Settings</el-dropdown-item>
                <el-dropdown-item divided command="serverRestart">Restart Server</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <!-- Navigation Tabs -->
      <div class="nav-tabs">
        <el-tabs v-model="activeTab" type="card" @tab-click="handleTabClick" class="main-tabs">
          <el-tab-pane
            v-for="route in menuRoutes"
            :key="route.path"
            :label="route.meta?.title"
            :name="route.path"
          >
            <template #label>
              <el-icon><component :is="route.meta?.icon" /></el-icon>
              <span>{{ route.meta?.title }}</span>
            </template>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- Main Content -->
    <div class="main-container">
      <!-- Content Area -->
      <div class="content">
        <div style="background: #fff3cd; padding: 10px; margin: 10px 0; border-radius: 4px; border: 2px solid #ffc107;">
          <h3>🔍 Router View Debug</h3>
          <p><strong>Current Route:</strong> {{ route.path }}</p>
          <p><strong>Route Name:</strong> {{ route.name }}</p>
          <p><strong>Route Meta:</strong> {{ JSON.stringify(route.meta) }}</p>
        </div>
        <router-view v-slot="{ Component }">
          <component :is="Component" />
          <div v-if="!Component" style="background: #f8d7da; padding: 20px; margin: 20px 0; border-radius: 4px; border: 2px solid #dc3545;">
            <h3>❌ No Component Rendered</h3>
            <p>The router-view is not rendering any component.</p>
            <p>This indicates a routing or component loading issue.</p>
          </div>
        </router-view>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { authService } from '@/services/auth'
import { listenerService } from '@/services/listener'

const route = useRoute()
const router = useRouter()

const activeTab = ref(route.path)
const currentUser = computed(() => authService.currentUser.value)
const currentPageTitle = computed(() => route.meta?.title || 'Dashboard')

const menuRoutes = computed(() => {
  const allRoutes = router.getRoutes()
  console.log('All routes:', allRoutes.map(r => ({ path: r.path, name: r.name, meta: r.meta })))
  
  const routes = allRoutes.filter(r => r.meta?.title && !r.meta?.hideFromMenu) || []
  console.log('Filtered menu routes:', routes.map(r => ({ path: r.path, name: r.name, meta: r.meta })))
  
  return routes
})

const handleTabClick = (tab: any) => {
  console.log('Tab clicked:', tab)
  
  // Element Plus tabs pass the paneName property
  const routePath = tab?.paneName || tab?.name
  
  if (routePath) {
    console.log('Navigating to:', routePath)
    router.push(routePath)
  } else {
    console.warn('Invalid tab data:', tab)
    console.log('Tab properties:', Object.keys(tab || {}))
  }
}

// Keep activeTab in sync with route
watch(() => route.path, (newPath) => {
  activeTab.value = newPath
})

// Debug route loading
onMounted(() => {
  console.log('Layout mounted, current route:', route.path)
  console.log('Available routes:', router.getRoutes())
  console.log('Menu routes:', menuRoutes.value)
  
  // Check if routes are properly configured
  const allRoutes = router.getRoutes()
  const filteredRoutes = allRoutes.filter(r => r.meta?.title && !r.meta?.hideFromMenu)
  
  console.log('Route configuration check:')
  allRoutes.forEach(r => {
    console.log(`- ${r.path}: title=${r.meta?.title}, hideFromMenu=${r.meta?.hideFromMenu}`)
  })
})



const handleServerRestart = async () => {
  try {
    let hasActiveListener = false
    
    // Check if there's an active listener
    try {
      const status = await listenerService.getStatus()
      hasActiveListener = status.active
      
      if (hasActiveListener) {
        await ElMessageBox.confirm(
          `You have an active C2 listener running on ${status.address}. ` +
          'Restarting the server will stop the C2 listener and disconnect all agents. ' +
          'All agent connections will be lost. Are you sure you want to restart the server?',
          'Server Restart Confirmation',
          {
            confirmButtonText: 'Yes, Restart Server',
            cancelButtonText: 'Cancel',
            type: 'warning',
            confirmButtonClass: 'el-button--danger',
          }
        )
      }
    } catch (statusError) {
      console.warn('Could not check listener status:', statusError)
    }

    // Show message about manual restart
    ElMessage.info('Please restart the server manually using the launcher script (run-mulic2.bat)')
  } catch (error) {
    if (error === 'cancel') {
      return
    }
    console.error('Server restart error:', error)
  }
}

const handleCommand = (command: string) => {
  switch (command) {
    case 'settings':
      // Settings is now handled within the Dashboard
      router.push('/')
      break
    case 'serverRestart':
      handleServerRestart()
      break
  }
}
</script>

<style scoped lang="scss">
.app-wrapper {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: var(--primary-black);
  font-family: 'Courier New', monospace;
}

.header-container {
  background: var(--secondary-black);
  border-bottom: 1px solid var(--accent-red);
  
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 20px;
    border-bottom: 1px solid var(--accent-red);
    
    .logo-section {
      display: flex;
      align-items: center;
      
      .logo-img {
        width: 32px;
        height: 32px;
        margin-right: 12px;
      }
      
      .logo-text {
        color: var(--accent-red);
        font-size: 20px;
        font-weight: bold;
        letter-spacing: 1px;
      }
    }
    
    .header-right {
      .user-info {
        display: flex;
        align-items: center;
        color: var(--text-gray);
        cursor: pointer;
        padding: 8px 12px;
        border-radius: 4px;
        transition: background-color 0.2s;
        border: 1px solid var(--border-color);
        
        &:hover {
          background: var(--medium-gray);
          border-color: var(--accent-red);
          color: var(--text-white);
        }
        
        .username {
          margin: 0 8px;
          font-weight: 500;
          font-size: 14px;
        }
      }
    }
  }
  
  .nav-tabs {
    padding: 0 20px;
    background: var(--secondary-black);
    
    .main-tabs {
      :deep(.el-tabs__header) {
        margin: 0;
        border-bottom: none;
        padding: 0;
      }
      
      :deep(.el-tabs__nav-wrap) {
        &::after {
          display: none;
        }
      }
      
      :deep(.el-tabs__item) {
        color: var(--text-gray);
        border: 1px solid var(--accent-red);
        border-bottom: none;
        border-radius: 6px 6px 0 0;
        margin-right: 4px;
        padding: 12px 20px;
        font-weight: 500;
        transition: all 0.2s;
        font-family: 'Courier New', monospace;
        background: var(--secondary-black);
        
        &:hover {
          color: var(--text-white);
          background: var(--medium-gray);
        }
        
        &.is-active {
          color: var(--text-white);
          background: var(--primary-black);
          border-color: var(--accent-red);
        }
        
        .el-icon {
          margin-right: 8px;
          font-size: 16px;
        }
      }
      
      :deep(.el-tabs__content) {
        display: none;
      }
    }
  }
}

.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background-color: var(--primary-black);
}

.content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  background-color: var(--primary-black);
  color: var(--text-white);
  min-height: 0; /* Important for flexbox */
}

// Dropdown menu overrides
:deep(.el-dropdown-menu) {
  background-color: var(--dark-gray) !important;
  border: 1px solid var(--accent-red) !important;
  
  .el-dropdown-menu__item {
    color: var(--text-gray) !important;
    font-family: 'Courier New', monospace !important;
    
    &:hover {
      background-color: var(--medium-gray) !important;
      color: var(--text-white) !important;
    }
  }
}
</style>

