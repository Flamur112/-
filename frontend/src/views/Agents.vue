<template>
  <div class="agents">
    <!-- Header Actions -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">Agent Management</h1>
        <p class="page-subtitle">Monitor and manage connected agents</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" icon="Plus" @click="showAddAgentDialog">
          Add Agent
        </el-button>
        <el-button icon="Refresh" @click="refreshAgents">
          Refresh
        </el-button>
      </div>
    </div>

    <!-- Filters and Search -->
    <el-card class="filter-card" shadow="hover">
      <div class="filter-row">
        <el-input
          v-model="searchQuery"
          placeholder="Search agents..."
          prefix-icon="Search"
          clearable
          style="width: 300px"
        />
        <el-select v-model="statusFilter" placeholder="Status" clearable style="width: 150px">
          <el-option label="All" value="" />
          <el-option label="Online" value="online" />
          <el-option label="Offline" value="offline" />
          <el-option label="Error" value="error" />
        </el-select>
        <el-select v-model="platformFilter" placeholder="Platform" clearable style="width: 150px">
          <el-option label="All" value="" />
          <el-option label="Windows" value="windows" />
          <el-option label="Linux" value="linux" />
          <el-option label="macOS" value="macos" />
        </el-select>
      </div>
    </el-card>

    <!-- Agents Table -->
    <el-card class="agents-table-card" shadow="hover">
      <el-table
        :data="filteredAgents"
        v-loading="loading"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        
        <el-table-column prop="id" label="Agent ID" width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.id }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="name" label="Name" width="200">
          <template #default="{ row }">
            <div class="agent-name">
              <el-icon class="platform-icon" :class="row.platform">
                <component :is="getPlatformIcon(row.platform)" />
              </el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column prop="status" label="Status" width="120">
          <template #default="{ row }">
            <el-tag
              :type="getStatusType(row.status)"
              size="small"
              effect="dark"
            >
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="platform" label="Platform" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.platform }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="ip" label="IP Address" width="140" />
        
        <el-table-column prop="lastSeen" label="Last Seen" width="180">
          <template #default="{ row }">
            {{ formatTime(row.lastSeen) }}
          </template>
        </el-table-column>
        
        <el-table-column prop="version" label="Version" width="100" />
        
        <el-table-column label="Actions" width="200" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-button
                size="small"
                icon="Monitor"
                @click="viewAgentDetails(row)"
                title="View Details"
              />
              <el-button
                size="small"
                icon="Terminal"
                @click="openTerminal(row)"
                title="Open Terminal"
              />
              <el-button
                size="small"
                icon="Setting"
                @click="configureAgent(row)"
                title="Configure"
              />
              <el-button
                size="small"
                icon="Delete"
                type="danger"
                @click="removeAgent(row)"
                title="Remove"
              />
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
      
      <!-- Pagination -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="totalAgents"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- Add Agent Dialog -->
    <el-dialog
      v-model="addAgentDialogVisible"
      title="Add New Agent"
      width="500px"
      @close="resetAddAgentForm"
    >
      <el-form
        ref="addAgentFormRef"
        :model="addAgentForm"
        :rules="addAgentRules"
        label-width="100px"
      >
        <el-form-item label="Agent Name" prop="name">
          <el-input v-model="addAgentForm.name" placeholder="Enter agent name" />
        </el-form-item>
        
        <el-form-item label="Platform" prop="platform">
          <el-select v-model="addAgentForm.platform" placeholder="Select platform" style="width: 100%">
            <el-option label="Windows" value="windows" />
            <el-option label="Linux" value="linux" />
            <el-option label="macOS" value="macos" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="IP Address" prop="ip">
          <el-input v-model="addAgentForm.ip" placeholder="Enter IP address" />
        </el-form-item>
        
        <el-form-item label="Description" prop="description">
          <el-input
            v-model="addAgentForm.description"
            type="textarea"
            :rows="3"
            placeholder="Enter description"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="addAgentDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleAddAgent" :loading="addingAgent">
          Add Agent
        </el-button>
      </template>
    </el-dialog>

    <!-- Agent Details Dialog -->
    <el-dialog
      v-model="agentDetailsDialogVisible"
      title="Agent Details"
      width="700px"
    >
      <div v-if="selectedAgent" class="agent-details">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Agent ID">{{ selectedAgent.id }}</el-descriptions-item>
          <el-descriptions-item label="Name">{{ selectedAgent.name }}</el-descriptions-item>
          <el-descriptions-item label="Status">
            <el-tag :type="getStatusType(selectedAgent.status)" effect="dark">
              {{ selectedAgent.status }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Platform">{{ selectedAgent.platform }}</el-descriptions-item>
          <el-descriptions-item label="IP Address">{{ selectedAgent.ip }}</el-descriptions-item>
          <el-descriptions-item label="Version">{{ selectedAgent.version }}</el-descriptions-item>
          <el-descriptions-item label="Last Seen" :span="2">
            {{ formatTime(selectedAgent.lastSeen) }}
          </el-descriptions-item>
          <el-descriptions-item label="Description" :span="2">
            {{ selectedAgent.description || 'No description available' }}
          </el-descriptions-item>
        </el-descriptions>
        
        <div class="agent-metrics">
          <h4>Performance Metrics</h4>
          <el-row :gutter="20">
            <el-col :span="8">
              <div class="metric-item">
                <div class="metric-value">{{ selectedAgent.metrics?.cpu || '0' }}%</div>
                <div class="metric-label">CPU Usage</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="metric-item">
                <div class="metric-value">{{ selectedAgent.metrics?.memory || '0' }}%</div>
                <div class="metric-label">Memory Usage</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="metric-item">
                <div class="metric-value">{{ selectedAgent.metrics?.disk || '0' }}%</div>
                <div class="metric-label">Disk Usage</div>
              </div>
            </el-col>
          </el-row>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

dayjs.extend(relativeTime)

// Reactive data
const loading = ref(false)
const searchQuery = ref('')
const statusFilter = ref('')
const platformFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const selectedAgents = ref<any[]>([])
const addAgentDialogVisible = ref(false)
const agentDetailsDialogVisible = ref(false)
const selectedAgent = ref<any>(null)
const addingAgent = ref(false)

// Form refs
const addAgentFormRef = ref<FormInstance>()

// Form data
const addAgentForm = ref({
  name: '',
  platform: '',
  ip: '',
  description: ''
})

// Form validation rules
const addAgentRules: FormRules = {
  name: [
    { required: true, message: 'Please enter agent name', trigger: 'blur' }
  ],
  platform: [
    { required: true, message: 'Please select platform', trigger: 'change' }
  ],
  ip: [
    { required: true, message: 'Please enter IP address', trigger: 'blur' },
    { pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/, message: 'Please enter valid IP address', trigger: 'blur' }
  ]
}

// Empty agents data - no dummy data
const agents = ref([])

// Computed properties
const filteredAgents = computed(() => {
  let filtered = agents.value

  if (searchQuery.value) {
    filtered = filtered.filter(agent =>
      agent.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      agent.id.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      agent.ip.includes(searchQuery.value)
    )
  }

  if (statusFilter.value) {
    filtered = filtered.filter(agent => agent.status === statusFilter.value)
  }

  if (platformFilter.value) {
    filtered = filtered.filter(agent => agent.platform === platformFilter.value)
  }

  return filtered
})

const totalAgents = computed(() => filteredAgents.value.length)

// Methods
const refreshAgents = () => {
  loading.value = true
  setTimeout(() => {
    loading.value = false
    ElMessage.success('Agents refreshed successfully')
  }, 1000)
}

const handleSelectionChange = (selection: any[]) => {
  selectedAgents.value = selection
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
}

const handleCurrentChange = (page: number) => {
  currentPage.value = page
}

const showAddAgentDialog = () => {
  addAgentDialogVisible.value = true
}

const resetAddAgentForm = () => {
  addAgentForm.value = {
    name: '',
    platform: '',
    ip: '',
    description: ''
  }
  addAgentFormRef.value?.clearValidate()
}

const handleAddAgent = async () => {
  if (!addAgentFormRef.value) return

  try {
    await addAgentFormRef.value.validate()
    addingAgent.value = true

    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1000))

    const newAgent = {
      id: `${addAgentForm.value.platform.toUpperCase()}-${Math.random().toString(36).substr(2, 6).toUpperCase()}`,
      name: addAgentForm.value.name,
      status: 'offline',
      platform: addAgentForm.value.platform,
      ip: addAgentForm.value.ip,
      lastSeen: new Date(),
      version: '1.0.0',
      description: addAgentForm.value.description,
      metrics: { cpu: 0, memory: 0, disk: 0 }
    }

    agents.value.unshift(newAgent)
    addAgentDialogVisible.value = false
    ElMessage.success('Agent added successfully')
  } catch (error) {
    console.error('Add agent error:', error)
  } finally {
    addingAgent.value = false
  }
}

