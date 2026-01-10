import { createSlice, PayloadAction } from '@reduxjs/toolkit'
import { TeamPlayer } from '@/types'

interface EditingSlot {
  slotType: 'ban' | 'pick'
  team: 'blue' | 'red'
  slotIndex: number
}

interface PendingEdit {
  proposedBy: string
  proposedSide: 'blue' | 'red'
  slotType: 'ban' | 'pick'
  team: 'blue' | 'red'
  slotIndex: number
  oldChampionId: string
  newChampionId: string
  expiresAt: number
}

interface DraftSliceState {
  currentPhase: number
  currentTeam: 'blue' | 'red' | null
  actionType: 'ban' | 'pick' | null
  timerRemainingMs: number
  blueBans: string[]
  redBans: string[]
  bluePicks: string[]
  redPicks: string[]
  isComplete: boolean
  hoveredChampion: {
    blue: string | null
    red: string | null
  }
  yourSide: 'blue' | 'red' | 'spectator' | null
  fearlessBans: string[]
  teamPlayers: TeamPlayer[]
  isTeamDraft: boolean
  isBufferPeriod: boolean
  // Pause state
  isPaused: boolean
  pausedBy: string | null
  pausedBySide: 'blue' | 'red' | null
  // Edit state
  editingSlot: EditingSlot | null
  pendingEdit: PendingEdit | null
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
  teamPlayers: [],
  isTeamDraft: false,
  isBufferPeriod: false,
  // Pause state
  isPaused: false,
  pausedBy: null,
  pausedBySide: null,
  // Edit state
  editingSlot: null,
  pendingEdit: null,
}

const draftSlice = createSlice({
  name: 'draft',
  initialState,
  reducers: {
    syncState: (state, action: PayloadAction<{
      currentPhase: number
      currentTeam: 'blue' | 'red' | null
      actionType: 'ban' | 'pick' | null
      timerRemainingMs: number
      blueBans: string[]
      redBans: string[]
      bluePicks: string[]
      redPicks: string[]
      isComplete: boolean
      yourSide: string
      fearlessBans?: string[]
      teamPlayers?: TeamPlayer[]
      isTeamDraft?: boolean
      isPaused?: boolean
      pausedBy?: string | null
      pausedBySide?: string | null
      pendingEdit?: PendingEdit | null
    }>) => {
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
      state.teamPlayers = action.payload.teamPlayers || []
      state.isTeamDraft = action.payload.isTeamDraft || false
      state.isPaused = action.payload.isPaused || false
      state.pausedBy = action.payload.pausedBy || null
      state.pausedBySide = (action.payload.pausedBySide as 'blue' | 'red') || null
      state.pendingEdit = action.payload.pendingEdit || null
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
      state.isBufferPeriod = false
    },
    updateTimer: (state, action: PayloadAction<{ remainingMs: number; isBufferPeriod?: boolean }>) => {
      state.timerRemainingMs = action.payload.remainingMs
      state.isBufferPeriod = action.payload.isBufferPeriod ?? false
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

    // Pause reducers
    draftPaused: (state, action: PayloadAction<{
      pausedBy: string
      pausedBySide: 'blue' | 'red'
      timerFrozenAt: number
    }>) => {
      state.isPaused = true
      state.pausedBy = action.payload.pausedBy
      state.pausedBySide = action.payload.pausedBySide
      state.timerRemainingMs = action.payload.timerFrozenAt
      state.isBufferPeriod = false
    },

    draftResumed: (state, action: PayloadAction<{
      timerRemainingMs: number
    }>) => {
      state.isPaused = false
      state.pausedBy = null
      state.pausedBySide = null
      state.timerRemainingMs = action.payload.timerRemainingMs
      state.editingSlot = null
      state.pendingEdit = null
    },

    // Edit reducers
    setEditingSlot: (state, action: PayloadAction<EditingSlot>) => {
      state.editingSlot = action.payload
    },

    clearEditingSlot: (state) => {
      state.editingSlot = null
    },

    editProposed: (state, action: PayloadAction<PendingEdit>) => {
      state.pendingEdit = action.payload
      state.editingSlot = null
    },

    editApplied: (state, action: PayloadAction<{
      slotType: 'ban' | 'pick'
      team: 'blue' | 'red'
      slotIndex: number
      newChampionId: string
      blueBans: string[]
      redBans: string[]
      bluePicks: string[]
      redPicks: string[]
    }>) => {
      state.blueBans = action.payload.blueBans
      state.redBans = action.payload.redBans
      state.bluePicks = action.payload.bluePicks
      state.redPicks = action.payload.redPicks
      state.pendingEdit = null
    },

    editRejected: (state) => {
      state.pendingEdit = null
    },
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
  draftPaused,
  draftResumed,
  setEditingSlot,
  clearEditingSlot,
  editProposed,
  editApplied,
  editRejected,
} = draftSlice.actions

export default draftSlice.reducer
