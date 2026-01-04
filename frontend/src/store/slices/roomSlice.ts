import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import { Room, Player } from '@/types'

interface RoomSliceState {
  room: Room | null
  players: {
    blue: Player | null
    red: Player | null
  }
  spectatorCount: number
  connectionStatus: 'disconnected' | 'connecting' | 'connected'
  error: string | null
}

const initialState: RoomSliceState = {
  room: null,
  players: {
    blue: null,
    red: null,
  },
  spectatorCount: 0,
  connectionStatus: 'disconnected',
  error: null,
}

const roomSlice = createSlice({
  name: 'room',
  initialState,
  reducers: {
    setRoom: (state, action: PayloadAction<Room>) => {
      state.room = action.payload
    },
    syncRoom: (state, action: PayloadAction<{ room: Room; players: { blue: Player | null; red: Player | null }; spectatorCount: number }>) => {
      state.room = action.payload.room
      state.players = action.payload.players
      state.spectatorCount = action.payload.spectatorCount
    },
    playerUpdate: (state, action: PayloadAction<{ side: string; player: Player | null; action: string }>) => {
      const { side, player } = action.payload
      if (side === 'blue') {
        state.players.blue = player
      } else if (side === 'red') {
        state.players.red = player
      }
    },
    updateRoomStatus: (state, action: PayloadAction<'waiting' | 'in_progress' | 'completed'>) => {
      if (state.room) {
        state.room.status = action.payload
      }
    },
    setConnectionStatus: (state, action: PayloadAction<'disconnected' | 'connecting' | 'connected'>) => {
      state.connectionStatus = action.payload
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload
    },
    resetRoom: () => initialState,
  },
})

export const {
  setRoom,
  syncRoom,
  playerUpdate,
  updateRoomStatus,
  setConnectionStatus,
  setError,
  resetRoom,
} = roomSlice.actions

export default roomSlice.reducer
