<template>
  <div class="auth-container">
    <!-- Logo and Title Section -->
    <div class="auth-header">
      <div class="logo">
        <img src="/favicon.ico" alt="MulisC2" class="favicon" />
        <h1 class="title">MulisC2</h1>
      </div>
    </div>

    <!-- Main Content Area -->
    <div class="auth-content">
      <!-- Tab Navigation -->
      <div class="auth-tabs">
        <el-tabs v-model="activeTab" class="auth-tabs-content">
          <el-tab-pane label="Login" name="login">
            <!-- Login Form -->
            <el-form
              ref="loginFormRef"
              :model="loginForm"
              :rules="loginRules"
              class="auth-form"
              @submit.prevent="handleLogin"
            >
              <el-form-item prop="username">
                <el-input
                  v-model="loginForm.username"
                  placeholder="Username"
                  size="large"
                  prefix-icon="User"
                  clearable
                />
              </el-form-item>
              
              <el-form-item prop="password">
                <el-input
                  v-model="loginForm.password"
                  type="password"
                  placeholder="Password"
                  size="large"
                  prefix-icon="Lock"
                  show-password
                  clearable
                  @keyup.enter="handleLogin"
                />
              </el-form-item>
              
              <el-form-item>
                <el-checkbox v-model="loginForm.remember">Remember me</el-checkbox>
              </el-form-item>
              
              <el-form-item>
                <el-button
                  type="primary"
                  size="large"
                  class="auth-btn"
                  :loading="loginLoading"
                  @click="handleLogin"
                >
                  {{ loginLoading ? 'Signing in...' : 'Sign In' }}
                </el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="Register" name="register">
            <!-- Registration Form -->
            <el-form
              ref="registerFormRef"
              :model="registerForm"
              :rules="registerRules"
              class="auth-form"
              @submit.prevent="handleRegister"
            >
              <el-form-item prop="username">
                <el-input
                  v-model="registerForm.username"
                  placeholder="Username"
                  size="large"
                  prefix-icon="User"
                  clearable
                />
              </el-form-item>

              <el-form-item prop="password">
                <el-input
                  v-model="registerForm.password"
                  type="password"
                  placeholder="Password"
                  size="large"
                  prefix-icon="Lock"
                  show-password
                  clearable
                />
              </el-form-item>

              <el-form-item prop="confirmPassword">
                <el-input
                  v-model="registerForm.confirmPassword"
                  type="password"
                  placeholder="Confirm Password"
                  size="large"
                  prefix-icon="Lock"
                  show-password
                  clearable
                />
              </el-form-item>
              
              <el-form-item>
                <el-button
                  type="primary"
                  size="large"
                  class="auth-btn"
                  :loading="registerLoading"
                  @click="handleRegister"
                >
                  {{ registerLoading ? 'Creating Account...' : 'Create Account' }}
                </el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </div>

      <!-- Footer -->
      <div class="auth-footer">
        <p class="version">Version 1.0.0</p>
        <p class="copyright">© 2025 MulisC2. All rights reserved.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useAuth } from '@/services/auth'

const { login, register } = useAuth()

const activeTab = ref('login')
const loginFormRef = ref<FormInstance>()
const registerFormRef = ref<FormInstance>()
const loginLoading = ref(false)
const registerLoading = ref(false)

// Login form
const loginForm = reactive({
  username: '',
  password: '',
  remember: false
})

// Registration form
const registerForm = reactive({
  username: '',
  password: '',
  confirmPassword: ''
})

// Validation rules
const loginRules: FormRules = {
  username: [
    { required: true, message: 'Please enter username', trigger: 'blur' }
  ],
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' }
  ]
}

const validateConfirmPassword = (rule: any, value: string, callback: any) => {
  if (value === '') {
    callback(new Error('Please confirm your password'))
  } else if (value !== registerForm.password) {
    callback(new Error('Passwords do not match'))
  } else {
    callback()
  }
}

