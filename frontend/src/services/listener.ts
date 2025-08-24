import { authService } from './auth'

export interface Profile {
  id: string
  name: string
  projectName: string
  host: string
  port: number
  description: string
}

export interface ListenerStatus {
  active: boolean
  profile?: Profile
  address?: string
}

class ListenerService {
  private baseUrl = '/api'

  // Start the C2 listener with the specified profile
  async startListener(profile: Profile): Promise<boolean> {
    try {
      const response = await fetch(`${this.baseUrl}/profile/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...authService.getAuthHeader(),
        },
        body: JSON.stringify({ profile }),
      })

      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Failed to start listener')
      }

      const data = await response.json()
      return data.success
    } catch (error) {
      console.error('Start listener error:', error)
      throw error
    }
  }

  // Get the current listener status
  async getStatus(): Promise<ListenerStatus> {
    try {
      const response = await fetch(`${this.baseUrl}/profile/status`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          ...authService.getAuthHeader(),
        },
      })

      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Failed to get listener status')
      }

      return await response.json()
    } catch (error) {
      console.error('Get status error:', error)
      throw error
    }
  }


}

export const listenerService = new ListenerService()
