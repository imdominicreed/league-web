import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit'
import { authApi } from '@/api/auth'
import { User } from '@/types'

// Fix: Use rejectWithValue to properly propagate error messages to Redux state

interface AuthState {
  user: User | null
  accessToken: string | null
  isAuthenticated: boolean
  loading: boolean
  error: string | null
}

const initialState: AuthState = {
  user: null,
  accessToken: localStorage.getItem('accessToken'),
  isAuthenticated: !!localStorage.getItem('accessToken'),
  loading: false,
  error: null,
}

export const login = createAsyncThunk(
  'auth/login',
  async ({ displayName, password }: { displayName: string; password: string }, { rejectWithValue }) => {
    try {
      const response = await authApi.login(displayName, password)
      localStorage.setItem('accessToken', response.accessToken)
      localStorage.setItem('refreshToken', response.refreshToken)
      return response
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Login failed')
    }
  }
)

export const register = createAsyncThunk(
  'auth/register',
  async ({ displayName, password }: { displayName: string; password: string }, { rejectWithValue }) => {
    try {
      const response = await authApi.register(displayName, password)
      localStorage.setItem('accessToken', response.accessToken)
      localStorage.setItem('refreshToken', response.refreshToken)
      return response
    } catch (error) {
      return rejectWithValue(error instanceof Error ? error.message : 'Registration failed')
    }
  }
)

export const fetchCurrentUser = createAsyncThunk(
  'auth/fetchCurrentUser',
  async () => {
    return await authApi.me()
  }
)

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    logout: (state) => {
      state.user = null
      state.accessToken = null
      state.isAuthenticated = false
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')
    },
    setCredentials: (state, action: PayloadAction<{ user: User; accessToken: string }>) => {
      state.user = action.payload.user
      state.accessToken = action.payload.accessToken
      state.isAuthenticated = true
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(login.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(login.fulfilled, (state, action) => {
        state.loading = false
        state.user = action.payload.user
        state.accessToken = action.payload.accessToken
        state.isAuthenticated = true
      })
      .addCase(login.rejected, (state, action) => {
        state.loading = false
        state.error = (action.payload as string) || 'Login failed'
      })
      .addCase(register.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(register.fulfilled, (state, action) => {
        state.loading = false
        state.user = action.payload.user
        state.accessToken = action.payload.accessToken
        state.isAuthenticated = true
      })
      .addCase(register.rejected, (state, action) => {
        state.loading = false
        state.error = (action.payload as string) || 'Registration failed'
      })
      .addCase(fetchCurrentUser.fulfilled, (state, action) => {
        state.user = action.payload
      })
  },
})

export const { logout, setCredentials } = authSlice.actions
export default authSlice.reducer
