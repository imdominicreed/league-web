import { createSlice, createAsyncThunk } from '@reduxjs/toolkit'
import { pendingActionsApi, LobbyPendingAction, DraftPendingAction } from '@/api/pending-actions'

interface PendingActionsState {
  lobbyActions: LobbyPendingAction[]
  draftActions: DraftPendingAction[]
  loading: boolean
  error: string | null
  lastFetched: number | null
}

const initialState: PendingActionsState = {
  lobbyActions: [],
  draftActions: [],
  loading: false,
  error: null,
  lastFetched: null,
}

export const fetchPendingActions = createAsyncThunk(
  'pendingActions/fetchAll',
  async () => {
    return await pendingActionsApi.getAll()
  }
)

const pendingActionsSlice = createSlice({
  name: 'pendingActions',
  initialState,
  reducers: {
    clearPendingActions: (state) => {
      state.lobbyActions = []
      state.draftActions = []
      state.error = null
      state.lastFetched = null
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchPendingActions.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(fetchPendingActions.fulfilled, (state, action) => {
        state.loading = false
        state.lobbyActions = action.payload.lobbyActions
        state.draftActions = action.payload.draftActions
        state.lastFetched = Date.now()
      })
      .addCase(fetchPendingActions.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || 'Failed to fetch pending actions'
      })
  },
})

export const { clearPendingActions } = pendingActionsSlice.actions
export default pendingActionsSlice.reducer
