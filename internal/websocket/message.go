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
	MessageTypeError            MessageType = "ERROR"
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
	Room           RoomInfo    `json:"room"`
	Draft          DraftInfo   `json:"draft"`
	Players        PlayersInfo `json:"players"`
	YourSide       string      `json:"yourSide"`
	IsCaptain      bool        `json:"isCaptain"`
	IsTeamDraft    bool        `json:"isTeamDraft"`
	SpectatorCount int         `json:"spectatorCount"`
	FearlessBans   []string    `json:"fearlessBans,omitempty"`
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
	RemainingMs int `json:"remainingMs"`
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
