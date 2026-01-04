import { api } from './client'
import { UserProfile, RoleProfile, Role, LeagueRank } from '@/types'

interface UpdateRoleProfileRequest {
  leagueRank?: LeagueRank
  mmr?: number
  comfortRating?: number
}

export const profileApi = {
  getProfile: (): Promise<UserProfile> =>
    api.get('/profile'),

  getRoleProfiles: (): Promise<RoleProfile[]> =>
    api.get('/profile/roles'),

  updateRoleProfile: (role: Role, data: UpdateRoleProfileRequest): Promise<RoleProfile> =>
    api.put(`/profile/roles/${role}`, data),

  initializeProfiles: (): Promise<RoleProfile[]> =>
    api.post('/profile/roles/initialize'),
}
