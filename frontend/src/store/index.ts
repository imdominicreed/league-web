import { configureStore } from '@reduxjs/toolkit'
import authReducer from './slices/authSlice'
import draftReducer from './slices/draftSlice'
import championsReducer from './slices/championsSlice'
import roomReducer from './slices/roomSlice'
import profileReducer from './slices/profileSlice'
import lobbyReducer from './slices/lobbySlice'
import pendingActionsReducer from './slices/pendingActionsSlice'

export const store = configureStore({
  reducer: {
    auth: authReducer,
    draft: draftReducer,
    champions: championsReducer,
    room: roomReducer,
    profile: profileReducer,
    lobby: lobbyReducer,
    pendingActions: pendingActionsReducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
