<template>
  <div class="dashboard">
    <h1>🎯 Dashboard</h1>
    
    <!-- Main Dashboard Tabs -->
    <el-tabs v-model="activeTab" type="card" class="dashboard-tabs">
      
      <!-- Agents Tab -->
      <el-tab-pane label="Agents" name="agents">
        <div class="tab-content">
          <div class="section-header">
            <h3>🤖 Connected Agents</h3>
            <el-button type="primary" @click="refreshAgents">
              <el-icon><Refresh /></el-icon>
              Refresh
            </el-button>
          </div>
          
          <div v-if="agents.length === 0" class="empty-state">
            <el-icon size="64" color="#909399"><Monitor /></el-icon>
            <h3>No Agents Connected</h3>
            <p>Deploy agents to see them here.</p>
          </div>
          
          <el-table v-else :data="agents" style="width: 100%">
            <el-table-column prop="id" label="Agent ID" width="120" />
            <el-table-column prop="hostname" label="Hostname" width="150" />
            <el-table-column prop="ip" label="IP Address" width="120" />
            <el-table-column prop="os" label="OS" width="100" />
            <el-table-column prop="status" label="Status" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.status === 'online' ? 'success' : 'danger'">
                  {{ scope.row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="lastSeen" label="Last Seen" width="150" />
            <el-table-column label="Actions" width="200">
              <template #default="scope">
                <el-button size="small" @click="viewAgent(scope.row)">
                  <el-icon><View /></el-icon>
                  View
                </el-button>
                <el-button size="small" type="danger" @click="disconnectAgent(scope.row.id)">
                  <el-icon><Close /></el-icon>
                  Disconnect
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
      
      <!-- Listeners Tab -->
      <el-tab-pane label="Listeners" name="listeners">
        <div class="tab-content">
          <div class="section-header">
            <h3>📡 Listener Profiles</h3>
            <div style="display: flex; gap: 10px;">
              <el-button type="warning" @click="cleanupDuplicateProfiles" :loading="cleaning">
                <el-icon><Delete /></el-icon>
                Cleanup Duplicates
              </el-button>
              <el-button type="primary" @click="createListener">
                <el-icon><Plus /></el-icon>
                Create Listener
              </el-button>
              <el-button type="info" @click="loadProfiles">
                <el-icon><Refresh /></el-icon>
                Refresh
              </el-button>
            </div>
          </div>
          
          <!-- Stats -->
          <div class="stats-grid">
            <div class="stat-card">
              <h4>Total Profiles</h4>
              <span class="stat-number">{{ listeners.length }}</span>
            </div>
            <div class="stat-card">
              <h4>Active Profiles</h4>
              <span class="stat-number">{{ activeProfilesCount }}</span>
            </div>
            <div class="stat-card">
              <h4>TLS Enabled</h4>
              <span class="stat-number">{{ tlsProfilesCount }}</span>
            </div>
            <div class="stat-card">
              <h4>Unique Ports</h4>
              <span class="stat-number">{{ uniquePortsCount }}</span>
            </div>
          </div>
          
          <div v-if="listeners.length === 0" class="empty-state">
            <el-icon size="64" color="#909399"><Setting /></el-icon>
            <h3>No Listener Profiles</h3>
            <p>Create a listener profile to start accepting connections.</p>
          </div>
          
          <el-table v-else :data="listeners" style="width: 100%" class="listeners-table">
            <el-table-column prop="name" label="Profile Name" width="150" />
            <el-table-column prop="protocol" label="Protocol" width="100" />
            <el-table-column prop="host" label="Host" width="120" />
            <el-table-column prop="port" label="Port" width="80" />
            <el-table-column prop="isActive" label="Status" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.isActive ? 'success' : 'info'">
                  {{ scope.row.isActive ? 'Active' : 'Inactive' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="connections" label="Connections" width="100" />
            <el-table-column prop="createdAt" label="Created" width="150">
              <template #default="scope">
                {{ formatDate(scope.row.createdAt) }}
              </template>
            </el-table-column>
            <el-table-column label="Actions" width="200">
              <template #default="scope">
                <el-button size="small" @click="editListener(scope.row)">
                  <el-icon><Edit /></el-icon>
                  Edit
                </el-button>
                <el-button size="small" type="danger" @click="deleteProfile(scope.row.id)">
                  <el-icon><Delete /></el-icon>
                  Delete
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
      
      <!-- VNC Tab -->
      <el-tab-pane label="VNC" name="vnc">
        <div class="tab-content">
          <div class="section-header">
            <h3>🖥️ VNC Screen Capture</h3>
            <el-button type="primary" @click="startVNCCapture">
              <el-icon><VideoPlay /></el-icon>
              Start Capture
            </el-button>
          </div>
          
          <div class="vnc-container">
            <div v-if="!vncActive" class="vnc-placeholder">
              <el-icon size="64" color="#909399"><Monitor /></el-icon>
              <h3>VNC Not Active</h3>
              <p>Click "Start Capture" to begin VNC screen capture.</p>
            </div>
            
            <div v-else class="vnc-viewer">
              <div class="vnc-controls">
                <el-button @click="stopVNCCapture" type="danger">
                  <el-icon><VideoPause /></el-icon>
                  Stop Capture
                </el-button>
                <el-button @click="refreshVNC" type="info">
                  <el-icon><Refresh /></el-icon>
                  Refresh
                </el-button>
              </div>
              <div class="vnc-screen">
                <img v-if="vncImage" :src="vncImage" alt="VNC Screen" />
                <div v-else class="vnc-loading">
                  <el-icon class="is-loading"><Loading /></el-icon>
                  <p>Capturing screen...</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>
      
      <!-- Tasks Tab -->
      <el-tab-pane label="Tasks" name="tasks">
        <div class="tab-content">
          <div class="section-header">
            <h3>📋 Task Management</h3>
            <el-button type="primary" @click="createTask">
              <el-icon><Plus /></el-icon>
              Create Task
            </el-button>
          </div>
          
          <div v-if="tasks.length === 0" class="empty-state">
            <el-icon size="64" color="#909399"><Document /></el-icon>
            <h3>No Tasks</h3>
            <p>Create tasks to manage agent operations.</p>
          </div>
          
          <el-table v-else :data="tasks" style="width: 100%">
            <el-table-column prop="id" label="Task ID" width="120" />
            <el-table-column prop="agentId" label="Agent" width="120" />
            <el-table-column prop="command" label="Command" width="200" />
            <el-table-column prop="status" label="Status" width="100">
              <template #default="scope">
                <el-tag :type="getTaskStatusType(scope.row.status)">
                  {{ scope.row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="createdAt" label="Created" width="150" />
            <el-table-column prop="result" label="Result" width="200" />
            <el-table-column label="Actions" width="120">
              <template #default="scope">
                <el-button size="small" @click="viewTaskResult(scope.row)">
                  <el-icon><View /></el-icon>
                  View
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
      
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Delete, Refresh, Setting, Plus, Edit, View, Close, 
  Monitor, VideoPlay, VideoPause, Document, Loading 
} from '@element-plus/icons-vue'

// Reactive data
const activeTab = ref('agents')
const listeners = ref<any[]>([])
const agents = ref<any[]>([])
const tasks = ref<any[]>([])
const cleaning = ref(false)
const vncActive = ref(false)
const vncImage = ref('')

// Computed properties
const activeProfilesCount = computed(() => 
  listeners.value.filter(p => p.isActive).length
)

const tlsProfilesCount = computed(() => 
  listeners.value.filter(p => p.useTLS).length
)

const uniquePortsCount = computed(() => 
  new Set(listeners.value.map(p => p.port)).size
)

// Load profiles function
const loadProfiles = async () => {
  try {
    const response = await fetch('/api/profile/list')
    
    if (!response.ok) {
      ElMessage.error(`Failed to load profiles: Server returned ${response.status} ${response.statusText}`)
      listeners.value = []
      return
    }
    
    const data = await response.json()
    const profiles = data.profiles || []
    
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
    
    listeners.value = mappedListeners
    
    if (listeners.value.length === 0) {
      ElMessage.warning('No listener profiles found.')
    }
  } catch (error) {
    console.error('Failed to load profiles:', error)
    ElMessage.error('Failed to load profiles: Network error or server unavailable')
    listeners.value = []
  }
}

// Load agents
const refreshAgents = async () => {
  try {
    const response = await fetch('/api/agents')
    if (response.ok) {
      const data = await response.json()
      agents.value = data.agents || []
    } else {
      agents.value = []
    }
  } catch (error) {
    console.error('Failed to load agents:', error)
    agents.value = []
  }
}

// Load tasks
const loadTasks = async () => {
  try {
    const response = await fetch('/api/tasks')
    if (response.ok) {
      const data = await response.json()
      tasks.value = data.tasks || []
    } else {
      tasks.value = []
    }
  } catch (error) {
    console.error('Failed to load tasks:', error)
    tasks.value = []
  }
}

// Cleanup duplicate profiles
const cleanupDuplicateProfiles = async () => {
  try {
    cleaning.value = true
    
    const duplicates = findDuplicateProfiles()
    
    if (duplicates.length === 0) {
      ElMessage.info('No duplicate profiles found to clean up.')
      return
    }
    
    const result = await ElMessageBox.confirm(
      `Found ${duplicates.length} duplicate profiles. This will keep only the most recent version of each unique profile. Continue?`,
      'Cleanup Duplicates',
      {
        confirmButtonText: 'Yes, Cleanup',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )
    
    if (result === 'confirm') {
      for (const duplicate of duplicates) {
        await deleteProfile(duplicate.id)
      }
      
      ElMessage.success(`Cleaned up ${duplicates.length} duplicate profiles.`)
      await loadProfiles()
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Cleanup failed:', error)
      ElMessage.error('Failed to cleanup duplicate profiles.')
    }
  } finally {
    cleaning.value = false
  }
}

// Find duplicate profiles
const findDuplicateProfiles = () => {
  const seen = new Map<string, any>()
  const duplicates: any[] = []
  
  const sorted = [...listeners.value].sort((a, b) => 
    new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  )
  
  for (const profile of sorted) {
    const key = `${profile.name}-${profile.port}-${profile.host}`
    
    if (seen.has(key)) {
      duplicates.push(profile)
    } else {
      seen.set(key, profile)
    }
  }
  
  return duplicates
}

// Delete a profile
const deleteProfile = async (profileId: string) => {
  try {
    const result = await ElMessageBox.confirm(
      'Are you sure you want to delete this profile?',
      'Delete Profile',
      {
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )
    
    if (result === 'confirm') {
      const response = await fetch(`/api/profile/delete/${profileId}`, {
        method: 'DELETE'
      })
      
      if (response.ok) {
        ElMessage.success('Profile deleted successfully.')
        await loadProfiles()
      } else {
        ElMessage.error('Failed to delete profile.')
      }
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Delete failed:', error)
      ElMessage.error('Failed to delete profile.')
    }
  }
}

// VNC functions
const startVNCCapture = async () => {
  try {
    const response = await fetch('/api/vnc/start', { method: 'POST' })
    if (response.ok) {
      vncActive.value = true
      ElMessage.success('VNC capture started.')
      refreshVNC()
    } else {
      ElMessage.error('Failed to start VNC capture.')
    }
  } catch (error) {
    console.error('VNC start failed:', error)
    ElMessage.error('Failed to start VNC capture.')
  }
}

const stopVNCCapture = async () => {
  try {
    const response = await fetch('/api/vnc/stop', { method: 'POST' })
    if (response.ok) {
      vncActive.value = false
      vncImage.value = ''
      ElMessage.success('VNC capture stopped.')
    } else {
      ElMessage.error('Failed to stop VNC capture.')
    }
  } catch (error) {
    console.error('VNC stop failed:', error)
    ElMessage.error('Failed to stop VNC capture.')
  }
}

const refreshVNC = async () => {
  if (!vncActive.value) return
  
  try {
    const response = await fetch('/api/vnc/screenshot')
    if (response.ok) {
      const blob = await response.blob()
      vncImage.value = URL.createObjectURL(blob)
    }
  } catch (error) {
    console.error('VNC refresh failed:', error)
  }
}

// Agent functions
const viewAgent = (agent: any) => {
  ElMessage.info(`Viewing agent: ${agent.id}`)
  // TODO: Implement agent detail view
}

const disconnectAgent = async (agentId: string) => {
  try {
    const result = await ElMessageBox.confirm(
      'Are you sure you want to disconnect this agent?',
      'Disconnect Agent',
      {
        confirmButtonText: 'Disconnect',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )
    
    if (result === 'confirm') {
      const response = await fetch(`/api/agents/${agentId}/disconnect`, { method: 'POST' })
      if (response.ok) {
        ElMessage.success('Agent disconnected.')
        refreshAgents()
      } else {
        ElMessage.error('Failed to disconnect agent.')
      }
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Disconnect failed:', error)
      ElMessage.error('Failed to disconnect agent.')
    }
  }
}

// Task functions
const createTask = () => {
  ElMessage.info('Create task functionality coming soon...')
}

const viewTaskResult = (task: any) => {
  ElMessage.info(`Viewing task result: ${task.id}`)
  // TODO: Implement task result view
}

const getTaskStatusType = (status: string) => {
  switch (status) {
    case 'completed': return 'success'
    case 'running': return 'warning'
    case 'failed': return 'danger'
    default: return 'info'
  }
}

// Listener functions
const createListener = () => {
  ElMessage.info('Create listener functionality coming soon...')
}

const editListener = (listener: any) => {
  ElMessage.info(`Editing listener: ${listener.name}`)
  // TODO: Implement listener edit
}

// Format date
const formatDate = (dateString: string) => {
  try {
    return new Date(dateString).toLocaleDateString()
  } catch {
    return dateString
  }
}

// Listen for profile creation events
const handleProfileCreated = () => {
  loadProfiles()
}

onMounted(() => {
  // Load initial data
  loadProfiles()
  refreshAgents()
  loadTasks()
  
  // Register event listener
  window.addEventListener('profileCreated', handleProfileCreated)
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

h1 {
  color: #409eff;
  text-align: center;
  margin-bottom: 20px;
}

.dashboard-tabs {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.tab-content {
  padding: 20px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h3 {
  margin: 0;
  color: #303133;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 15px;
  margin-bottom: 20px;
}

.stat-card {
  background: #f8f9fa;
  padding: 15px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #dee2e6;
}

.stat-card h4 {
  margin: 0 0 10px 0;
  color: #6c757d;
  font-size: 14px;
}

.stat-number {
  font-size: 24px;
  font-weight: bold;
  color: #409eff;
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

.listeners-table {
  margin-top: 20px;
}

.vnc-container {
  min-height: 400px;
}

.vnc-placeholder {
  text-align: center;
  padding: 60px;
  color: #909399;
}

.vnc-viewer {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  overflow: hidden;
}

.vnc-controls {
  padding: 15px;
  background: #f8f9fa;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  gap: 10px;
}

.vnc-screen {
  padding: 20px;
  text-align: center;
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.vnc-screen img {
  max-width: 100%;
  max-height: 400px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
}

.vnc-loading {
  color: #909399;
}

.vnc-loading .el-icon {
  font-size: 32px;
  margin-bottom: 10px;
}
</style>

