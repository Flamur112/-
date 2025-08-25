import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'

export interface User {
  id: number
  username: string
  email?: string
  role: string
  is_active: boolean
  created_at: string
  last_login?: string
}

export interface LoginCredentials {
  username: string
  password: string
}

export interface RegisterData {
  username: string
  password: string
}

export interface AuthResponse {
  token: string
  user: User
}

class AuthService {
  private token = ref<string | null>(null) // Don't auto-load from localStorage
  private user = ref<User | null>(null)
  private backendAvailable = ref(true) // Global flag to track backend availability

  // Computed properties
  public isAuthenticated = computed(() => !!this.token.value)
  public currentUser = computed(() => this.user.value)
  public userRole = computed(() => this.user.value?.role || '')

  constructor() {
    // Don't auto-restore user from token - require fresh login each time
    // This ensures security by requiring explicit authentication on each server start
    this.clearAuth() // Clear any existing tokens on service initialization
  }

  // Set authentication token and user
  private setAuth(token: string, user: User) {
    this.token.value = token
    this.user.value = user
    localStorage.setItem('auth_token', token)
    localStorage.setItem('user_data', JSON.stringify(user))
  }

  // Clear authentication data
  public clearAuth() {
    this.token.value = null
    this.user.value = null
    localStorage.removeItem('auth_token')
    localStorage.removeItem('user_data')
    localStorage.removeItem('active_profile_id')
  }

  // Get authorization header for API requests
  public getAuthHeader(): { Authorization: string } | {} {
    return this.token.value ? { Authorization: `Bearer ${this.token.value}` } : {}
  }

