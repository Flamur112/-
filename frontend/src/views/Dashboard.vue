<template>
  <div class="dashboard">
    <h1>🎯 Dashboard</h1>
    
    <!-- Success message -->
    <div style="background: #e8f5e8; padding: 15px; margin: 15px 0; border-radius: 8px; border: 2px solid #67c23a;">
      <h3>✅ Dashboard Component Successfully Loaded!</h3>
      <p>The router is now working correctly.</p>
    </div>
    
    <!-- Listeners Panel -->
    <div class="listeners-panel">
      <div class="panel-header">
        <h3>Listener Profiles</h3>
        <div style="display: flex; gap: 10px;">
          <el-button type="info" @click="loadProfiles">
            <el-icon><Refresh /></el-icon>
            Refresh Profiles
          </el-button>
        </div>
      </div>
      
      <!-- Debug Info -->
      <div style="background: #f0f0f0; padding: 10px; margin: 10px 0; border-radius: 4px;">
        <h4>Debug Info:</h4>
        <p><strong>Listeners length:</strong> {{ listeners.length }}</p>
        <p><strong>Listeners data:</strong> {{ JSON.stringify(listeners, null, 2) }}</p>
      </div>
      
      <!-- Empty State -->
      <div v-if="listeners.length === 0" class="empty-state">
        <el-icon size="64" color="#909399"><Setting /></el-icon>
        <h3>No Listener Profiles</h3>
        <p>Create a listener profile to start accepting connections.</p>
        <p><strong>Debug: Listeners length = {{ listeners.length }}</strong></p>
      </div>
      
      <!-- Listeners Table -->
      <el-table v-else :data="listeners" style="width: 100%" class="listeners-table">
        <el-table-column prop="name" label="Profile Name" width="150" />
        <el-table-column prop="protocol" label="Protocol" width="100" />
        <el-table-column prop="host" label="Host" width="150" />
        <el-table-column prop="port" label="Port" width="100" />
        <el-table-column prop="isActive" label="Status" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.isActive ? 'success' : 'info'">
              {{ scope.row.isActive ? 'Active' : 'Inactive' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="connections" label="Connections" width="100" />
        <el-table-column prop="createdAt" label="Created" width="180" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

// Reactive data
const listeners = ref<any[]>([])

// Load profiles function
const loadProfiles = async () => {
  console.log('🔄 Starting to load profiles...')
  try {
    console.log('📡 Making request to /api/profile/list...')
    const response = await fetch('/api/profile/list')
    
    console.log('📥 Response received:', response.status, response.statusText)
    if (!response.ok) {
      console.error('❌ Profile API not available - server returned:', response.status, response.statusText)
      ElMessage.error(`Failed to load profiles: Server returned ${response.status} ${response.statusText}`)
      listeners.value = []
      return
    }
    
    const data = await response.json()
    console.log('📊 Raw response data:', data)
    const profiles = data.profiles || []
    console.log('📋 Profiles found:', profiles.length)
    
    // Convert profiles to listeners format
    const mappedListeners = profiles.map((profile: any) => ({
      id: profile.id,
      name: profile.name,
      projectName: profile.projectName,
      host: profile.host,
      port: profile.port,
      description: profile.description,
      useTLS: profile.useTLS,
      certFile: profile.certFile,
      keyFile: profile.keyFile,
      pollInterval: profile.pollInterval,
      isActive: profile.isActive,
      createdAt: profile.createdAt,
      updatedAt: profile.updatedAt,
      protocol: profile.useTLS ? 'HTTPS' : 'HTTP',
      connections: 0
    }))
    
    console.log('🔄 Setting listeners.value to:', mappedListeners)
    listeners.value = mappedListeners
    
    if (listeners.value.length === 0) {
      console.log('⚠️ No listener profiles found in response')
      ElMessage.warning('No listener profiles found.')
    } else {
      console.log('✅ Loaded listeners:', listeners.value)
    }
  } catch (error) {
    console.error('Failed to load profiles:', error)
    ElMessage.error('Failed to load profiles: Network error or server unavailable')
    listeners.value = []
  }
}

// Listen for profile creation events
const handleProfileCreated = () => {
  console.log('🎉 Profile created event detected, reloading profiles...')
  loadProfiles()
}

onMounted(() => {
  console.log('🚀 Dashboard component mounted - starting initialization...')
  console.log('📊 Initial listeners value:', listeners.value)
  console.log('📊 Initial listeners length:', listeners.value.length)
  
  // Load profiles
  loadProfiles()
  console.log('📡 loadProfiles() called')
  
  // Register event listener
  window.addEventListener('profileCreated', handleProfileCreated)
  console.log('📡 Dashboard registered profileCreated event listener')
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

h1 {
  color: #409eff;
  text-align: center;
  margin-bottom: 20px;
}

.listeners-panel {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: #909399;
}

.empty-state h3 {
  margin: 20px 0 10px 0;
  color: #606266;
}

.empty-state p {
  margin: 5px 0;
  font-size: 14px;
}
</style>
