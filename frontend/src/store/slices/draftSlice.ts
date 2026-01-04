import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import { DraftState } from '@/types'

interface DraftSliceState extends DraftState {
  hoveredChampion: {
    blue: string | null
    red: string | null
  }
  yourSide: 'blue' | 'red' | 'spectator' | null
  fearlessBans: string[]
}

const initialState: DraftSliceState = {
  currentPhase: 0,
  currentTeam: null,
  actionType: null,
  timerRemainingMs: 30000,
  blueBans: [],
  redBans: [],
  bluePicks: [],
  redPicks: [],
  isComplete: false,
  hoveredChampion: {
    blue: null,
    red: null,
  },
  yourSide: null,
  fearlessBans: [],
}

const draftSlice = createSlice({
  name: 'draft',
  initialState,
  reducers: {
    syncState: (state, action: PayloadAction<DraftState & { yourSide: string; fearlessBans?: string[] }>) => {
      state.currentPhase = action.payload.currentPhase
      state.currentTeam = action.payload.currentTeam
      state.actionType = action.payload.actionType
      state.timerRemainingMs = action.payload.timerRemainingMs
      state.blueBans = action.payload.blueBans
      state.redBans = action.payload.redBans
      state.bluePicks = action.payload.bluePicks
      state.redPicks = action.payload.redPicks
      state.isComplete = action.payload.isComplete
      state.yourSide = action.payload.yourSide as 'blue' | 'red' | 'spectator'
      state.fearlessBans = action.payload.fearlessBans || []
    },
    championSelected: (state, action: PayloadAction<{ phase: number; team: string; actionType: string; championId: string }>) => {
      const { team, actionType, championId } = action.payload
      if (actionType === 'ban') {
        if (team === 'blue') {
          state.blueBans.push(championId)
        } else {
          state.redBans.push(championId)
        }
      } else {
        if (team === 'blue') {
          state.bluePicks.push(championId)
        } else {
          state.redPicks.push(championId)
        }
      }
    },
    phaseChanged: (state, action: PayloadAction<{ currentPhase: number; currentTeam: string; actionType: string; timerRemainingMs: number }>) => {
      state.currentPhase = action.payload.currentPhase
      state.currentTeam = action.payload.currentTeam as 'blue' | 'red'
      state.actionType = action.payload.actionType as 'ban' | 'pick'
      state.timerRemainingMs = action.payload.timerRemainingMs
      state.hoveredChampion = { blue: null, red: null }
    },
    updateTimer: (state, action: PayloadAction<{ remainingMs: number }>) => {
      state.timerRemainingMs = action.payload.remainingMs
    },
    championHovered: (state, action: PayloadAction<{ side: string; championId: string | null }>) => {
      if (action.payload.side === 'blue') {
        state.hoveredChampion.blue = action.payload.championId
      } else {
        state.hoveredChampion.red = action.payload.championId
      }
    },
    draftCompleted: (state, action: PayloadAction<{ blueBans: string[]; redBans: string[]; bluePicks: string[]; redPicks: string[] }>) => {
      state.blueBans = action.payload.blueBans
      state.redBans = action.payload.redBans
      state.bluePicks = action.payload.bluePicks
      state.redPicks = action.payload.redPicks
      state.isComplete = true
    },
    resetDraft: () => initialState,
  },
})

export const {
  syncState,
  championSelected,
  phaseChanged,
  updateTimer,
  championHovered,
  draftCompleted,
  resetDraft,
} = draftSlice.actions

export default draftSlice.reducer
