package websocket

import (
	"encoding/json"
	"time"
)

type MessageType string

const (
	// Client to Server
	MessageTypeJoinRoom        MessageType = "JOIN_ROOM"
	MessageTypeSelectChampion  MessageType = "SELECT_CHAMPION"
	MessageTypeLockIn          MessageType = "LOCK_IN"
	MessageTypeHoverChampion   MessageType = "HOVER_CHAMPION"
	MessageTypeSyncState       MessageType = "SYNC_STATE"
	MessageTypeReady           MessageType = "READY"
	MessageTypeStartDraft      MessageType = "START_DRAFT"
	MessageTypePauseDraft      MessageType = "PAUSE_DRAFT"
	MessageTypeResumeDraft     MessageType = "RESUME_DRAFT"
	MessageTypeProposeEdit     MessageType = "PROPOSE_EDIT"
	MessageTypeConfirmEdit     MessageType = "CONFIRM_EDIT"
	MessageTypeRejectEdit      MessageType = "REJECT_EDIT"
	MessageTypeReadyToResume   MessageType = "READY_TO_RESUME"

	// Server to Client
	MessageTypeStateSync        MessageType = "STATE_SYNC"
	MessageTypePlayerUpdate     MessageType = "PLAYER_UPDATE"
	MessageTypeDraftStarted     MessageType = "DRAFT_STARTED"
	MessageTypeChampionHovered  MessageType = "CHAMPION_HOVERED"
	MessageTypeChampionSelected MessageType = "CHAMPION_SELECTED"
	MessageTypePhaseChanged     MessageType = "PHASE_CHANGED"
	MessageTypeTimerTick        MessageType = "TIMER_TICK"
	MessageTypeTimerExpired     MessageType = "TIMER_EXPIRED"
	MessageTypeDraftCompleted   MessageType = "DRAFT_COMPLETED"
	MessageTypeDraftPaused      MessageType = "DRAFT_PAUSED"
	MessageTypeDraftResumed     MessageType = "DRAFT_RESUMED"
	MessageTypeEditProposed      MessageType = "EDIT_PROPOSED"
	MessageTypeEditApplied       MessageType = "EDIT_APPLIED"
	MessageTypeEditRejected      MessageType = "EDIT_REJECTED"
	MessageTypeResumeReadyUpdate MessageType = "RESUME_READY_UPDATE"
	MessageTypeResumeCountdown   MessageType = "RESUME_COUNTDOWN"
	MessageTypeError             MessageType = "ERROR"
)

type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
	Seq       int             `json:"seq,omitempty"`
}

func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// Client to Server payloads

type JoinRoomPayload struct {
	RoomID string `json:"roomId"`
	Side   string `json:"side"`
}

type SelectChampionPayload struct {
	ChampionID string `json:"championId"`
}

type HoverChampionPayload struct {
	ChampionID *string `json:"championId"`
}

type ReadyPayload struct {
	Ready bool `json:"ready"`
}

// Server to Client payloads

type StateSyncPayload struct {
	Room           RoomInfo         `json:"room"`
	Draft          DraftInfo        `json:"draft"`
	Players        PlayersInfo      `json:"players"`
	YourSide       string           `json:"yourSide"`
	IsCaptain      bool             `json:"isCaptain"`
	IsTeamDraft    bool             `json:"isTeamDraft"`
	TeamPlayers    []TeamPlayerInfo `json:"teamPlayers,omitempty"`
	SpectatorCount int              `json:"spectatorCount"`
	FearlessBans   []string         `json:"fearlessBans,omitempty"`
}

type TeamPlayerInfo struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	Team         string `json:"team"`
	AssignedRole string `json:"assignedRole"`
	IsCaptain    bool   `json:"isCaptain"`
}

type RoomInfo struct {
	ID            string `json:"id"`
	ShortCode     string `json:"shortCode"`
	DraftMode     string `json:"draftMode"`
	Status        string `json:"status"`
	TimerDuration int    `json:"timerDuration"`
}

type DraftInfo struct {
	CurrentPhase     int      `json:"currentPhase"`
	CurrentTeam      string   `json:"currentTeam"`
	ActionType       string   `json:"actionType"`
	TimerRemainingMs int      `json:"timerRemainingMs"`
	BlueBans         []string `json:"blueBans"`
	RedBans          []string `json:"redBans"`
	BluePicks        []string `json:"bluePicks"`
	RedPicks         []string `json:"redPicks"`
	IsComplete       bool     `json:"isComplete"`
	IsPaused         bool     `json:"isPaused"`
	PausedBy         string   `json:"pausedBy,omitempty"`
	PausedBySide     string   `json:"pausedBySide,omitempty"`
	PendingEdit      *PendingEditInfo `json:"pendingEdit,omitempty"`
	BlueResumeReady  bool     `json:"blueResumeReady,omitempty"`
	RedResumeReady   bool     `json:"redResumeReady,omitempty"`
	ResumeCountdown  int      `json:"resumeCountdown,omitempty"`
}

