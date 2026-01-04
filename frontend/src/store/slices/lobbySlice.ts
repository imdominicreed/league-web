import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit'
import { lobbyApi } from '@/api/lobby'
import { Lobby, LobbyPlayer, MatchOption, Room } from '@/types'

interface LobbyState {
  lobby: Lobby | null
  matchOptions: MatchOption[] | null
  loading: boolean
  error: string | null
  generatingTeams: boolean
  selectingOption: boolean
  startingDraft: boolean
  createdRoom: Room | null
}

const initialState: LobbyState = {
  lobby: null,
  matchOptions: null,
  loading: false,
  error: null,
  generatingTeams: false,
  selectingOption: false,
  startingDraft: false,
  createdRoom: null,
}

export const createLobby = createAsyncThunk(
  'lobby/create',
  async (data: { draftMode: 'pro_play' | 'fearless'; timerDurationSeconds?: number }) => {
    return await lobbyApi.create(data)
  }
)

export const fetchLobby = createAsyncThunk(
  'lobby/fetch',
  async (idOrCode: string) => {
    return await lobbyApi.get(idOrCode)
  }
)

export const joinLobby = createAsyncThunk(
  'lobby/join',
  async (idOrCode: string) => {
    return await lobbyApi.join(idOrCode)
  }
)

export const leaveLobby = createAsyncThunk(
  'lobby/leave',
  async (idOrCode: string) => {
    await lobbyApi.leave(idOrCode)
    return idOrCode
  }
)

export const setReady = createAsyncThunk(
  'lobby/setReady',
  async ({ idOrCode, ready }: { idOrCode: string; ready: boolean }) => {
    return await lobbyApi.setReady(idOrCode, ready)
  }
)

export const generateTeams = createAsyncThunk(
  'lobby/generateTeams',
  async (lobbyId: string) => {
    return await lobbyApi.generateTeams(lobbyId)
  }
)

export const fetchMatchOptions = createAsyncThunk(
  'lobby/fetchMatchOptions',
  async (lobbyId: string) => {
    return await lobbyApi.getMatchOptions(lobbyId)
  }
)

export const selectMatchOption = createAsyncThunk(
  'lobby/selectOption',
  async ({ lobbyId, optionNumber }: { lobbyId: string; optionNumber: number }) => {
    return await lobbyApi.selectOption(lobbyId, optionNumber)
  }
)

export const startDraft = createAsyncThunk(
  'lobby/startDraft',
  async (lobbyId: string, { rejectWithValue }) => {
    try {
      const room = await lobbyApi.startDraft(lobbyId)
      return room
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } }
      return rejectWithValue(err.response?.data?.error || 'Failed to start draft')
    }
  }
)

const lobbySlice = createSlice({
  name: 'lobby',
  initialState,
  reducers: {
    clearLobby: (state) => {
      state.lobby = null
      state.matchOptions = null
      state.error = null
    },
    clearLobbyError: (state) => {
      state.error = null
    },
    updatePlayer: (state, action: PayloadAction<LobbyPlayer>) => {
      if (state.lobby) {
        const index = state.lobby.players.findIndex(p => p.userId === action.payload.userId)
        if (index >= 0) {
          state.lobby.players[index] = action.payload
        } else {
          state.lobby.players.push(action.payload)
        }
      }
    },
    removePlayer: (state, action: PayloadAction<string>) => {
      if (state.lobby) {
        state.lobby.players = state.lobby.players.filter(p => p.userId !== action.payload)
      }
    },
    setLobby: (state, action: PayloadAction<Lobby>) => {
      state.lobby = action.payload
    },
    setMatchOptions: (state, action: PayloadAction<MatchOption[]>) => {
      state.matchOptions = action.payload
    },
  },
  extraReducers: (builder) => {
    builder
      // Create lobby
      .addCase(createLobby.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(createLobby.fulfilled, (state, action) => {
        state.loading = false
        state.lobby = action.payload
      })
      .addCase(createLobby.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || 'Failed to create lobby'
      })
      // Fetch lobby
      .addCase(fetchLobby.pending, (state) => {
        state.loading = true
        state.error = null
      })
      .addCase(fetchLobby.fulfilled, (state, action) => {
        state.loading = false
        state.lobby = action.payload
      })
      .addCase(fetchLobby.rejected, (state, action) => {
        state.loading = false
        state.error = action.error.message || 'Failed to fetch lobby'
      })
      // Join lobby
      .addCase(joinLobby.fulfilled, (state, action) => {
        if (state.lobby) {
          const existing = state.lobby.players.find(p => p.userId === action.payload.userId)
          if (!existing) {
            state.lobby.players.push(action.payload)
          }
        }
      })
      // Leave lobby
      .addCase(leaveLobby.fulfilled, (state) => {
        state.lobby = null
        state.matchOptions = null
      })
      // Set ready - state updated via polling or WebSocket
      .addCase(setReady.fulfilled, () => {
        // No-op: state will be updated via polling/WebSocket
      })
      // Generate teams
      .addCase(generateTeams.pending, (state) => {
        state.generatingTeams = true
        state.error = null
      })
      .addCase(generateTeams.fulfilled, (state, action) => {
        state.generatingTeams = false
        state.matchOptions = action.payload
        if (state.lobby) {
          state.lobby.status = 'matchmaking'
        }
      })
      .addCase(generateTeams.rejected, (state, action) => {
        state.generatingTeams = false
        state.error = action.error.message || 'Failed to generate teams'
      })
      // Fetch match options
      .addCase(fetchMatchOptions.fulfilled, (state, action) => {
        state.matchOptions = action.payload
      })
      // Select option
      .addCase(selectMatchOption.pending, (state) => {
        state.selectingOption = true
        state.error = null
      })
      .addCase(selectMatchOption.fulfilled, (state, action) => {
        state.selectingOption = false
        state.lobby = action.payload
      })
      .addCase(selectMatchOption.rejected, (state, action) => {
        state.selectingOption = false
        state.error = action.error.message || 'Failed to select option'
      })
      // Start draft
      .addCase(startDraft.pending, (state) => {
        state.startingDraft = true
        state.error = null
        state.createdRoom = null
      })
      .addCase(startDraft.fulfilled, (state, action) => {
        state.startingDraft = false
        state.createdRoom = action.payload
        if (state.lobby) {
          state.lobby.status = 'drafting'
          state.lobby.roomId = action.payload.id
        }
      })
      .addCase(startDraft.rejected, (state, action) => {
        state.startingDraft = false
        state.error = action.payload as string || 'Failed to start draft'
      })
  },
})

export const {
  clearLobby,
  clearLobbyError,
  updatePlayer,
  removePlayer,
  setLobby,
  setMatchOptions,
} = lobbySlice.actions
export default lobbySlice.reducer
