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
        <router-view />
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
  return allRoutes.filter(route => !route.meta?.hideFromMenu)
})

// Watch for route changes to update active tab
watch(() => route.path, (newPath) => {
  activeTab.value = newPath
})

// Handle tab clicks
const handleTabClick = (tab: any) => {
  if (tab.name !== route.path) {
    router.push(tab.name)
  }
}

// Handle dropdown commands
const handleCommand = async (command: string) => {
  switch (command) {
    case 'settings':
      ElMessage.info('Settings functionality coming soon...')
      break
    case 'serverRestart':
      try {
        const result = await ElMessageBox.confirm(
          'Are you sure you want to restart the server? This will disconnect all active connections.',
          'Restart Server',
          {
            confirmButtonText: 'Restart',
            cancelButtonText: 'Cancel',
            type: 'warning'
          }
        )
        
        if (result === 'confirm') {
          // TODO: Implement server restart
          ElMessage.success('Server restart initiated...')
        }
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error('Failed to restart server.')
        }
      }
      break
  }
}

onMounted(() => {
  // Initialize any required setup
})
</script>

<style scoped>
.app-wrapper {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #f5f7fa;
}

.header-container {
  background: white;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  z-index: 1000;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  height: 60px;
  border-bottom: 1px solid #e4e7ed;
}

.logo-section {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-img {
  width: 32px;
  height: 32px;
}

.logo-text {
  font-size: 20px;
  font-weight: bold;
  color: #409eff;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 15px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.user-info:hover {
  background-color: #f5f7fa;
}

.username {
  font-weight: 500;
  color: #303133;
}

.nav-tabs {
  padding: 0 20px;
  background: white;
  border-bottom: 1px solid #e4e7ed;
}

.main-tabs {
  border: none;
}

.main-tabs :deep(.el-tabs__header) {
  margin: 0;
}

.main-tabs :deep(.el-tabs__nav-wrap) {
  padding: 0;
}

.main-tabs :deep(.el-tabs__item) {
  height: 40px;
  line-height: 40px;
  padding: 0 20px;
  font-weight: 500;
  color: #606266;
  border: none;
  background: transparent;
  transition: all 0.2s;
}

.main-tabs :deep(.el-tabs__item:hover) {
  color: #409eff;
  background: #f0f9ff;
}

.main-tabs :deep(.el-tabs__item.is-active) {
  color: #409eff;
  background: #e6f7ff;
  border-bottom: 2px solid #409eff;
}

.main-tabs :deep(.el-tabs__content) {
  display: none;
}

.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}
</style>

