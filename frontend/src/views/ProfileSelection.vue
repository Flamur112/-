<template>
  <div class="profile-selection">
    <div class="selection-container">
      <!-- Header -->
      <div class="header">
        <h1 class="title">Select Profile</h1>
        <p class="subtitle">Choose or create a server profile to continue</p>
      </div>

      <!-- Profile List -->
      <div class="profiles-section" v-if="profiles.length > 0">
        <h2 class="section-title">Available Profiles</h2>
        <div class="profiles-grid">
          <div 
            v-for="profile in profiles" 
            :key="profile.id"
            class="profile-item"
            :class="{ selected: selectedProfileId === profile.id }"
            @click="selectProfile(profile.id)"
          >
            <div class="profile-info">
              <h3 class="profile-name">{{ profile.name }}</h3>
              <p class="profile-details">{{ profile.host }}:{{ profile.port }}</p>
              <p class="profile-project">{{ profile.projectName }}</p>
            </div>
            <div class="profile-actions">
              <el-button 
                v-if="profile.id === selectedProfileId"
                size="small" 
                type="success" 
                icon="Check"
                disabled
              >
                Active
              </el-button>
              <el-button 
                v-else
                size="small" 
                type="primary" 
                icon="Check"
                @click.stop="selectProfile(profile.id)"
              >
                Select
              </el-button>
              <el-button 
                size="small" 
                type="danger" 
                icon="Delete"
                @click.stop="deleteProfile(profile.id)"
              >
                Delete
              </el-button>
            </div>
          </div>
        </div>
      </div>

      <!-- No Profiles State -->
      <div class="no-profiles" v-if="profiles.length === 0">
        <el-icon size="64" color="#909399"><Setting /></el-icon>
        <h2>No Profiles Available</h2>
        <p>You need to create your first profile to get started</p>
      </div>

      <!-- Action Buttons -->
      <div class="actions">
        <el-button 
          type="primary" 
          size="large" 
          icon="Plus"
          @click="showCreateProfileDialog"
        >
          Create New Profile
        </el-button>
        
        <el-button 
          v-if="profiles.length > 0 && selectedProfileId"
          type="success" 
          size="large" 
          icon="Check"
          @click="continueWithProfile"
        >
          Continue with Selected Profile
        </el-button>

        <!-- Reconnect Button -->
        <el-button 
          v-if="backendUnavailable"
          type="warning" 
          size="large" 
          icon="Refresh"
          @click="checkBackendHealth"
        >
          Reconnect to Server
        </el-button>
      </div>

      <!-- Create Profile Dialog -->
      <el-dialog
        v-model="profileDialogVisible"
        title="Create New Profile"
        width="500px"
        @close="resetProfileForm"
      >
        <el-form
          ref="profileFormRef"
          :model="profileForm"
          :rules="profileRules"
          label-width="120px"
        >
          <el-form-item label="Profile Name" prop="name">
            <el-input 
              v-model="profileForm.name" 
              placeholder="Enter profile name"
            />
          </el-form-item>
          
          <el-form-item label="Project Name" prop="projectName">
            <el-input 
              v-model="profileForm.projectName" 
              placeholder="Enter project name"
            />
          </el-form-item>
          
                   <el-form-item label="Host" prop="host">
           <el-input 
             v-model="profileForm.host" 
             placeholder="192.168.1.100"
           />
         </el-form-item>
         
         <el-form-item label="Port" prop="port">
           <el-input 
             v-model="profileForm.port" 
             placeholder="8081"
             style="width: 100%"
           />
         </el-form-item>
         

          
          <el-form-item label="Description" prop="description">
            <el-input
              v-model="profileForm.description"
              type="textarea"
              :rows="3"
              placeholder="Enter description (optional)"
            />
          </el-form-item>
        </el-form>
        
        <template #footer>
          <div style="text-align: right;">
            <el-button @click="profileDialogVisible = false">Cancel</el-button>
            <el-button 
              type="primary" 
              @click="handleSaveProfile"
              :loading="savingProfile"
            >
              Create Profile
            </el-button>
          </div>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { listenerService, type Profile } from '@/services/listener'
