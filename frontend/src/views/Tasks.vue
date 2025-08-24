<template>
  <div class="tasks">
    <!-- Header Actions -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">Task Management</h1>
        <p class="page-subtitle">Create, monitor, and manage tasks across agents</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" icon="Plus" @click="showCreateTaskDialog">
          Create Task
        </el-button>
        <el-button icon="Refresh" @click="refreshTasks">
          Refresh
        </el-button>
      </div>
    </div>



    <!-- Filters and Search -->
    <el-card class="filter-card" shadow="hover">
      <div class="filter-row">
        <el-input
          v-model="searchQuery"
          placeholder="Search tasks..."
          prefix-icon="Search"
          clearable
          style="width: 300px"
        />
        <el-select v-model="statusFilter" placeholder="Status" clearable style="width: 150px">
          <el-option label="All" value="" />
          <el-option label="Pending" value="pending" />
          <el-option label="Running" value="running" />
          <el-option label="Completed" value="completed" />
          <el-option label="Failed" value="failed" />
        </el-select>
        <el-select v-model="typeFilter" placeholder="Type" clearable style="width: 150px">
          <el-option label="All" value="" />
          <el-option label="Command" value="command" />
          <el-option label="File Upload" value="upload" />
          <el-option label="File Download" value="download" />
          <el-option label="System Info" value="sysinfo" />
        </el-select>
        <el-select v-model="agentFilter" placeholder="Agent" clearable style="width: 200px">
          <el-option label="All" value="" />
          <el-option
            v-for="agent in availableAgents"
            :key="agent.id"
            :label="agent.name"
            :value="agent.id"
          />
        </el-select>
      </div>
    </el-card>

    <!-- Tasks Table -->
    <el-card class="tasks-table-card" shadow="hover">
      <el-table
        :data="filteredTasks"
        v-loading="loading"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        
        <el-table-column prop="id" label="Task ID" width="120">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.id }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="name" label="Task Name" width="200" />
        
        <el-table-column prop="type" label="Type" width="120">
          <template #default="{ row }">
            <el-tag :type="getTaskTypeColor(row.type)" size="small">
              {{ row.type }}
            </el-tag>
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
        
        <el-table-column prop="agent" label="Target Agent" width="180">
          <template #default="{ row }">
            <div class="agent-info">
              <el-icon class="platform-icon" :class="row.agent?.platform">
                <component :is="getPlatformIcon(row.agent?.platform)" />
              </el-icon>
              <span>{{ row.agent?.name || 'All Agents' }}</span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column prop="createdAt" label="Created" width="180">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>
        
        <el-table-column prop="updatedAt" label="Updated" width="180">
          <template #default="{ row }">
            {{ formatTime(row.updatedAt) }}
          </template>
        </el-table-column>
        
        <el-table-column label="Actions" width="200" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-button
                size="small"
                icon="View"
                @click="viewTaskDetails(row)"
                title="View Details"
              />
              <el-button
                v-if="row.status === 'pending'"
                size="small"
                icon="VideoPlay"
                type="success"
                @click="executeTask(row)"
                title="Execute"
              />
              <el-button
                v-if="row.status === 'running'"
                size="small"
                icon="VideoPause"
                type="warning"
                @click="pauseTask(row)"
                title="Pause"
              />
              <el-button
                size="small"
                icon="Delete"
                type="danger"
                @click="deleteTask(row)"
                title="Delete"
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
          :total="totalTasks"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- Create Task Dialog -->
    <el-dialog
      v-model="createTaskDialogVisible"
      title="Create New Task"
      width="600px"
      @close="resetCreateTaskForm"
    >
      <el-form
        ref="createTaskFormRef"
        :model="createTaskForm"
        :rules="createTaskRules"
        label-width="120px"
      >
        <el-form-item label="Task Name" prop="name">
          <el-input v-model="createTaskForm.name" placeholder="Enter task name" />
        </el-form-item>
        
        <el-form-item label="Task Type" prop="type">
          <el-select v-model="createTaskForm.type" placeholder="Select task type" style="width: 100%">
            <el-option label="Command Execution" value="command" />
            <el-option label="File Upload" value="upload" />
            <el-option label="File Download" value="download" />
            <el-option label="System Information" value="sysinfo" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="Target Agent" prop="agentId">
          <el-select v-model="createTaskForm.agentId" placeholder="Select target agent" style="width: 100%">
            <el-option label="All Agents" value="" />
            <el-option
              v-for="agent in availableAgents"
              :key="agent.id"
              :label="agent.name"
              :value="agent.id"
            />
          </el-select>
        </el-form-item>
        
        <el-form-item
          v-if="createTaskForm.type === 'command'"
          label="Command" prop="command"
        >
          <el-input
            v-model="createTaskForm.command"
            type="textarea"
            :rows="3"
            placeholder="Enter command to execute"
          />
        </el-form-item>
        
        <el-form-item
          v-if="createTaskForm.type === 'upload'"
          label="File Path" prop="filePath"
        >
          <el-input v-model="createTaskForm.filePath" placeholder="Enter target file path" />
        </el-form-item>
        
        <el-form-item
          v-if="createTaskForm.type === 'download'"
          label="File Path" prop="filePath"
        >
          <el-input v-model="createTaskForm.filePath" placeholder="Enter source file path" />
        </el-form-item>
        
        <el-form-item label="Priority" prop="priority">
          <el-select v-model="createTaskForm.priority" placeholder="Select priority" style="width: 100%">
            <el-option label="Low" value="low" />
            <el-option label="Normal" value="normal" />
            <el-option label="High" value="high" />
            <el-option label="Critical" value="critical" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="Description" prop="description">
          <el-input
            v-model="createTaskForm.description"
            type="textarea"
            :rows="3"
            placeholder="Enter task description"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="createTaskDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="handleCreateTask" :loading="creatingTask">
          Create Task
        </el-button>
      </template>
    </el-dialog>

    <!-- Task Details Dialog -->
    <el-dialog
      v-model="taskDetailsDialogVisible"
      title="Task Details"
      width="800px"
    >
      <div v-if="selectedTask" class="task-details">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Task ID">{{ selectedTask.id }}</el-descriptions-item>
          <el-descriptions-item label="Name">{{ selectedTask.name }}</el-descriptions-item>
          <el-descriptions-item label="Type">
            <el-tag :type="getTaskTypeColor(selectedTask.type)">
              {{ selectedTask.type }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Status">
            <el-tag :type="getStatusType(selectedTask.status)" effect="dark">
              {{ selectedTask.status }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Priority">
            <el-tag :type="getPriorityType(selectedTask.priority)">
              {{ selectedTask.priority }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Target Agent">
            {{ selectedTask.agent?.name || 'All Agents' }}
          </el-descriptions-item>
          <el-descriptions-item label="Created" :span="2">
            {{ formatTime(selectedTask.createdAt) }}
          </el-descriptions-item>
          <el-descriptions-item label="Updated" :span="2">
            {{ formatTime(selectedTask.updatedAt) }}
          </el-descriptions-item>
          <el-descriptions-item label="Description" :span="2">
            {{ selectedTask.description || 'No description available' }}
          </el-descriptions-item>
        </el-descriptions>
        
        <div v-if="selectedTask.result" class="task-result">
          <h4>Execution Result</h4>
          <el-input
            v-model="selectedTask.result"
            type="textarea"
            :rows="8"
            readonly
            placeholder="No result available"
          />
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
const typeFilter = ref('')
const agentFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const selectedTasks = ref<any[]>([])
const createTaskDialogVisible = ref(false)
const taskDetailsDialogVisible = ref(false)
const selectedTask = ref<any>(null)
const creatingTask = ref(false)

// Form refs
const createTaskFormRef = ref<FormInstance>()

// Form data
const createTaskForm = ref({
  name: '',
  type: '',
  agentId: '',
  command: '',
  filePath: '',
  priority: 'normal',
  description: ''
})

// Form validation rules
const createTaskRules: FormRules = {
  name: [
    { required: true, message: 'Please enter task name', trigger: 'blur' }
  ],
  type: [
    { required: true, message: 'Please select task type', trigger: 'change' }
  ],
  command: [
    { required: true, message: 'Please enter command', trigger: 'blur' }
  ],
  filePath: [
    { required: true, message: 'Please enter file path', trigger: 'blur' }
  ]
}

// Empty data - no dummy data
const availableAgents = ref([])
const tasks = ref([])

// Computed properties
const filteredTasks = computed(() => {
  let filtered = tasks.value

  if (searchQuery.value) {
    filtered = filtered.filter(task =>
      task.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      task.id.toLowerCase().includes(searchQuery.value.toLowerCase())
    )
  }

  if (statusFilter.value) {
    filtered = filtered.filter(task => task.status === statusFilter.value)
  }

  if (typeFilter.value) {
    filtered = filtered.filter(task => task.type === typeFilter.value)
  }

  if (agentFilter.value) {
    filtered = filtered.filter(task => task.agent?.id === agentFilter.value)
  }

  return filtered
})

const totalTasks = computed(() => filteredTasks.value.length)

// Methods
const refreshTasks = () => {
  loading.value = true
  setTimeout(() => {
    loading.value = false
    ElMessage.success('Tasks refreshed successfully')
  }, 1000)
}

const handleSelectionChange = (selection: any[]) => {
  selectedTasks.value = selection
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
}

const handleCurrentChange = (page: number) => {
  currentPage.value = page
}

const showCreateTaskDialog = () => {
  createTaskDialogVisible.value = true
}

const resetCreateTaskForm = () => {
  createTaskForm.value = {
    name: '',
    type: '',
    agentId: '',
    command: '',
    filePath: '',
    priority: 'normal',
    description: ''
  }
  createTaskFormRef.value?.clearValidate()
}

const handleCreateTask = async () => {
  if (!createTaskFormRef.value) return

  try {
    await createTaskFormRef.value.validate()
    creatingTask.value = true

    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1000))

    const newTask = {
      id: `TASK-${Math.random().toString(36).substr(2, 6).toUpperCase()}`,
      name: createTaskForm.value.name,
      type: createTaskForm.value.type,
      status: 'pending',
      agent: createTaskForm.value.agentId ? availableAgents.value.find(a => a.id === createTaskForm.value.agentId) : null,
      priority: createTaskForm.value.priority,
      createdAt: new Date(),
      updatedAt: new Date(),
      description: createTaskForm.value.description,
      result: null
    }

    tasks.value.unshift(newTask)
    createTaskDialogVisible.value = false
    ElMessage.success('Task created successfully')
  } catch (error) {
    console.error('Create task error:', error)
  } finally {
    creatingTask.value = false
  }
}

const viewTaskDetails = (task: any) => {
  selectedTask.value = task
  taskDetailsDialogVisible.value = true
}

const executeTask = (task: any) => {
  task.status = 'running'
  task.updatedAt = new Date()
  ElMessage.success(`Task ${task.name} started execution`)
}

const pauseTask = (task: any) => {
  task.status = 'pending'
  task.updatedAt = new Date()
  ElMessage.success(`Task ${task.name} paused`)
}

const deleteTask = async (task: any) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to delete task "${task.name}"?`,
      'Delete Task',
      {
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    const index = tasks.value.findIndex(t => t.id === task.id)
    if (index > -1) {
      tasks.value.splice(index, 1)
      ElMessage.success('Task deleted successfully')
    }
  } catch {
    // User cancelled
  }
}

const getStatusType = (status: string) => {
  switch (status) {
    case 'pending': return 'info'
    case 'running': return 'warning'
    case 'completed': return 'success'
    case 'failed': return 'danger'
    default: return 'info'
  }
}

const getTaskTypeColor = (type: string) => {
  switch (type) {
    case 'command': return 'primary'
    case 'upload': return 'success'
    case 'download': return 'warning'
    case 'sysinfo': return 'info'
    default: return 'info'
  }
}

const getPriorityType = (priority: string) => {
  switch (priority) {
    case 'low': return 'info'
    case 'normal': return 'success'
    case 'high': return 'warning'
    case 'critical': return 'danger'
    default: return 'success'
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
.tasks {
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
        text-shadow: 0 0 10px rgba(255, 255, 255, 0.1);
      }

      .page-subtitle {
        color: var(--text-gray);
        margin: 0;
        font-size: 16px;
        opacity: 0.8;
      }
    }

    .header-actions {
      display: flex;
      gap: 12px;
    }
  }

  .task-stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
    margin-bottom: 24px;

    .stat-card {
      .stat-content {
        display: flex;
        align-items: center;
        padding: 8px 0;
      }

      .stat-icon {
        width: 50px;
        height: 50px;
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-right: 16px;
        font-size: 20px;
        color: #fff;

        &.pending { background: linear-gradient(135deg, #1890ff, #40a9ff); }
        &.running { background: linear-gradient(135deg, #fa8c16, #ffa940); }
        &.completed { background: linear-gradient(135deg, #52c41a, #73d13d); }
        &.failed { background: linear-gradient(135deg, #f5222d, #ff4d4f); }
      }

      .stat-info {
        .stat-number {
          font-size: 24px;
          font-weight: 700;
          color: var(--text-white);
          line-height: 1;
          margin-bottom: 4px;
        }

        .stat-label {
          font-size: 14px;
          color: var(--text-gray);
        }
      }
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

  .tasks-table-card {
    background: var(--secondary-black) !important;
    border: 1px solid var(--border-color) !important;

    :deep(.el-card__body) {
      background: var(--secondary-black) !important;
      color: var(--text-white) !important;
    }

    .agent-info {
      display: flex;
      align-items: center;
      gap: 8px;

      .platform-icon {
        font-size: 14px;
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

  .task-details {
    .task-result {
      margin-top: 24px;

      h4 {
        margin: 0 0 16px 0;
        color: var(--text-white);
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

:deep(.el-tag--warning) {
  background: #e6a23c !important;
  color: var(--text-white) !important;
  border-color: #e6a23c !important;
}

:deep(.el-tag--primary) {
  background: var(--primary-color) !important;
  color: var(--text-white) !important;
  border-color: var(--primary-color) !important;
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

/* Buttons */
:deep(.el-button:not([type])) {
  background-color: var(--secondary-black) !important;
  border-color: var(--border-color) !important;
  color: var(--text-white) !important;
}

:deep(.el-button:not([type]):hover) {
  background-color: var(--primary-black) !important;
  border-color: var(--primary-color) !important;
}

@media (max-width: 768px) {
  .tasks {
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

    .task-stats {
      grid-template-columns: repeat(2, 1fr);
    }
  }
}
</style>

