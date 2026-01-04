import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit'
import { profileApi } from '@/api/profile'
import { RoleProfile, Role, LeagueRank, User } from '@/types'

interface ProfileState {
  user: User | null
  roleProfiles: RoleProfile[]
  loading: boolean
  error: string | null
  updating: Role | null
}

const initialState: ProfileState = {
  user: null,
  roleProfiles: [],
  loading: false,
  error: null,
  updating: null,
}

export const fetchProfile = createAsyncThunk(
  'profile/fetchProfile',
  async () => {
    return await profileApi.getProfile()
  }
)

export const fetchRoleProfiles = createAsyncThunk(
  'profile/fetchRoleProfiles',
  async () => {
    return await profileApi.getRoleProfiles()
  }
)

export const updateRoleProfile = createAsyncThunk(
  'profile/updateRoleProfile',
  async ({ role, data }: { role: Role; data: { leagueRank?: LeagueRank; mmr?: number; comfortRating?: number } }) => {
    return await profileApi.updateRoleProfile(role, data)
  }
)

export const initializeProfiles = createAsyncThunk(
  'profile/initializeProfiles',
  async () => {
    return await profileApi.initializeProfiles()
  }
)

const profileSlice = createSlice({
  name: 'profile',
  initialState,
  reducers: {
    clearProfileError: (state) => {
      state.error = null
    },
    setRoleProfile: (state, action: PayloadAction<RoleProfile>) => {
      const index = state.roleProfiles.findIndex(p => p.role === action.payload.role)
      if (index >= 0) {
        state.roleProfiles[index] = action.payload
      } else {
        state.roleProfiles.push(action.payload)
      }
    },
  },
  extraReducers: (builder) => {
    builder
      // Fetch profile
      .addCase(fetchProfile.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(fetchProfile.fulfilled, (state, action) => {
        state.loading = false
        state.user = action.payload.user
        state.roleProfiles = action.payload.roleProfiles
      })
      .addCase(fetchProfile.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || 'Failed to fetch profile'
      })
      // Fetch role profiles
      .addCase(fetchRoleProfiles.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(fetchRoleProfiles.fulfilled, (state, action) => {
        state.loading = false
        state.roleProfiles = action.payload
      })
      .addCase(fetchRoleProfiles.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || 'Failed to fetch role profiles'
      })
      // Update role profile
      .addCase(updateRoleProfile.pending, (state, action) => {
        state.updating = action.meta.arg.role
        state.error = null
      })
      .addCase(updateRoleProfile.fulfilled, (state, action) => {
        state.updating = null
        const index = state.roleProfiles.findIndex(p => p.role === action.payload.role)
        if (index >= 0) {
          state.roleProfiles[index] = action.payload
        } else {
          state.roleProfiles.push(action.payload)
        }
      })
      .addCase(updateRoleProfile.rejected, (state, action) => {
        state.updating = null
        state.error = action.error.message || 'Failed to update role profile'
      })
      // Initialize profiles
      .addCase(initializeProfiles.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(initializeProfiles.fulfilled, (state, action) => {
        state.loading = false
        state.roleProfiles = action.payload
      })
      .addCase(initializeProfiles.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || 'Failed to initialize profiles'
      })
  },
})

export const { clearProfileError, setRoleProfile } = profileSlice.actions
export default profileSlice.reducer
