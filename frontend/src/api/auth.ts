import { api } from './client'
import { User } from '@/types'

interface AuthResponse {
  user: User
  accessToken: string
  refreshToken: string
}

export const authApi = {
  register: (displayName: string, password: string): Promise<AuthResponse> =>
    api.post('/auth/register', { displayName, password }),

  login: (displayName: string, password: string): Promise<AuthResponse> =>
    api.post('/auth/login', { displayName, password }),

  me: (): Promise<User> =>
    api.get('/auth/me'),

  logout: (): Promise<{ success: boolean }> =>
    api.post('/auth/logout'),
}