const registerRules: FormRules = {
  username: [
    { required: true, message: 'Please enter username', trigger: 'blur' },
    { min: 3, max: 20, message: 'Length should be 3 to 20 characters', trigger: 'blur' }
  ],

  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// Handle login
const handleLogin = async () => {
  if (!loginFormRef.value) return
  
  try {
    await loginFormRef.value.validate()
    loginLoading.value = true

    const success = await login({
      username: loginForm.username,
      password: loginForm.password
    })

    if (success) {
      ElMessage.success('Login successful!')
    }
  } catch (error: any) {
    ElMessage.error(error.message || 'Login failed')
  } finally {
    loginLoading.value = false
  }
}

// Handle registration
const handleRegister = async () => {
  if (!registerFormRef.value) return
  
  try {
    await registerFormRef.value.validate()
    registerLoading.value = true

    const success = await register({
      username: registerForm.username,
      password: registerForm.password
    })

    if (success) {
      ElMessage.success('Account created successfully! Please login.')
      activeTab.value = 'login'
      // Clear registration form
      registerForm.username = ''
      registerForm.password = ''
      registerForm.confirmPassword = ''
    }
  } catch (error: any) {
    ElMessage.error(error.message || 'Registration failed')
  } finally {
    registerLoading.value = false
  }
}
</script>

<style scoped>
.auth-container {
  width: 100%;
  max-width: 480px;
  margin: 0 auto;
  padding: 40px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
}

.auth-content {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.auth-header {
  text-align: center;
  margin-bottom: 40px;
  width: 100%;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  flex-direction: column;
}

.favicon {
  width: 48px;
  height: 48px;
  margin-right: 12px;
}

.title {
  color: #fff;
  margin: 0;
  font-size: 32px;
  font-weight: 700;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.5);
}

.subtitle {
  color: #ccc;
  margin: 0;
  font-size: 16px;
}

.auth-tabs {
  margin-bottom: 30px;
  width: 100%;
}

.auth-tabs-content {
  width: 100%;
}

.auth-form {
  margin-top: 20px;
  width: 100%;
}

.auth-form .el-form-item {
  margin-bottom: 20px;
  width: 100%;
}

.auth-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 8px;
  background-color: #409eff !important;
  border-color: #409eff !important;
}

.auth-btn:hover {
  background-color: #66b1ff !important;
  border-color: #66b1ff !important;
}

.auth-footer {
  text-align: center;
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px solid #333;
  width: 100%;
}

.version {
  color: #888;
  font-size: 14px;
  margin: 0 0 8px 0;
}

.copyright {
  color: #666;
  font-size: 12px;
  margin: 0;
}

:deep(.el-tabs__nav-wrap::after) {
  display: none;
}

:deep(.el-tabs__item) {
  font-size: 16px;
  font-weight: 500;
  padding: 0 24px;
  color: #888 !important;
}

:deep(.el-tabs__item.is-active) {
  color: #409eff !important;
  font-weight: 600;
}

:deep(.el-tabs__item:hover) {
  color: #409eff !important;
}

:deep(.el-input__wrapper) {
  border-radius: 8px;
  height: 48px;
  background-color: #1a1a1a !important;
  border: 1px solid #333 !important;
}

:deep(.el-input__wrapper:hover) {
  border-color: #409eff !important;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #409eff !important;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2) !important;
}

:deep(.el-input__inner) {
  font-size: 16px;
  color: #fff !important;
  background-color: transparent !important;
}

:deep(.el-input__inner::placeholder) {
  color: #666 !important;
}

:deep(.el-checkbox__label) {
  font-size: 14px;
  color: #ccc !important;
}

:deep(.el-checkbox__inner) {
  background-color: #1a1a1a !important;
  border-color: #333 !important;
}

:deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #409eff !important;
  border-color: #409eff !important;
}
</style>
