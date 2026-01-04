import { configureStore } from '@reduxjs/toolkit'
import authReducer from './slices/authSlice'
import draftReducer from './slices/draftSlice'
import championsReducer from './slices/championsSlice'
import roomReducer from './slices/roomSlice'

export const store = configureStore({
  reducer: {
    auth: authReducer,
    draft: draftReducer,
    champions: championsReducer,
    room: roomReducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
