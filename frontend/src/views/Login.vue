<template>
  <div class="login-container">
    <div class="login-header">
      <div class="logo">
        <img src="/favicon.ico" alt="MulisC2" class="favicon" />
        <h1 class="title">MulisC2</h1>
      </div>
      
      <div class="login-info">
        <p>🔐 <strong>Login to your account</strong></p>
        <p>⚠️  <strong>Note:</strong> Backend server must be running</p>
      </div>
      
      <el-form
        ref="loginFormRef"
        :model="loginForm"
        :rules="loginRules"
        class="login-form"
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
            class="login-btn"
            :loading="loading"
            @click="handleLogin"
          >
            {{ loading ? 'Signing in...' : 'Sign In' }}
          </el-button>
        </el-form-item>
      </el-form>
      
      <div class="login-footer">
        <p class="version">Version 1.0.0</p>
        <p class="copyright">© 2025 MulisC2. All rights reserved.</p>
        <p class="register-link">
          Don't have an account? <router-link to="/register">Register here</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { authService } from '@/services/auth'

const router = useRouter()
const loginFormRef = ref<FormInstance>()
const loading = ref(false)

const loginForm = reactive({
  username: '',
  password: '',
  remember: false
})

const loginRules: FormRules = {
  username: [
    { required: true, message: 'Please enter username', trigger: 'blur' }
  ],
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { min: 6, message: 'Password must be at least 6 characters', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (!loginFormRef.value) return
  
  try {
    await loginFormRef.value.validate()
    loading.value = true
    
    // Use the auth service instead of direct API call
    const success = await authService.login({
      username: loginForm.username,
      password: loginForm.password
    })
    
    if (success) {
      if (loginForm.remember) {
        localStorage.setItem('remember_username', loginForm.username)
      }
      
      ElMessage.success('Login successful')
      router.push('/dashboard')
    }
  } catch (error) {
    console.error('Login error:', error)
    ElMessage.error('Login failed. Please check your credentials and try again.')
  } finally {
    loading.value = false
  }
}

// Auto-fill username if remembered
const rememberedUsername = localStorage.getItem('remember_username')
if (rememberedUsername) {
  loginForm.username = rememberedUsername
  loginForm.remember = true
}
</script>

<style scoped lang="scss">
.login-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #000;
  padding: 20px;
  font-family: 'Courier New', monospace;
}

    .login-header {
      text-align: center;
      margin-bottom: 20px;
  
  .logo {
    margin-bottom: 16px;
    display: flex;
    flex-direction: column;
    align-items: center;
    
    .favicon {
      width: 64px;
      height: 64px;
      margin-bottom: 16px;
    }
    
    .title {
      font-size: 48px;
      font-weight: bold;
      color: #ff0000;
      margin: 0;
      font-family: 'Courier New', monospace;
      letter-spacing: 3px;
    }
  }
  
  .subtitle {
    color: #666;
    font-size: 18px;
    margin: 0;
    font-family: 'Courier New', monospace;
        }
    }
    
    .login-info {
      text-align: center;
      margin-bottom: 20px;
      padding: 15px;
      background: rgba(255, 0, 0, 0.1);
      border: 1px solid rgba(255, 0, 0, 0.3);
      border-radius: 5px;
      
      p {
        margin: 5px 0;
        color: #ff0000;
        font-size: 14px;
        font-family: 'Courier New', monospace;
      }
    }
    
    .login-form {
  width: 100%;
  max-width: 400px;
  margin-bottom: 40px;
  
  .el-form-item {
    margin-bottom: 24px;
  }
  
  .login-btn {
    width: 100%;
    height: 48px;
    font-size: 16px;
    font-weight: bold;
    font-family: 'Courier New', monospace;
    text-transform: uppercase;
    letter-spacing: 1px;
  }
}

.login-footer {
  text-align: center;
  
  .version {
    color: #333;
    font-size: 14px;
    margin: 0 0 8px 0;
    font-family: 'Courier New', monospace;
  }
  
  .copyright {
    color: #333;
    font-size: 12px;
    margin: 0;
    font-family: 'Courier New', monospace;
  }
  
  .register-link {
    color: #333;
    font-size: 14px;
    margin: 8px 0 0 0;
    font-family: 'Courier New', monospace;
    
    a {
      color: #ff0000;
      text-decoration: none;
      
      &:hover {
        text-decoration: underline;
      }
    }
  }
}

// Simple terminal-style inputs
:deep(.el-input__wrapper) {
  background: #111 !important;
  border: 1px solid #333 !important;
  border-radius: 0 !important;
  
  &:focus-within {
    border-color: #ff0000 !important;
  }
}

:deep(.el-input__inner) {
  color: #fff !important;
  font-family: 'Courier New', monospace !important;
  
  &::placeholder {
    color: #666 !important;
  }
}

:deep(.el-checkbox__label) {
  color: #666 !important;
  font-family: 'Courier New', monospace !important;
}

:deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #ff0000 !important;
  border-color: #ff0000 !important;
}
</style>



