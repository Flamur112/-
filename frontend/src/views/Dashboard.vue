<template>
  <div class="dashboard">
    <h1>🎯 Dashboard</h1>
    
    <!-- Listeners Panel -->
    <div class="listeners-panel">
      <div class="panel-header">
        <h3>Listener Profiles</h3>
        <div style="display: flex; gap: 10px;">
          <el-button type="warning" @click="cleanupDuplicateProfiles" :loading="cleaning">
            <el-icon><Delete /></el-icon>
            Cleanup Duplicates
          </el-button>
          <el-button type="info" @click="loadProfiles">
            <el-icon><Refresh /></el-icon>
            Refresh Profiles
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
      
      <!-- Empty State -->
      <div v-if="listeners.length === 0" class="empty-state">
        <el-icon size="64" color="#909399"><Setting /></el-icon>
        <h3>No Listener Profiles</h3>
        <p>Create a listener profile to start accepting connections.</p>
      </div>
      
      <!-- Listeners Table -->
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
        <el-table-column label="Actions" width="120">
          <template #default="scope">
            <el-button size="small" type="danger" @click="deleteProfile(scope.row.id)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Refresh, Setting } from '@element-plus/icons-vue'

// Reactive data
const listeners = ref<any[]>([])
const cleaning = ref(false)

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

// Cleanup duplicate profiles
const cleanupDuplicateProfiles = async () => {
  try {
    cleaning.value = true
    
    // Find duplicates (same name, port, and host)
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
      // Delete duplicates (keep the most recent)
      for (const duplicate of duplicates) {
        await deleteProfile(duplicate.id)
      }
      
      ElMessage.success(`Cleaned up ${duplicates.length} duplicate profiles.`)
      await loadProfiles() // Reload the list
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
  
  // Sort by creation date (newest first)
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
        await loadProfiles() // Reload the list
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
  // Load profiles
  loadProfiles()
  
  // Register event listener
  window.addEventListener('profileCreated', handleProfileCreated)
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
</style>