const viewAgentDetails = (agent: any) => {
  selectedAgent.value = agent
  agentDetailsDialogVisible.value = true
}

const openTerminal = (agent: any) => {
  ElMessage.info(`Opening terminal for ${agent.name}`)
}

const configureAgent = (agent: any) => {
  ElMessage.info(`Configuring ${agent.name}`)
}

const removeAgent = async (agent: any) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to remove agent "${agent.name}"?`,
      'Remove Agent',
      {
        confirmButtonText: 'Remove',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    const index = agents.value.findIndex(a => a.id === agent.id)
    if (index > -1) {
      agents.value.splice(index, 1)
      ElMessage.success('Agent removed successfully')
    }
  } catch {
    // User cancelled
  }
}

const getStatusType = (status: string) => {
  switch (status) {
    case 'online': return 'success'
    case 'offline': return 'info'
    case 'error': return 'danger'
    default: return 'info'
  }
}

const getPlatformIcon = (platform: string) => {
  switch (platform) {
    case 'windows': return 'Monitor'
    case 'linux': return 'Terminal'
    case 'macos': return 'Apple'
    default: return 'Monitor'
  }
}

const formatTime = (timestamp: Date) => {
  return dayjs(timestamp).fromNow()
}

// Lifecycle
onMounted(() => {
  // Initialize data
})
</script>

<style scoped lang="scss">
.agents {
  background: var(--primary-black);
  color: var(--text-white);
  min-height: 100vh;
  padding: 20px;

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 24px;

    .header-left {
      .page-title {
        font-size: 24px;
        font-weight: 600;
        color: var(--text-white);
        margin: 0 0 8px 0;
      }

      .page-subtitle {
        color: var(--text-gray);
        margin: 0;
        font-size: 16px;
      }
    }

    .header-actions {
      display: flex;
      gap: 12px;
    }
  }

  .filter-card {
    margin-bottom: 24px;
    background: var(--secondary-black) !important;
    border: 1px solid var(--border-color) !important;

    :deep(.el-card__body) {
      background: var(--secondary-black) !important;
      color: var(--text-white) !important;
    }

    .filter-row {
      display: flex;
      gap: 16px;
      align-items: center;
    }
  }

  .agents-table-card {
    background: var(--secondary-black) !important;
    border: 1px solid var(--border-color) !important;

    :deep(.el-card__body) {
      background: var(--secondary-black) !important;
      color: var(--text-white) !important;
    }

    .agent-name {
      display: flex;
      align-items: center;
      gap: 8px;

      .platform-icon {
        font-size: 16px;
        color: var(--text-gray);

        &.windows { color: #0078d4; }
        &.linux { color: #fcc624; }
        &.macos { color: #ffffff; }
      }
    }

    .pagination-wrapper {
      display: flex;
      justify-content: center;
      margin-top: 24px;
    }
  }

  .agent-details {
    .agent-metrics {
      margin-top: 24px;

      h4 {
        margin: 0 0 16px 0;
        color: var(--text-white);
      }

      .metric-item {
        text-align: center;
        padding: 16px;
        background-color: var(--primary-black);
        border-radius: 8px;
        border: 1px solid var(--border-color);

        .metric-value {
          font-size: 24px;
          font-weight: 600;
          color: var(--primary-color);
          margin-bottom: 8px;
        }

        .metric-label {
          font-size: 14px;
          color: var(--text-gray);
        }
      }
    }
  }
}

/* Dark theme for all Element Plus components */
:deep(.el-card) {
  background: var(--secondary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-card__header) {
  background: var(--primary-black) !important;
  border-bottom-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-card__body) {
  background: var(--secondary-black) !important;
  color: var(--text-white) !important;
}

/* Form styling */
:deep(.el-input__inner) {
  background: var(--primary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-input__inner::placeholder) {
  color: var(--text-gray) !important;
}

:deep(.el-select .el-input__inner) {
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

/* Table styling */
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

/* Tags */
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

:deep(.el-tag--info) {
  background: #909399 !important;
  color: var(--text-white) !important;
  border-color: #909399 !important;
}

/* Pagination */
:deep(.el-pagination) {
  background: transparent !important;
  color: var(--text-white) !important;
}

:deep(.el-pagination .el-pager li) {
  background: var(--primary-black) !important;
  color: var(--text-white) !important;
  border-color: var(--border-color) !important;
}

:deep(.el-pagination .el-pager li.is-active) {
  background: var(--primary-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-pagination .btn-prev),
:deep(.el-pagination .btn-next) {
  background: var(--primary-black) !important;
  color: var(--text-white) !important;
  border-color: var(--border-color) !important;
}

@media (max-width: 768px) {
  .agents {
    .page-header {
      flex-direction: column;
      gap: 16px;

      .header-actions {
        width: 100%;
        justify-content: flex-end;
      }
    }

    .filter-card .filter-row {
      flex-direction: column;
      align-items: stretch;
    }
  }
}
</style>

