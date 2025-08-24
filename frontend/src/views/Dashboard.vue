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
            <el-table-column prop="status" label="Status" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.status === 'active' ? 'success' : 'info'">
                  {{ scope.row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="connections" label="Connections" width="100" />
            <el-table-column prop="createdAt" label="Created" width="180" />
            <el-table-column label="Actions" width="200">
              <template #default="scope">
                <el-button size="small" @click="startListener(scope.row)" v-if="scope.row.status !== 'active'">
                  <el-icon><CaretRight /></el-icon>
                  Start
                </el-button>
                <el-button size="small" type="warning" @click="stopListener(scope.row)" v-if="scope.row.status === 'active'">
                  <el-icon><CaretLeft /></el-icon>
                  Stop
                </el-button>
                <el-button size="small" type="danger" @click="deleteListener(scope.row)" v-if="scope.row.status !== 'active'">
                  <el-icon><Delete /></el-icon>
                  Delete
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Payloads Panel -->
      <el-tab-pane label="Payloads" name="payloads">
        <div class="payloads-panel">
          <div class="panel-header">
            <h3>Payload Generation & Management</h3>
            <el-button type="primary" @click="showPayloadDialog">
              <el-icon><Plus /></el-icon>
              Generate New Payload
            </el-button>
          </div>
          
          <!-- Payload Generation Form -->
          <div class="payload-generator">
            <el-form :model="payloadForm" label-width="150px" class="payload-form">
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Target IP:">
                    <el-input v-model="payloadForm.targetIP" placeholder="192.168.1.100" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Target Port:">
                    <el-input v-model="payloadForm.targetPort" placeholder="4444" />
                  </el-form-item>
                </el-col>
              </el-row>
              
              <el-row :gutter="20">
                <el-col :span="12">
                  <el-form-item label="Payload Type:">
                    <el-select v-model="payloadForm.type" placeholder="Select payload type">
                      <el-option label="PowerShell (.ps1)" value="ps1" />
                      <el-option label="Executable (.exe)" value="exe" />
                      <el-option label="Binary (.bin)" value="bin" />
                      <el-option label="Python (.py)" value="py" />
                      <el-option label="Batch (.bat)" value="bat" />
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="Output Format:">
                    <el-select v-model="payloadForm.outputFormat" placeholder="Select output format">
                      <el-option label="Raw Code" value="raw" />
                      <el-option label="Base64 Encoded" value="base64" />
                      <el-option label="Compressed" value="compressed" />
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
              
              <el-form-item label="Additional Options:">
                <el-checkbox-group v-model="payloadForm.options">
                  <el-checkbox label="Bypass AMSI" value="bypass-amsi" />
                  <el-checkbox label="Bypass Defender" value="bypass-defender" />
                  <el-checkbox label="Use HTTPS" value="use-https" />
                  <el-checkbox label="Randomize Variables" value="randomize-vars" />
                  <el-checkbox label="Add Persistence" value="persistence" />
                </el-checkbox-group>
              </el-form-item>
              
              <el-form-item>
                <el-button type="primary" @click="generatePayload" :loading="generatingPayload">
                  <el-icon><Setting /></el-icon>
                  Generate Payload
                </el-button>
                <el-button @click="clearPayloadForm">
                  <el-icon><Delete /></el-icon>
                  Clear
                </el-button>
                <el-button type="info" @click="autoFillFromActiveListener" v-if="hasActiveListener">
                  <el-icon><Connection /></el-icon>
                  Use Active Listener
                </el-button>
              </el-form-item>
            </el-form>
          </div>
          
          <!-- Generated Payload Output -->
          <div v-if="generatedPayload" class="payload-output">
            <h3>Generated Payload:</h3>
            <div class="payload-info">
              <p><strong>Type:</strong> {{ payloadForm.type.toUpperCase() }}</p>
              <p><strong>Target:</strong> {{ payloadForm.targetIP }}:{{ payloadForm.targetPort }}</p>
              <p><strong>Generated:</strong> {{ new Date().toLocaleString() }}</p>
            </div>
            <div class="code-container">
              <el-input
                v-model="generatedPayload"
                type="textarea"
                :rows="15"
                readonly
                class="payload-code"
              />
              <div class="code-actions">
                <el-button type="success" @click="copyPayload">
                  <el-icon><CopyDocument /></el-icon>
                  Copy to Clipboard
                </el-button>
                <el-button type="warning" @click="downloadPayload">
                  <el-icon><Download /></el-icon>
                  Download as .{{ payloadForm.type }}
                </el-button>
                <el-button type="info" @click="saveToHistory">
                  <el-icon><Star /></el-icon>
                  Save to History
                </el-button>
              </div>
            </div>
          </div>
          
          <!-- Payload History -->
          <div class="payload-history">
            <h3>Generated Payloads History</h3>
            <el-table :data="payloadHistory" style="width: 100%" class="payload-history-table">
              <el-table-column prop="timestamp" label="Generated" width="180" />
              <el-table-column prop="type" label="Type" width="100" />
              <el-table-column prop="targetIP" label="Target IP" width="120" />
              <el-table-column prop="targetPort" label="Port" width="80" />
              <el-table-column prop="options" label="Options" width="200" />
              <el-table-column label="Actions" width="150">
                <template #default="scope">
                  <el-button size="small" @click="loadPayloadFromHistory(scope.row)">
                    <el-icon><View /></el-icon>
                    Load
                  </el-button>
                  <el-button size="small" type="danger" @click="deletePayloadFromHistory(scope.row)">
                    <el-icon><Delete /></el-icon>
      </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </el-tab-pane>

      <!-- VNC Panel (Placeholder for future implementation) -->
      <el-tab-pane label="Remote Control" name="vnc">
        <div class="vnc-panel">
          <div class="panel-header">
            <h3>Reverse VNC / Remote Control</h3>
            <el-button type="primary" disabled>
              <el-icon><Monitor /></el-icon>
              Connect VNC (Coming Soon)
      </el-button>
    </div>

          <div class="placeholder-content">
            <el-icon size="64" color="#909399"><Monitor /></el-icon>
            <h3>Remote Control</h3>
            <p>This feature will be implemented later to provide:</p>
            <ul>
              <li>Stream GUI of connected machines</li>
              <li>Control mouse/keyboard through the dashboard</li>
              <li>Screen recording and snapshots</li>
              <li>Real-time remote desktop access</li>
            </ul>
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
            placeholder="Enter PowerShell command, script, or payload execution command..."
          />
        </el-form-item>
        <el-form-item label="Type:">
          <el-select v-model="commandForm.type" placeholder="Select command type">
            <el-option label="PowerShell Command" value="powershell" />
            <el-option label="Script Execution" value="script" />
            <el-option label="Payload Execution" value="payload" />
            <el-option label="System Command" value="system" />
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
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { PayloadGenerator, type PayloadConfig } from '../utils/payloadGenerator'

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
  protocol: string
  host: string
  port: string
  status: string
  connections: number
  createdAt: string
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
const listeners = ref<Listener[]>([
  {
    id: '1',
    name: 'Default TCP',
    protocol: 'TCP',
    host: '0.0.0.0',
    port: '8080',
    status: 'active',
    connections: 0,
    createdAt: new Date().toLocaleString()
  }
])

// Command dialog
const commandDialogVisible = ref(false)
const selectedImplant = ref('')
const selectedImplantName = ref('')
const commandForm = ref({
  command: '',
  type: 'powershell',
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

// Payload generation
const payloadForm = ref({
  targetIP: '',
  targetPort: '4444',
  type: 'ps1',
  outputFormat: 'raw',
  options: ['bypass-amsi', 'randomize-vars']
})
const generatingPayload = ref(false)
const generatedPayload = ref('')
interface PayloadHistoryItem {
  id: string
  timestamp: string
  type: string
  targetIP: string
  targetPort: string
  options: string
  payload: string
}

const payloadHistory = ref<PayloadHistoryItem[]>([])

// Computed properties
const onlineImplants = computed(() => {
  return implants.value.filter(implant => implant.status === 'online')
})

const hasActiveListener = computed(() => {
  return listeners.value.some(listener => listener.status === 'active')
})

const activeListener = computed(() => {
  return listeners.value.find(listener => listener.status === 'active')
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
  
  creatingListener.value = true
  
  try {
    // TODO: Implement actual listener creation logic
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    const newListener = {
      id: Date.now().toString(),
      name: listenerForm.value.name,
      protocol: listenerForm.value.protocol.toUpperCase(),
      host: listenerForm.value.host,
      port: listenerForm.value.port,
      status: 'inactive',
      connections: 0,
      createdAt: new Date().toLocaleString()
    }
    
    listeners.value.push(newListener)
    
    ElMessage.success('Listener created successfully')
    listenerDialogVisible.value = false
    listenerForm.value = { name: '', protocol: 'tcp', host: '0.0.0.0', port: '8080', description: '' }
  } catch (error) {
    ElMessage.error('Failed to create listener')
  } finally {
    creatingListener.value = false
  }
}

const startListener = async (listener: any) => {
  try {
    // TODO: Implement actual listener start logic
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    listener.status = 'active'
    ElMessage.success(`Started listener: ${listener.name}`)
  } catch (error) {
    ElMessage.error('Failed to start listener')
  }
}

const stopListener = async (listener: any) => {
  try {
    await ElMessageBox.confirm(
      `Stop listener ${listener.name}? This will disconnect all active connections.`,
      'Stop Listener',
      { confirmButtonText: 'Stop', cancelButtonText: 'Cancel', type: 'warning' }
    )
    
    // TODO: Implement actual listener stop logic
    listener.status = 'inactive'
    listener.connections = 0
    ElMessage.success(`Stopped listener: ${listener.name}`)
  } catch {
    // User cancelled
  }
}

const deleteListener = async (listener: any) => {
  try {
    await ElMessageBox.confirm(
      `Delete listener profile ${listener.name}?`,
      'Delete Listener',
      { confirmButtonText: 'Delete', cancelButtonText: 'Cancel', type: 'warning' }
    )
    
    listeners.value = listeners.value.filter(l => l.id !== listener.id)
    ElMessage.success('Listener deleted')
  } catch {
    // User cancelled
  }
}

// Payload generation methods
const showPayloadDialog = () => {
  // This method is not needed as the form is already visible
  // Just ensure we're on the payloads tab
  activeTab.value = 'payloads'
}

const generatePayload = async () => {
  if (!payloadForm.value.targetIP || !payloadForm.value.targetPort) {
    ElMessage.error('Please provide target IP and port')
    return
  }
  
  generatingPayload.value = true
  
  try {
    // Use the PayloadGenerator utility
    const payloadConfig: PayloadConfig = {
      targetIP: payloadForm.value.targetIP,
      targetPort: payloadForm.value.targetPort,
      type: payloadForm.value.type,
      outputFormat: payloadForm.value.outputFormat,
      options: payloadForm.value.options
    }
    
    const result = await PayloadGenerator.generatePayload(payloadConfig)
    generatedPayload.value = result.code
    
    ElMessage.success('Payload generated successfully!')
  } catch (error) {
    ElMessage.error('Failed to generate payload: ' + error)
  } finally {
    generatingPayload.value = false
  }
}



const clearPayloadForm = () => {
  payloadForm.value = {
    targetIP: '',
    targetPort: '4444',
    type: 'ps1',
    outputFormat: 'raw',
    options: ['bypass-amsi', 'randomize-vars']
  }
  generatedPayload.value = ''
}

const copyPayload = async () => {
  try {
    await navigator.clipboard.writeText(generatedPayload.value)
    ElMessage.success('Payload copied to clipboard')
  } catch (error) {
    ElMessage.error('Failed to copy payload')
  }
}

const downloadPayload = () => {
  const blob = new Blob([generatedPayload.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `payload_${payloadForm.value.targetIP}_${payloadForm.value.targetPort}.${payloadForm.value.type}`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('Payload downloaded')
}

const saveToHistory = () => {
  const historyItem = {
    id: Date.now().toString(),
    timestamp: new Date().toLocaleString(),
    type: payloadForm.value.type.toUpperCase(),
    targetIP: payloadForm.value.targetIP,
    targetPort: payloadForm.value.targetPort,
    options: payloadForm.value.options.join(', '),
    payload: generatedPayload.value
  }
  
  payloadHistory.value.unshift(historyItem)
  ElMessage.success('Payload saved to history')
}

const loadPayloadFromHistory = (item: any) => {
  payloadForm.value.targetIP = item.targetIP
  payloadForm.value.targetPort = item.targetPort
  generatedPayload.value = item.payload
  ElMessage.success('Payload loaded from history')
}

const deletePayloadFromHistory = async (item: any) => {
  try {
    await ElMessageBox.confirm(
      'Delete this payload from history?',
      'Delete Payload',
      { confirmButtonText: 'Delete', cancelButtonText: 'Cancel', type: 'warning' }
    )
    
    payloadHistory.value = payloadHistory.value.filter(h => h.id !== item.id)
    ElMessage.success('Payload deleted from history')
  } catch {
    // User cancelled
  }
}

const autoFillFromActiveListener = () => {
  if (activeListener.value) {
    payloadForm.value.targetIP = activeListener.value.host === '0.0.0.0' ? '127.0.0.1' : activeListener.value.host
    payloadForm.value.targetPort = activeListener.value.port
    ElMessage.success(`Auto-filled from active listener: ${activeListener.value.name}`)
  }
}

// Utility functions
const getImplantTypeColor = (type: string) => {
  switch (type) {
    case 'reverse-shell': return 'primary'
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
  updateStats()
  loadDashboardData()
})

// Update dashboard statistics
const updateStats = () => {
  stats.value.totalImplants = implants.value.length
  stats.value.activeListeners = listeners.value.filter(l => l.status === 'active').length
  stats.value.runningTasks = tasks.value.filter(t => t.status === 'running').length
  
  const onlineImplants = implants.value.filter(i => i.status === 'online')
  if (onlineImplants.length > 0) {
    stats.value.lastCheckin = onlineImplants[0].lastCheckin
  }
}

// Load dashboard data from backend
const loadDashboardData = async () => {
  try {
    // TODO: Replace with actual API calls
    // const implantsData = await fetch('/api/implants')
    // const tasksData = await fetch('/api/tasks')
    // const listenersData = await fetch('/api/listeners')
    
    // For now, use mock data but update stats
    updateStats()
  } catch (error) {
    console.error('Failed to load dashboard data:', error)
  }
}
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

/* Payloads Panel */
.payloads-panel {
  padding: 20px;
}

.payload-generator {
  background: var(--secondary-black);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
}

.payload-form {
  max-width: 800px;
}

.payload-output {
  background: var(--secondary-black);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
}

.payload-info {
  display: flex;
  gap: 20px;
  margin-bottom: 15px;
  padding: 10px;
  background: var(--primary-black);
  border-radius: 4px;
}

.payload-info p {
  margin: 0;
  font-size: 14px;
}

.code-container {
  position: relative;
}

.payload-code {
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.code-actions {
  display: flex;
  gap: 10px;
  margin-top: 15px;
  justify-content: center;
}

.payload-history {
  background: var(--secondary-black);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 20px;
}

.payload-history-table {
  margin-top: 15px;
}

/* Payload code textarea styling */
.payload-code :deep(.el-textarea__inner) {
  font-family: 'Courier New', monospace;
}

/* Empty state styling */
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-gray);
  
  h3 {
    color: var(--text-white);
    margin: 20px 0 15px 0;
    font-size: 18px;
    font-weight: 600;
  }
  
  p {
    color: var(--text-gray);
    font-size: 14px;
    margin: 0;
  }
}

/* Fix text colors for all panels */
.implants-panel,
.tasks-panel,
.listeners-panel,
.payloads-panel,
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
.payloads-panel *,
.overview-panel * {
  background: transparent !important;
}

/* Ensure panel content has proper backgrounds */
.implants-panel > *,
.tasks-panel > *,
.listeners-panel > *,
.payloads-panel > *,
.overview-panel > * {
  background: transparent !important;
}

/* Fix any div elements that might have white backgrounds */
.implants-panel div,
.tasks-panel div,
.listeners-panel div,
.payloads-panel div,
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
</style>