import { initializePorts, getPorts, isPortConflicting } from '@/config/ports'

const router = useRouter()

// Reactive data
const profiles = ref<any[]>([])
const selectedProfileId = ref<string | null>(null)
const profileDialogVisible = ref(false)
const savingProfile = ref(false)
const backendUnavailable = ref(false)

// Form refs
const profileFormRef = ref<FormInstance>()

// Profile form data
const profileForm = reactive({
  name: '',
  projectName: '',
  host: '',
  port: '', // Will be set after ports are initialized
  description: ''
})

// Get default port from configuration
const getDefaultPort = () => {
  try {
    const ports = getPorts()
    return ports.C2_DEFAULT.toString()
  } catch (error) {
    console.error('Ports not initialized:', error)
    return '8081' // Fallback only for UI display, not for actual binding
  }
}



// Form validation rules
const profileRules: FormRules = {
  name: [
    { required: true, message: 'Please enter profile name', trigger: 'blur' }
  ],
  projectName: [
    { required: true, message: 'Please enter project name', trigger: 'blur' }
  ],
  host: [
    { required: true, message: 'Please enter host', trigger: 'blur' },
    { 
      validator: (rule: any, value: string, callback: any) => {
        if (!value) {
          callback(new Error('Host is required'))
          return
        }
        
        // Validate IP address format
        const ipRegex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
        const localhostRegex = /^localhost$/
        const domainRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$/
        
        if (!ipRegex.test(value) && !localhostRegex.test(value) && !domainRegex.test(value)) {
          callback(new Error('Please enter a valid IP address (e.g., 192.168.1.100), localhost, or domain name'))
          return
        }
        
        callback()
      }, 
      trigger: 'blur' 
    }
  ],
  port: [
    { required: true, message: 'Please enter port', trigger: 'blur' },
    { 
      validator: (rule: any, value: string, callback: any) => {
        try {
          const portNum = parseInt(value)
          if (isNaN(portNum)) {
            callback(new Error('Port must be a valid number'))
            return
          }
          if (isPortConflicting(portNum)) {
            const ports = getPorts()
            callback(new Error(`Port ${ports.BACKEND_API} is reserved for the backend API server. Please use a different port.`))
          } else if (portNum < 1 || portNum > 65535) {
            callback(new Error('Port must be between 1 and 65535'))
          } else {
            callback()
          }
        } catch (error) {
          callback(new Error('Port validation failed: configuration not loaded'))
        }
      }, 
      trigger: 'blur' 
    }
  ]
}

// Load profiles from localStorage
const loadProfiles = () => {
  const savedProfiles = localStorage.getItem('user_profiles')
  if (savedProfiles) {
    profiles.value = JSON.parse(savedProfiles)
  }
}



// Save profiles to localStorage
const saveProfiles = () => {
  localStorage.setItem('user_profiles', JSON.stringify(profiles.value))
}

// Methods
const selectProfile = (profileId: string) => {
  selectedProfileId.value = profileId
}

const checkBackendHealth = async () => {
  try {
    backendUnavailable.value = false
    // Try to get listener status to check if backend is available
    await listenerService.getStatus()
    ElMessage.success('Successfully reconnected to server!')
  } catch (error) {
    backendUnavailable.value = true
    ElMessage.error('Failed to reconnect to server. Please check if the backend is running.')
  }
}

// Check backend health on mount
const checkBackendHealthOnMount = async () => {
  try {
    await listenerService.getStatus()
    backendUnavailable.value = false
  } catch (error) {
    backendUnavailable.value = true
    // If backend is unavailable, clear any stored auth state to prevent conflicts
    localStorage.removeItem('auth_token')
    localStorage.removeItem('user_data')
    localStorage.removeItem('active_profile_id')
    
    // Also clear any active profile selection to prevent conflicts
    selectedProfileId.value = null
  }
}

