package websocket

import (
	"encoding/json"
	"time"
)

// New Protocol Message Types (v2)
// These coexist with the old types in message.go during migration

type MsgType string

const (
	// Client → Server
	MsgTypeCommand MsgType = "COMMAND"
	MsgTypeQuery   MsgType = "QUERY"

	// Server → Client
	MsgTypeEvent MsgType = "EVENT"
	MsgTypeState MsgType = "STATE"
	MsgTypeTimer MsgType = "TIMER"
	MsgTypeErr   MsgType = "ERR" // Renamed to avoid conflict with ERROR
)

// Msg is the new message envelope (v2 protocol)
type Msg struct {
	Type      MsgType         `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
	Seq       int             `json:"seq,omitempty"`
}

func NewMsg(msgType MsgType, payload interface{}) (*Msg, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Msg{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// ============================================================================
// COMMAND - Client → Server actions
// ============================================================================

type CommandAction string

const (
	CmdJoinRoom        CommandAction = "join_room"
	CmdSelectChampion  CommandAction = "select_champion"
	CmdLockIn          CommandAction = "lock_in"
	CmdHoverChampion   CommandAction = "hover_champion"
	CmdSetReady        CommandAction = "set_ready"
	CmdStartDraft      CommandAction = "start_draft"
	CmdPauseDraft      CommandAction = "pause_draft"
	CmdResumeReady     CommandAction = "resume_ready"
	CmdProposeEdit     CommandAction = "propose_edit"
	CmdRespondEdit     CommandAction = "respond_edit"
)

// Command is the envelope for all client→server actions
type Command struct {
	Action  CommandAction   `json:"action"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Command payloads

type CmdJoinRoomPayload struct {
	RoomID string `json:"roomId"`
	Side   string `json:"side"`
}

type CmdSelectChampionPayload struct {
	ChampionID string `json:"championId"`
}

type CmdHoverChampionPayload struct {
	ChampionID *string `json:"championId"`
}

type CmdSetReadyPayload struct {
	Ready bool `json:"ready"`
}

type CmdResumeReadyPayload struct {
	Ready bool `json:"ready"`
}

type CmdProposeEditPayload struct {
	SlotType   string `json:"slotType"`   // "ban" or "pick"
	Team       string `json:"team"`       // "blue" or "red"
	SlotIndex  int    `json:"slotIndex"`  // 0-4 for picks, 0-4 for bans
	ChampionID string `json:"championId"` // New champion to put in slot
}

type CmdRespondEditPayload struct {
	Accept bool `json:"accept"`
}

// ============================================================================
// QUERY - Client → Server state requests
// ============================================================================

type QueryType string

const (
	QuerySyncState QueryType = "sync_state"
)

type Query struct {
	Query QueryType `json:"query"`
}

// ============================================================================
// EVENT - Server → Client state changes
// ============================================================================

type EventType string

const (
	// Draft lifecycle
	EvtDraftStarted   EventType = "draft_started"
	EvtDraftCompleted EventType = "draft_completed"
	EvtPhaseChanged   EventType = "phase_changed"

	// Champion actions
	EvtChampionSelected EventType = "champion_selected"
	EvtChampionHovered  EventType = "champion_hovered"

	// Player events
	EvtPlayerJoined       EventType = "player_joined"
	EvtPlayerLeft         EventType = "player_left"
	EvtPlayerReadyChanged EventType = "player_ready_changed"

	// Pause/Resume
	EvtDraftPaused        EventType = "draft_paused"
	EvtDraftResumed       EventType = "draft_resumed"
	EvtResumeReadyChanged EventType = "resume_ready_changed"
	EvtResumeCountdown    EventType = "resume_countdown"

	// Edit workflow
	EvtEditProposed EventType = "edit_proposed"
	EvtEditApplied  EventType = "edit_applied"
	EvtEditRejected EventType = "edit_rejected"
)

// Event is the envelope for all server→client state changes
type Event struct {
	Event   EventType       `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func NewEvent(eventType EventType, payload interface{}) (*Event, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Event{
		Event:   eventType,
		Payload: payloadBytes,
	}, nil
}

// Event payloads

type PhaseInfo struct {
	CurrentPhase     int    `json:"currentPhase"`
	CurrentTeam      string `json:"currentTeam"`
	ActionType       string `json:"actionType"`
	TimerRemainingMs int    `json:"timerRemainingMs"`
}

type EvtDraftStartedPayload struct {
	Phase PhaseInfo `json:"phase"`
}

type EvtDraftCompletedPayload struct {
	Result DraftResult `json:"result"`
}

type DraftResult struct {
	BlueBans  []string `json:"blueBans"`
	RedBans   []string `json:"redBans"`
	BluePicks []string `json:"bluePicks"`
	RedPicks  []string `json:"redPicks"`
}

type EvtPhaseChangedPayload struct {
	Phase PhaseInfo `json:"phase"`
}

type ChampionSelection struct {
	Phase      int    `json:"phase"`
	Team       string `json:"team"`
	ActionType string `json:"actionType"`
	ChampionID string `json:"championId"`
}

type EvtChampionSelectedPayload struct {
	Selection ChampionSelection `json:"selection"`
}

type EvtChampionHoveredPayload struct {
	Side       string  `json:"side"`
	ChampionID *string `json:"championId"`
}

type EvtPlayerJoinedPayload struct {
	Side   string     `json:"side"`
	Player PlayerData `json:"player"`
}

type PlayerData struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

type EvtPlayerLeftPayload struct {
	Side string `json:"side"`
}

type EvtPlayerReadyChangedPayload struct {
	Side  string `json:"side"`
	Ready bool   `json:"ready"`
}

type EvtDraftPausedPayload struct {
	PausedBy     string `json:"pausedBy"`
	Side         string `json:"side"`
	TimerFrozen  int    `json:"timerFrozen"`
	MaxPauseTime int    `json:"maxPauseTime,omitempty"`
}

type EvtDraftResumedPayload struct {
	TimerRemaining int `json:"timerRemaining"`
}

type EvtResumeReadyChangedPayload struct {
	Blue bool `json:"blue"`
	Red  bool `json:"red"`
}

type EvtResumeCountdownPayload struct {
	Seconds   int  `json:"seconds"`
	Cancelled bool `json:"cancelled,omitempty"`
}

type EditProposal struct {
	SlotType      string `json:"slotType"`
	Team          string `json:"team"`
	SlotIndex     int    `json:"slotIndex"`
	OldChampionID string `json:"oldChampionId"`
	NewChampionID string `json:"newChampionId"`
}

type EvtEditProposedPayload struct {
	Proposal   EditProposal `json:"proposal"`
	ProposedBy string       `json:"proposedBy"`
	Side       string       `json:"side"`
	ExpiresAt  int64        `json:"expiresAt"`
}

type EvtEditAppliedPayload struct {
	Edit     EditProposal    `json:"edit"`
	NewState PicksAndBansV2  `json:"newState"`
}

type PicksAndBansV2 struct {
	BlueBans  []string `json:"blueBans"`
	RedBans   []string `json:"redBans"`
	BluePicks []string `json:"bluePicks"`
	RedPicks  []string `json:"redPicks"`
}

type EvtEditRejectedPayload struct {
	RejectedBy string `json:"rejectedBy,omitempty"` // Empty if auto-expired
	Side       string `json:"side,omitempty"`
	Expired    bool   `json:"expired,omitempty"`
}

// ============================================================================
// STATE - Server → Client full state snapshot
// ============================================================================

type StatePayload struct {
	Room    RoomStateV2    `json:"room"`
	Draft   DraftStateV2   `json:"draft"`
	Players PlayerStateV2  `json:"players"`
	Client  ClientStateV2  `json:"client"`
}

type RoomStateV2 struct {
	ID            string `json:"id"`
	ShortCode     string `json:"shortCode"`
	DraftMode     string `json:"draftMode"`
	Status        string `json:"status"`
	TimerDuration int    `json:"timerDuration"`
}

type DraftStateV2 struct {
	CurrentPhase     int             `json:"currentPhase"`
	CurrentTeam      *string         `json:"currentTeam"`
	ActionType       *string         `json:"actionType"`
	TimerRemainingMs int             `json:"timerRemainingMs"`
	Picks            PicksAndBansV2  `json:"picks"`
	IsComplete       bool            `json:"isComplete"`
	IsPaused         bool            `json:"isPaused"`
	PauseInfo        *PauseState     `json:"pauseInfo,omitempty"`
	PendingEdit      *PendingEditState `json:"pendingEdit,omitempty"`
	ResumeState      *ResumeState    `json:"resumeState,omitempty"`
	FearlessBans     []string        `json:"fearlessBans,omitempty"`
}

type PauseState struct {
	PausedBy    string `json:"pausedBy"`
	PausedSide  string `json:"pausedSide"`
	TimerFrozen int    `json:"timerFrozen"`
}

type PendingEditState struct {
	Proposal   EditProposal `json:"proposal"`
	ProposedBy string       `json:"proposedBy"`
	Side       string       `json:"side"`
	ExpiresAt  int64        `json:"expiresAt"`
}

type ResumeState struct {
	BlueReady bool `json:"blueReady"`
	RedReady  bool `json:"redReady"`
	Countdown int  `json:"countdown,omitempty"`
}

// PlayerStateV2 uses discriminated union pattern for 1v1 vs team mode
type PlayerStateV2 struct {
	Mode        string           `json:"mode"` // "1v1" or "team"
	Players     *Players1v1      `json:"players,omitempty"`     // For 1v1 mode
	TeamPlayers []TeamPlayerData `json:"teamPlayers,omitempty"` // For team mode
}

type Players1v1 struct {
	Blue *PlayerData `json:"blue"`
	Red  *PlayerData `json:"red"`
}

type TeamPlayerData struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	Team         string `json:"team"`
	AssignedRole string `json:"assignedRole"`
	IsCaptain    bool   `json:"isCaptain"`
}

type ClientStateV2 struct {
	YourSide       string `json:"yourSide"` // "blue", "red", or "spectator"
	IsCaptain      bool   `json:"isCaptain"`
	SpectatorCount int    `json:"spectatorCount"`
}

// ============================================================================
// TIMER - Server → Client high-frequency updates
// ============================================================================

type TimerType string

const (
	TimerTick    TimerType = "tick"
	TimerExpired TimerType = "expired"
)

type Timer struct {
	Timer   TimerType       `json:"timer"`
	Payload json.RawMessage `json:"payload"`
}

type TimerTickPayloadV2 struct {
	Remaining      int  `json:"remaining"`
	IsBufferPeriod bool `json:"isBufferPeriod"`
}

type TimerExpiredPayloadV2 struct {
	Phase        int     `json:"phase"`
	AutoSelected *string `json:"autoSelected"`
}

// ============================================================================
// ERR - Server → Client error responses
// ============================================================================

type Err struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