  // Login user
  public async login(credentials: LoginCredentials): Promise<boolean> {
    try {
      // Check if backend is available before attempting login
      if (!await this.checkBackendHealth()) {
        throw new Error('Backend server is not available. Please restart the server using run-mulic2.bat')
      }

      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      })

      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Login failed')
      }

      const data: AuthResponse = await response.json()
      this.setAuth(data.token, data.user)
      return true
    } catch (error) {
      console.error('Login error:', error)
      throw error
    }
  }

  // Register new user
  public async register(userData: RegisterData): Promise<boolean> {
    try {
      // Check if backend is available before attempting registration
      if (!await this.checkBackendHealth()) {
        throw new Error('Backend server is not available. Please restart the server using run-mulic2.bat')
      }

      const response = await fetch('/api/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(userData),
      })

      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Registration failed')
      }

      const data = await response.json()
      console.log('Registration successful:', data.message)
      return true
    } catch (error) {
      console.error('Registration error:', error)
      throw error
    }
  }

  // Logout user
  public async logout(): Promise<void> {
    try {
      if (this.token.value && this.backendAvailable.value) {
        await fetch('/api/auth/logout', {
          method: 'POST',
          headers: {
            ...this.getAuthHeader(),
            'Content-Type': 'application/json',
          },
        })
      }
    } catch (error) {
      console.error('Logout error:', error)
    } finally {
      this.clearAuth()
    }
  }

  // Load user profile from API
  public async loadUserProfile(): Promise<void> {
    if (!this.token.value) return
    
    // Prevent API calls if backend is down
    if (!this.backendAvailable.value) {
      console.warn('Backend is unavailable, skipping profile load')
      return
    }

    // Try to load profile with retry logic
    for (let attempt = 1; attempt <= 3; attempt++) {
      try {
        const response = await fetch('/api/auth/profile', {
          headers: this.getAuthHeader(),
          // Add timeout to prevent hanging requests
          signal: AbortSignal.timeout(3000) // 3 second timeout
        })

        if (response.ok) {
          const userData: User = await response.json()
          this.user.value = userData
          localStorage.setItem('user_data', JSON.stringify(userData))
          return // Success, exit retry loop
        } else if (response.status === 401) {
          // Unauthorized, clear auth immediately
          this.clearAuth()
          return
        } else {
          // Other HTTP errors, might retry
          console.warn(`Profile load attempt ${attempt} failed with status ${response.status}`)
        }
      } catch (error) {
        console.warn(`Profile load attempt ${attempt} failed:`, error)
        
        // If it's a timeout or network error, retry
        if (error instanceof TypeError || (error instanceof Error && error.name === 'AbortError')) {
          if (attempt < 3) {
            // Wait a bit before retrying
            await new Promise(resolve => setTimeout(resolve, 1000 * attempt))
            continue
          }
        }
        
        // If all retries failed, clear auth state to prevent conflicts
        console.warn('Backend appears to be unavailable after all retries, clearing auth state to prevent conflicts')
        this.clearAuth()
        return
      }
    }
    
    // If we get here, all retries failed
    console.warn('All profile load attempts failed, clearing auth state to prevent conflicts')
    this.clearAuth()
  }

     // Check if user is admin
   public isAdmin(): boolean {
     return this.user.value?.role === 'admin'
   }

  // Refresh token (if needed)
  public async refreshToken(): Promise<boolean> {
    try {
      // Prevent API calls if backend is down
      if (!this.backendAvailable.value) {
        console.warn('Backend is unavailable, skipping token refresh')
        return false
      }

      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: this.getAuthHeader(),
      })

      if (response.ok) {
        const data: AuthResponse = await response.json()
        this.setAuth(data.token, data.user)
        return true
      }
      return false
    } catch (error) {
      console.error('Token refresh failed:', error)
      return false
      }
  }

  // Initialize auth service (call this in main.ts)
  public async init(): Promise<void> {
    if (this.token.value) {
      try {
        // Small delay to give backend time to be ready
        await new Promise(resolve => setTimeout(resolve, 500))
        await this.loadUserProfile()
      } catch (error) {
        console.warn('Failed to initialize auth service, clearing auth state to prevent conflicts:', error)
        // Clear auth state if backend is unavailable to prevent conflicts
        this.clearAuth()
      }
    }
  }

  // Check if backend is available
  public async checkBackendHealth(): Promise<boolean> {
    try {
      const response = await fetch('/api/health', {
        method: 'GET',
        signal: AbortSignal.timeout(2000) // 2 second timeout
      })
      const isHealthy = response.ok
      this.backendAvailable.value = isHealthy
      
      if (!isHealthy) {
        // If backend is down, immediately clear auth state to prevent further API calls
        this.clearAuth()
      }
      
      return isHealthy
    } catch (error) {
      console.log('Backend health check failed:', error)
      this.backendAvailable.value = false
      // If backend is down, immediately clear auth state to prevent further API calls
      this.clearAuth()
      return false
    }
  }

  // Check if backend is available (synchronous check)
  public isBackendAvailable(): boolean {
    return this.backendAvailable.value
  }

  // Start periodic health check
  public startHealthCheck(): void {
    // Check backend health every 10 seconds for faster response
    setInterval(async () => {
      if (this.token.value && this.user.value) {
        const isHealthy = await this.checkBackendHealth()
        if (!isHealthy) {
          console.warn('Backend health check failed, clearing auth state to prevent conflicts')
          this.clearAuth()
        }
      }
    }, 10000) // 10 seconds for faster detection
  }
}

// Create singleton instance
export const authService = new AuthService()

// Export composable for Vue components
export function useAuth() {
  return {
    // State
    isAuthenticated: authService.isAuthenticated,
    currentUser: authService.currentUser,
    userRole: authService.userRole,

    // Methods
    login: authService.login.bind(authService),
    register: authService.register.bind(authService),
    logout: authService.logout.bind(authService),
    isAdmin: authService.isAdmin.bind(authService),
    getAuthHeader: authService.getAuthHeader.bind(authService),
    checkBackendHealth: authService.checkBackendHealth.bind(authService),
    isBackendAvailable: authService.isBackendAvailable.bind(authService),
  }
}
