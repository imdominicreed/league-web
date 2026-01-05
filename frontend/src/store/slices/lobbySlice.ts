import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit'
import { lobbyApi } from '@/api/lobby'
import { Lobby, LobbyPlayer, MatchOption, Room, PendingAction, TeamStats } from '@/types'

interface LobbyState {
  lobby: Lobby | null
  matchOptions: MatchOption[] | null
  pendingAction: PendingAction | null
  teamStats: TeamStats | null
  loading: boolean
  error: string | null
  generatingTeams: boolean
  selectingOption: boolean
  startingDraft: boolean
  createdRoom: Room | null
  // Captain action loading states
  takingCaptain: boolean
  promotingCaptain: boolean
  kickingPlayer: boolean
  // Pending action loading states
  proposingAction: boolean
  approvingAction: boolean
  cancellingAction: boolean
  fetchingTeamStats: boolean
}

const initialState: LobbyState = {
  lobby: null,
  matchOptions: null,
  pendingAction: null,
  teamStats: null,
  loading: false,
  error: null,
  generatingTeams: false,
  selectingOption: false,
  startingDraft: false,
  createdRoom: null,
  takingCaptain: false,
  promotingCaptain: false,
  kickingPlayer: false,
  proposingAction: false,
  approvingAction: false,
  cancellingAction: false,
  fetchingTeamStats: false,
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

// Captain management thunks
export const takeCaptain = createAsyncThunk(
  'lobby/takeCaptain',
  async (lobbyId: string) => {
    return await lobbyApi.takeCaptain(lobbyId)
  }
)

export const promoteCaptain = createAsyncThunk(
  'lobby/promoteCaptain',
  async ({ lobbyId, userId }: { lobbyId: string; userId: string }) => {
    return await lobbyApi.promoteCaptain(lobbyId, userId)
  }
)

export const kickPlayer = createAsyncThunk(
  'lobby/kickPlayer',
  async ({ lobbyId, userId }: { lobbyId: string; userId: string }) => {
    return await lobbyApi.kickPlayer(lobbyId, userId)
  }
)

// Pending action thunks
export const proposeSwap = createAsyncThunk(
  'lobby/proposeSwap',
  async ({ lobbyId, player1Id, player2Id, swapType }: {
    lobbyId: string
    player1Id: string
    player2Id: string
    swapType: 'players' | 'roles'
  }) => {
    return await lobbyApi.proposeSwap(lobbyId, { player1Id, player2Id, swapType })
  }
)

export const proposeMatchmake = createAsyncThunk(
  'lobby/proposeMatchmake',
  async (lobbyId: string) => {
    return await lobbyApi.proposeMatchmake(lobbyId)
  }
)

export const proposeStartDraft = createAsyncThunk(
  'lobby/proposeStartDraft',
  async (lobbyId: string) => {
    return await lobbyApi.proposeStartDraft(lobbyId)
  }
)

export const fetchPendingAction = createAsyncThunk(
  'lobby/fetchPendingAction',
  async (lobbyId: string) => {
    return await lobbyApi.getPendingAction(lobbyId)
  }
)

export const approvePendingAction = createAsyncThunk(
  'lobby/approvePendingAction',
  async ({ lobbyId, actionId }: { lobbyId: string; actionId: string }) => {
    return await lobbyApi.approvePendingAction(lobbyId, actionId)
  }
)

export const cancelPendingAction = createAsyncThunk(
  'lobby/cancelPendingAction',
  async ({ lobbyId, actionId }: { lobbyId: string; actionId: string }) => {
    return await lobbyApi.cancelPendingAction(lobbyId, actionId)
  }
)

// Team stats thunk
export const fetchTeamStats = createAsyncThunk(
  'lobby/fetchTeamStats',
  async (lobbyId: string) => {
    return await lobbyApi.getTeamStats(lobbyId)
  }
)

const lobbySlice = createSlice({
  name: 'lobby',
  initialState,
  reducers: {
    clearLobby: (state) => {
      state.lobby = null
      state.matchOptions = null
      state.pendingAction = null
      state.teamStats = null
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
    setPendingAction: (state, action: PayloadAction<PendingAction | null>) => {
      state.pendingAction = action.payload
    },
    setTeamStats: (state, action: PayloadAction<TeamStats | null>) => {
      state.teamStats = action.payload
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
      // Take captain
      .addCase(takeCaptain.pending, (state) => {
        state.takingCaptain = true
        state.error = null
      })
      .addCase(takeCaptain.fulfilled, (state, action) => {
        state.takingCaptain = false
        state.lobby = action.payload
      })
      .addCase(takeCaptain.rejected, (state, action) => {
        state.takingCaptain = false
        state.error = action.error.message || 'Failed to take captain'
      })
      // Promote captain
      .addCase(promoteCaptain.pending, (state) => {
        state.promotingCaptain = true
        state.error = null
      })
      .addCase(promoteCaptain.fulfilled, (state, action) => {
        state.promotingCaptain = false
        state.lobby = action.payload
      })
      .addCase(promoteCaptain.rejected, (state, action) => {
        state.promotingCaptain = false
        state.error = action.error.message || 'Failed to promote captain'
      })
      // Kick player
      .addCase(kickPlayer.pending, (state) => {
        state.kickingPlayer = true
        state.error = null
      })
      .addCase(kickPlayer.fulfilled, (state, action) => {
        state.kickingPlayer = false
        state.lobby = action.payload
      })
      .addCase(kickPlayer.rejected, (state, action) => {
        state.kickingPlayer = false
        state.error = action.error.message || 'Failed to kick player'
      })
      // Propose swap
      .addCase(proposeSwap.pending, (state) => {
        state.proposingAction = true
        state.error = null
      })
      .addCase(proposeSwap.fulfilled, (state, action) => {
        state.proposingAction = false
        state.pendingAction = action.payload
      })
      .addCase(proposeSwap.rejected, (state, action) => {
        state.proposingAction = false
        state.error = action.error.message || 'Failed to propose swap'
      })
      // Propose matchmake
      .addCase(proposeMatchmake.pending, (state) => {
        state.proposingAction = true
        state.error = null
      })
      .addCase(proposeMatchmake.fulfilled, (state, action) => {
        state.proposingAction = false
        state.pendingAction = action.payload
      })
      .addCase(proposeMatchmake.rejected, (state, action) => {
        state.proposingAction = false
        state.error = action.error.message || 'Failed to propose matchmake'
      })
      // Propose start draft
      .addCase(proposeStartDraft.pending, (state) => {
        state.proposingAction = true
        state.error = null
      })
      .addCase(proposeStartDraft.fulfilled, (state, action) => {
        state.proposingAction = false
        state.pendingAction = action.payload
      })
      .addCase(proposeStartDraft.rejected, (state, action) => {
        state.proposingAction = false
        state.error = action.error.message || 'Failed to propose start draft'
      })
      // Fetch pending action
      .addCase(fetchPendingAction.fulfilled, (state, action) => {
        state.pendingAction = action.payload
      })
      // Approve pending action
      .addCase(approvePendingAction.pending, (state) => {
        state.approvingAction = true
        state.error = null
      })
      .addCase(approvePendingAction.fulfilled, (state, action) => {
        state.approvingAction = false
        state.lobby = action.payload
        state.pendingAction = null
      })
      .addCase(approvePendingAction.rejected, (state, action) => {
        state.approvingAction = false
        state.error = action.error.message || 'Failed to approve action'
      })
      // Cancel pending action
      .addCase(cancelPendingAction.pending, (state) => {
        state.cancellingAction = true
        state.error = null
      })
      .addCase(cancelPendingAction.fulfilled, (state, action) => {
        state.cancellingAction = false
        state.lobby = action.payload
        state.pendingAction = null
      })
      .addCase(cancelPendingAction.rejected, (state, action) => {
        state.cancellingAction = false
        state.error = action.error.message || 'Failed to cancel action'
      })
      // Fetch team stats
      .addCase(fetchTeamStats.pending, (state) => {
        state.fetchingTeamStats = true
      })
      .addCase(fetchTeamStats.fulfilled, (state, action) => {
        state.fetchingTeamStats = false
        state.teamStats = action.payload
      })
      .addCase(fetchTeamStats.rejected, (state) => {
        state.fetchingTeamStats = false
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
  setPendingAction,
  setTeamStats,
} = lobbySlice.actions
export default lobbySlice.reducer