const continueWithProfile = async () => {
  if (selectedProfileId.value) {
    // Check if backend is available before proceeding
    if (backendUnavailable.value) {
      ElMessage.error('Backend server is not available. Please restart the server using run-mulic2.bat')
      return
    }

    try {
      // Find the selected profile
      const selectedProfile = profiles.value.find(p => p.id === selectedProfileId.value)
      if (!selectedProfile) {
        ElMessage.error('Selected profile not found')
        return
      }

      // Start the C2 listener with the selected profile
      await listenerService.startListener(selectedProfile)
      
      // Save selected profile to localStorage for the session
      localStorage.setItem('active_profile_id', selectedProfileId.value)
      ElMessage.success(`C2 Listener started on ${selectedProfile.host}:${selectedProfile.port}`)
      
      // Trigger a custom event to notify App.vue
      window.dispatchEvent(new CustomEvent('profileSelected'))
      
      // Navigate to dashboard
      router.push('/')
    } catch (error) {
      console.error('Failed to start listener:', error)
      
      // Check if it's a backend connection error
      if (error instanceof Error && error.message.includes('ECONNREFUSED')) {
        ElMessage.error('Backend server is not running. Please start the backend first.')
      } else {
        ElMessage.error('Failed to start C2 listener: ' + (error as Error).message)
      }
    }
  }
}

const showCreateProfileDialog = () => {
  // Check if backend is available before allowing profile creation
  if (backendUnavailable.value) {
    ElMessage.error('Backend server is not available. Please restart the server using run-mulic2.bat')
    return
  }
  
  profileDialogVisible.value = true
}

// Reset profile form
const resetProfileForm = () => {
  profileForm.name = ''
  profileForm.projectName = ''
  profileForm.host = ''
  profileForm.port = getDefaultPort()
  profileForm.description = ''
  profileFormRef.value?.clearValidate()
}

const handleSaveProfile = async () => {
  if (!profileFormRef.value) return

  try {
    await profileFormRef.value.validate()
    savingProfile.value = true

    // Create new profile
    const newProfile = {
      id: `profile-${Date.now()}`,
      name: profileForm.name,
      projectName: profileForm.projectName,
      host: profileForm.host,
      port: parseInt(profileForm.port),
      description: profileForm.description
    }
    profiles.value.push(newProfile)
    ElMessage.success('Profile created successfully')
    
    // Auto-select the newly created profile
    selectedProfileId.value = newProfile.id

    saveProfiles()
    profileDialogVisible.value = false
  } catch (error) {
    console.error('Save profile error:', error)
  } finally {
    savingProfile.value = false
  }
}

const deleteProfile = async (profileId: string) => {
  const profile = profiles.value.find(p => p.id === profileId)
  if (!profile) return

  // Check if backend is available before proceeding
  if (backendUnavailable.value) {
    ElMessage.error('Backend server is not available. Please restart the server using run-mulic2.bat')
    return
  }

  try {
    await ElMessageBox.confirm(
      `This will permanently delete the profile "${profile.name}" (${profile.host}:${profile.port}). ` +
      'All agents connected to this profile will be disconnected and their connections will be lost. ' +
      'This action cannot be undone.',
      'Delete Profile & Disconnect All Agents',
      {
        confirmButtonText: 'Delete Profile',
        cancelButtonText: 'Cancel',
        type: 'error',
        confirmButtonClass: 'el-button--danger',
      }
    )

    // Remove profile from list
    profiles.value = profiles.value.filter(p => p.id !== profileId)
    
    // Clear selection if deleted profile was selected
    if (selectedProfileId.value === profileId) {
      selectedProfileId.value = null
    }
    
    // Clear active profile if this was the active one
    const activeProfileId = localStorage.getItem('active_profile_id')
    if (activeProfileId === profileId) {
      localStorage.removeItem('active_profile_id')
    }

    saveProfiles()
    ElMessage.success('Profile deleted successfully')
  } catch (error) {
    // User cancelled deletion
    console.log('Profile deletion cancelled')
  }
}

