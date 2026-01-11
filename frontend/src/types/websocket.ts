/**
 * WebSocket Protocol v2 Types
 *
 * New message architecture using Command/Event pattern with discriminated unions.
 * These types coexist with the old types during migration.
 */

// ============================================================================
// Core Message Types
// ============================================================================

export type MsgType = 'COMMAND' | 'QUERY' | 'EVENT' | 'STATE' | 'TIMER' | 'ERR'

export interface Msg<T = unknown> {
  type: MsgType
  payload: T
  timestamp: number
  seq?: number
}

// ============================================================================
// COMMAND - Client → Server actions
// ============================================================================

export type CommandAction =
  | 'join_room'
  | 'select_champion'
  | 'lock_in'
  | 'hover_champion'
  | 'set_ready'
  | 'start_draft'
  | 'pause_draft'
  | 'resume_ready'
  | 'propose_edit'
  | 'respond_edit'

export interface Command<A extends CommandAction = CommandAction, P = unknown> {
  action: A
  payload?: P
}

// Command payloads
export interface CmdJoinRoomPayload {
  roomId: string
  side: string
}

export interface CmdSelectChampionPayload {
  championId: string
}

export interface CmdHoverChampionPayload {
  championId: string | null
}

export interface CmdSetReadyPayload {
  ready: boolean
}

export interface CmdResumeReadyPayload {
  ready: boolean
}

export interface CmdProposeEditPayload {
  slotType: 'ban' | 'pick'
  team: 'blue' | 'red'
  slotIndex: number
  championId: string
}

export interface CmdRespondEditPayload {
  accept: boolean
}

// Typed command union
export type TypedCommand =
  | Command<'join_room', CmdJoinRoomPayload>
  | Command<'select_champion', CmdSelectChampionPayload>
  | Command<'lock_in', undefined>
  | Command<'hover_champion', CmdHoverChampionPayload>
  | Command<'set_ready', CmdSetReadyPayload>
  | Command<'start_draft', undefined>
  | Command<'pause_draft', undefined>
  | Command<'resume_ready', CmdResumeReadyPayload>
  | Command<'propose_edit', CmdProposeEditPayload>
  | Command<'respond_edit', CmdRespondEditPayload>

// ============================================================================
// QUERY - Client → Server state requests
// ============================================================================

export type QueryType = 'sync_state'

export interface Query {
  query: QueryType
}

// ============================================================================
// EVENT - Server → Client state changes
// ============================================================================

export type EventType =
  // Draft lifecycle
  | 'draft_started'
  | 'draft_completed'
  | 'phase_changed'
  // Champion actions
  | 'champion_selected'
  | 'champion_hovered'
  // Player events
  | 'player_joined'
  | 'player_left'
  | 'player_ready_changed'
  // Pause/Resume
  | 'draft_paused'
  | 'draft_resumed'
  | 'resume_ready_changed'
  | 'resume_countdown'
  // Edit workflow
  | 'edit_proposed'
  | 'edit_applied'
  | 'edit_rejected'

export interface Event<E extends EventType = EventType, P = unknown> {
  event: E
  payload: P
}

// Event payloads

export interface PhaseInfo {
  currentPhase: number
  currentTeam: 'blue' | 'red'
  actionType: 'ban' | 'pick'
  timerRemainingMs: number
}

export interface DraftResult {
  blueBans: string[]
  redBans: string[]
  bluePicks: string[]
  redPicks: string[]
}

export interface ChampionSelection {
  phase: number
  team: 'blue' | 'red'
  actionType: 'ban' | 'pick'
  championId: string
}

export interface PlayerData {
  userId: string
  displayName: string
}

export interface EditProposal {
  slotType: 'ban' | 'pick'
  team: 'blue' | 'red'
  slotIndex: number
  oldChampionId: string
  newChampionId: string
}

export interface EvtDraftStartedPayload {
  phase: PhaseInfo
}

export interface EvtDraftCompletedPayload {
  result: DraftResult
}

export interface EvtPhaseChangedPayload {
  phase: PhaseInfo
}

export interface EvtChampionSelectedPayload {
  selection: ChampionSelection
}

export interface EvtChampionHoveredPayload {
  side: 'blue' | 'red'
  championId: string | null
}

export interface EvtPlayerJoinedPayload {
  side: 'blue' | 'red'
  player: PlayerData
}

export interface EvtPlayerLeftPayload {
  side: 'blue' | 'red'
}

export interface EvtPlayerReadyChangedPayload {
  side: 'blue' | 'red'
  ready: boolean
}

export interface EvtDraftPausedPayload {
  pausedBy: string
  side: 'blue' | 'red'
  timerFrozen: number
  maxPauseTime?: number
}

export interface EvtDraftResumedPayload {
  timerRemaining: number
}

export interface EvtResumeReadyChangedPayload {
  blue: boolean
  red: boolean
}

export interface EvtResumeCountdownPayload {
  seconds: number
  cancelled?: boolean
}

export interface EvtEditProposedPayload {
  proposal: EditProposal
  proposedBy: string
  side: 'blue' | 'red'
  expiresAt: number
}