type PendingEditInfo struct {
	ProposedBy    string `json:"proposedBy"`
	ProposedSide  string `json:"proposedSide"`
	SlotType      string `json:"slotType"`
	Team          string `json:"team"`
	SlotIndex     int    `json:"slotIndex"`
	OldChampionID string `json:"oldChampionId"`
	NewChampionID string `json:"newChampionId"`
	ExpiresAt     int64  `json:"expiresAt"`
}

type PlayersInfo struct {
	Blue *PlayerInfo `json:"blue"`
	Red  *PlayerInfo `json:"red"`
}

type PlayerInfo struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Ready       bool   `json:"ready"`
}

type PlayerUpdatePayload struct {
	Side   string      `json:"side"`
	Player *PlayerInfo `json:"player"`
	Action string      `json:"action"` // "joined", "left", "ready_changed"
}

type DraftStartedPayload struct {
	CurrentPhase     int    `json:"currentPhase"`
	CurrentTeam      string `json:"currentTeam"`
	ActionType       string `json:"actionType"`
	TimerRemainingMs int    `json:"timerRemainingMs"`
}

type ChampionHoveredPayload struct {
	Side       string  `json:"side"`
	ChampionID *string `json:"championId"`
}

type ChampionSelectedPayload struct {
	Phase      int    `json:"phase"`
	Team       string `json:"team"`
	ActionType string `json:"actionType"`
	ChampionID string `json:"championId"`
}

type PhaseChangedPayload struct {
	CurrentPhase     int    `json:"currentPhase"`
	CurrentTeam      string `json:"currentTeam"`
	ActionType       string `json:"actionType"`
	TimerRemainingMs int    `json:"timerRemainingMs"`
}

type TimerTickPayload struct {
	RemainingMs    int  `json:"remainingMs"`
	IsBufferPeriod bool `json:"isBufferPeriod"`
}

type TimerExpiredPayload struct {
	Phase        int     `json:"phase"`
	AutoSelected *string `json:"autoSelected"`
}

type DraftCompletedPayload struct {
	BlueBans  []string `json:"blueBans"`
	RedBans   []string `json:"redBans"`
	BluePicks []string `json:"bluePicks"`
	RedPicks  []string `json:"redPicks"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Pause/Edit payloads

type ProposeEditPayload struct {
	SlotType   string `json:"slotType"`   // "ban" or "pick"
	Team       string `json:"team"`       // "blue" or "red"
	SlotIndex  int    `json:"slotIndex"`  // 0-4 for picks, 0-4 for bans
	ChampionID string `json:"championId"` // New champion to put in slot
}

type DraftPausedPayload struct {
	PausedBy         string `json:"pausedBy"`
	PausedBySide     string `json:"pausedBySide"`
	TimerFrozenAt    int    `json:"timerFrozenAt"`
	MaxPauseDuration int    `json:"maxPauseDuration"`
}

type DraftResumedPayload struct {
	ResumedBy        string `json:"resumedBy"`
	TimerRemainingMs int    `json:"timerRemainingMs"`
}

type EditProposedPayload struct {
	ProposedBy    string `json:"proposedBy"`
	ProposedSide  string `json:"proposedSide"`
	SlotType      string `json:"slotType"`
	Team          string `json:"team"`
	SlotIndex     int    `json:"slotIndex"`
	OldChampionID string `json:"oldChampionId"`
	NewChampionID string `json:"newChampionId"`
	ExpiresAt     int64  `json:"expiresAt"`
}

type EditAppliedPayload struct {
	SlotType      string   `json:"slotType"`
	Team          string   `json:"team"`
	SlotIndex     int      `json:"slotIndex"`
	OldChampionID string   `json:"oldChampionId"`
	NewChampionID string   `json:"newChampionId"`
	BlueBans      []string `json:"blueBans"`
	RedBans       []string `json:"redBans"`
	BluePicks     []string `json:"bluePicks"`
	RedPicks      []string `json:"redPicks"`
}

type EditRejectedPayload struct {
	RejectedBy   string `json:"rejectedBy"`
	RejectedSide string `json:"rejectedSide"`
}

// Resume ready payloads

type ReadyToResumePayload struct {
	Ready bool `json:"ready"`
}

type ResumeReadyUpdatePayload struct {
	BlueReady bool `json:"blueReady"`
	RedReady  bool `json:"redReady"`
}

type ResumeCountdownPayload struct {
	SecondsRemaining int    `json:"secondsRemaining"`
	CancelledBy      string `json:"cancelledBy,omitempty"`
}