// Initialize ports on component mount
onMounted(async () => {
  try {
    await initializePorts()
    loadProfiles()
    await checkBackendHealthOnMount()
  } catch (error) {
    console.error('Failed to initialize ports:', error)
    ElMessage.error('Failed to load port configuration. Please check your config.json file.')
  }
})
</script>

<style scoped lang="scss">
.profile-selection {
  min-height: 100vh;
  background: var(--primary-black);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  font-family: 'Courier New', monospace;

  .selection-container {
    background: var(--secondary-black);
    border: 1px solid var(--border-color);
    border-radius: 0;
    padding: 40px;
    max-width: 800px;
    width: 100%;
    color: var(--text-white);

    .header {
      text-align: center;
      margin-bottom: 40px;

      .title {
        font-size: 32px;
        font-weight: 600;
        color: var(--text-white);
        margin: 0 0 12px 0;
      }

      .subtitle {
        font-size: 16px;
        color: var(--text-gray);
        margin: 0;
      }
    }

    .profiles-section {
      margin-bottom: 40px;

      .section-title {
        font-size: 20px;
        color: var(--text-white);
        margin: 0 0 20px 0;
        font-weight: 600;
      }

      .profiles-grid {
        display: grid;
        gap: 16px;

        .profile-item {
          background: var(--dark-gray);
          border: 2px solid var(--border-color);
          border-radius: 0;
          padding: 20px;
          cursor: pointer;
          transition: all 0.3s ease;
          display: flex;
          justify-content: space-between;
          align-items: center;

          &:hover {
            border-color: var(--accent-red);
          }

          &.selected {
            border-color: var(--accent-red);
            background: var(--dark-gray);
          }

          .profile-info {
            .profile-name {
              font-size: 18px;
              font-weight: 600;
              color: var(--text-white);
              margin: 0 0 8px 0;
            }

            .profile-details {
              font-size: 14px;
              color: var(--text-gray);
              margin: 0 0 4px 0;
            }

            .profile-project {
              font-size: 12px;
              color: var(--text-gray);
              margin: 0;
            }
          }

          .profile-actions {
            display: flex;
            gap: 8px;
            flex-wrap: wrap;
          }
        }
      }
    }

    .no-profiles {
      text-align: center;
      padding: 60px 20px;
      margin-bottom: 40px;

      h2 {
        color: var(--text-white);
        margin: 20px 0 12px 0;
        font-size: 24px;
      }

      p {
        color: var(--text-gray);
        margin: 0;
        font-size: 16px;
      }
    }

    .actions {
      display: flex;
      justify-content: center;
      gap: 16px;
      flex-wrap: wrap;
    }
  }
}



// Element Plus overrides
:deep(.el-input__wrapper) {
  background: var(--dark-gray) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-input__inner) {
  color: var(--text-white) !important;
  
  &::placeholder {
    color: var(--text-gray) !important;
  }
}

:deep(.el-form-item__label) {
  color: var(--text-gray) !important;
}

:deep(.el-dialog) {
  background: var(--secondary-black) !important;
  border: 1px solid var(--border-color) !important;
  border-radius: 0 !important;
}

:deep(.el-dialog__header) {
  background: var(--dark-gray) !important;
  border-bottom: 1px solid var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-dialog__title) {
  color: var(--text-white) !important;
}

:deep(.el-dialog__body) {
  background: var(--secondary-black) !important;
  color: var(--text-white) !important;
}

:deep(.el-dialog__footer) {
  background: var(--dark-gray) !important;
  border-top: 1px solid var(--border-color) !important;
}
</style>
