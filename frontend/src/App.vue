<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useAuth } from '@/services/auth'
import { useRouter } from 'vue-router'
import LoginRegister from '@/components/LoginRegister.vue'
import MainLayout from '@/layout/Layout.vue'
import ProfileSelection from '@/views/ProfileSelection.vue'

const { isAuthenticated, checkBackendHealth } = useAuth()
const router = useRouter()

// Reactive profile selection state
const hasSelectedProfile = ref(localStorage.getItem('active_profile_id') !== null)
const backendUnavailable = ref(false)

// Update profile selection state
const updateProfileSelection = () => {
  hasSelectedProfile.value = localStorage.getItem('active_profile_id') !== null
}

// Check backend health
const checkBackendStatus = async () => {
  try {
    const isHealthy = await checkBackendHealth()
    backendUnavailable.value = !isHealthy
  } catch (error) {
    backendUnavailable.value = true
  }
}

// Single page logic - show auth forms, profile selection, or main app based on state
const showAuth = computed(() => !isAuthenticated.value && !backendUnavailable.value)
const showProfileSelection = computed(() => isAuthenticated.value && !hasSelectedProfile.value && !backendUnavailable.value)
const showRestartMessage = computed(() => backendUnavailable.value)

// Watch for authentication changes and update profile selection
watch(isAuthenticated, () => {
  updateProfileSelection()
})

// Add event listener for storage changes
onMounted(async () => {
  window.addEventListener('storage', updateProfileSelection)
  window.addEventListener('profileSelected', updateProfileSelection)
  
  // Also check on mount
  updateProfileSelection()
  await checkBackendStatus()
  
  // Set up periodic health check
  setInterval(checkBackendStatus, 10000) // Check every 10 seconds
})
</script>

<template>
  <div id="app">
    <!-- Single Page Application -->
    <div v-if="showRestartMessage" class="restart-container">
      <div class="restart-message">
        <div class="restart-icon">🔄</div>
        <h1>Server Unavailable</h1>
        <p>The backend server is not responding. This usually means the server has been stopped or crashed.</p>
        <div class="restart-instructions">
          <h3>To fix this:</h3>
          <ol>
            <li>Kill the both servers</li>
            <li>Generate new TLS certificates if not already generated (using <code>generate-certs.ps1</code>if on windows) </li>
            <li>Run <code>run-mulic2.bat</code> again to restart the server</li>
            <li>Wait for the server to fully start</li>
            <li>Refresh this page</li>
          </ol>
        </div>
      </div>
    </div>
    
    <div v-else-if="showAuth" class="auth-container">
      <LoginRegister />
    </div>
    
    <div v-else-if="showProfileSelection" class="profile-selection-container">
      <ProfileSelection />
    </div>
    
    <div v-else class="app-container">
      <MainLayout />
    </div>
  </div>
</template>

<style>
#app {
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', '微软雅黑', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  height: 100vh;
  margin: 0;
  padding: 0;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  padding: 0;
  background-color: #000;
}

html {
  height: 100%;
}

.auth-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
}

.app-container {
  height: 100vh;
}

.restart-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
  padding: 20px;
}

.restart-message {
  background: #1a1a1a;
  border: 2px solid #f56c6c;
  border-radius: 8px;
  padding: 40px;
  max-width: 600px;
  width: 100%;
  text-align: center;
  color: #fff;
  font-family: 'Courier New', monospace;
}

.restart-icon {
  font-size: 64px;
  margin-bottom: 20px;
}

.restart-message h1 {
  color: #f56c6c;
  margin: 0 0 20px 0;
  font-size: 28px;
}

.restart-message p {
  color: #ccc;
  margin: 0 0 30px 0;
  font-size: 16px;
  line-height: 1.5;
}

.restart-instructions {
  text-align: left;
  margin-bottom: 30px;
}

.restart-instructions h3 {
  color: #f56c6c;
  margin: 0 0 15px 0;
  font-size: 18px;
}

.restart-instructions ol {
  color: #ccc;
  margin: 0;
  padding-left: 20px;
  line-height: 1.8;
}

.restart-instructions li {
  margin-bottom: 8px;
}

.restart-instructions code {
  background: #333;
  color: #f56c6c;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
}

</style>