export interface EvtEditAppliedPayload {
  edit: EditProposal
  newState: PicksAndBans
}

export interface EvtEditRejectedPayload {
  rejectedBy?: string
  side?: 'blue' | 'red'
  expired?: boolean
}

export interface PicksAndBans {
  blueBans: string[]
  redBans: string[]
  bluePicks: string[]
  redPicks: string[]
}

// Typed event union
export type TypedEvent =
  | Event<'draft_started', EvtDraftStartedPayload>
  | Event<'draft_completed', EvtDraftCompletedPayload>
  | Event<'phase_changed', EvtPhaseChangedPayload>
  | Event<'champion_selected', EvtChampionSelectedPayload>
  | Event<'champion_hovered', EvtChampionHoveredPayload>
  | Event<'player_joined', EvtPlayerJoinedPayload>
  | Event<'player_left', EvtPlayerLeftPayload>
  | Event<'player_ready_changed', EvtPlayerReadyChangedPayload>
  | Event<'draft_paused', EvtDraftPausedPayload>
  | Event<'draft_resumed', EvtDraftResumedPayload>
  | Event<'resume_ready_changed', EvtResumeReadyChangedPayload>
  | Event<'resume_countdown', EvtResumeCountdownPayload>
  | Event<'edit_proposed', EvtEditProposedPayload>
  | Event<'edit_applied', EvtEditAppliedPayload>
  | Event<'edit_rejected', EvtEditRejectedPayload>

// ============================================================================
// STATE - Server → Client full state snapshot
// ============================================================================

export interface StatePayload {
  room: RoomStateV2
  draft: DraftStateV2
  players: PlayerStateV2
  client: ClientStateV2
}

export interface RoomStateV2 {
  id: string
  shortCode: string
  draftMode: 'pro_play' | 'fearless'
  status: 'waiting' | 'in_progress' | 'completed'
  timerDuration: number
}

export interface DraftStateV2 {
  currentPhase: number
  currentTeam: 'blue' | 'red' | null
  actionType: 'ban' | 'pick' | null
  timerRemainingMs: number
  picks: PicksAndBans
  isComplete: boolean
  isPaused: boolean
  pauseInfo?: PauseStateV2
  pendingEdit?: PendingEditStateV2
  resumeState?: ResumeStateV2
  fearlessBans?: string[]
}

export interface PauseStateV2 {
  pausedBy: string
  pausedSide: 'blue' | 'red'
  timerFrozen: number
}

export interface PendingEditStateV2 {
  proposal: EditProposal
  proposedBy: string
  side: 'blue' | 'red'
  expiresAt: number
}

export interface ResumeStateV2 {
  blueReady: boolean
  redReady: boolean
  countdown?: number
}

// PlayerStateV2 uses discriminated union for 1v1 vs team mode
export type PlayerStateV2 =
  | { mode: '1v1'; players: Players1v1 }
  | { mode: 'team'; teamPlayers: TeamPlayerData[] }

export interface Players1v1 {
  blue: PlayerData | null
  red: PlayerData | null
}

export interface TeamPlayerData {
  id: string
  displayName: string
  team: 'blue' | 'red'
  assignedRole: 'top' | 'jungle' | 'mid' | 'adc' | 'support'
  isCaptain: boolean
}

export interface ClientStateV2 {
  yourSide: 'blue' | 'red' | 'spectator'
  isCaptain: boolean
  spectatorCount: number
}

// ============================================================================
// TIMER - Server → Client high-frequency updates
// ============================================================================

export type TimerType = 'tick' | 'expired'

export interface TimerTick {
  timer: 'tick'
  remaining: number
  isBufferPeriod: boolean
}

export interface TimerExpired {
  timer: 'expired'
  phase: number
  autoSelected?: string
}

export type Timer = TimerTick | TimerExpired

// ============================================================================
// ERR - Server → Client error responses
// ============================================================================

export interface Err {
  code: string
  message: string
}

// ============================================================================
// Message Handler Types (for MessageRouter)
// ============================================================================

export type EventHandler<T> = (payload: T) => void

export interface EventHandlers {
  draft_started?: EventHandler<EvtDraftStartedPayload>
  draft_completed?: EventHandler<EvtDraftCompletedPayload>
  phase_changed?: EventHandler<EvtPhaseChangedPayload>
  champion_selected?: EventHandler<EvtChampionSelectedPayload>
  champion_hovered?: EventHandler<EvtChampionHoveredPayload>
  player_joined?: EventHandler<EvtPlayerJoinedPayload>
  player_left?: EventHandler<EvtPlayerLeftPayload>
  player_ready_changed?: EventHandler<EvtPlayerReadyChangedPayload>
  draft_paused?: EventHandler<EvtDraftPausedPayload>
  draft_resumed?: EventHandler<EvtDraftResumedPayload>
  resume_ready_changed?: EventHandler<EvtResumeReadyChangedPayload>
  resume_countdown?: EventHandler<EvtResumeCountdownPayload>
  edit_proposed?: EventHandler<EvtEditProposedPayload>
  edit_applied?: EventHandler<EvtEditAppliedPayload>
  edit_rejected?: EventHandler<EvtEditRejectedPayload>
}
