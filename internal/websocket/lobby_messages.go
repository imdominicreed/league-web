package websocket

import (
	"time"

	"github.com/google/uuid"
)

// LobbyMessageType represents the type of lobby WebSocket message
type LobbyMessageType string

const (
	// Server -> Client events
	LobbyMsgStateSync             LobbyMessageType = "lobby_state_sync"
	LobbyMsgPlayerJoined          LobbyMessageType = "player_joined"
	LobbyMsgPlayerLeft            LobbyMessageType = "player_left"
	LobbyMsgPlayerReadyChanged    LobbyMessageType = "player_ready_changed"
	LobbyMsgStatusChanged         LobbyMessageType = "status_changed"
	LobbyMsgMatchOptionsGenerated LobbyMessageType = "match_options_generated"
	LobbyMsgTeamSelected          LobbyMessageType = "team_selected"
	LobbyMsgVoteCast              LobbyMessageType = "vote_cast"
	LobbyMsgActionProposed        LobbyMessageType = "action_proposed"
	LobbyMsgActionApproved        LobbyMessageType = "action_approved"
	LobbyMsgActionExecuted        LobbyMessageType = "action_executed"
	LobbyMsgActionCancelled       LobbyMessageType = "action_cancelled"
	LobbyMsgDraftStarting         LobbyMessageType = "draft_starting"
	LobbyMsgCaptainChanged        LobbyMessageType = "captain_changed"
	LobbyMsgPlayerKicked          LobbyMessageType = "player_kicked"
	LobbyMsgTeamStatsUpdated      LobbyMessageType = "team_stats_updated"
	LobbyMsgVotingStatusUpdated   LobbyMessageType = "voting_status_updated"
	LobbyMsgError                 LobbyMessageType = "error"

	// Client -> Server commands
	LobbyMsgJoinLobby LobbyMessageType = "join_lobby"
)

// LobbyMessage is the envelope for all lobby WebSocket messages
type LobbyMessage struct {
	Type      LobbyMessageType `json:"type"`
	Payload   interface{}      `json:"payload,omitempty"`
	Timestamp int64            `json:"timestamp"`
}

// NewLobbyMessage creates a new lobby message
func NewLobbyMessage(msgType LobbyMessageType, payload interface{}) *LobbyMessage {
	return &LobbyMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ============== Payload types ==============

// LobbyStateSyncPayload is the full state sent on join
type LobbyStateSyncPayload struct {
	Lobby         *LobbyInfo            `json:"lobby"`
	Players       []LobbyPlayerInfo     `json:"players"`
	MatchOptions  []MatchOptionInfo     `json:"matchOptions,omitempty"`
	TeamStats     *TeamStatsInfo        `json:"teamStats,omitempty"`
	VotingStatus  *VotingStatusInfo     `json:"votingStatus,omitempty"`
	Votes         map[string]int        `json:"votes"`         // userID -> optionNumber (in-memory)
	PendingAction *InMemoryActionInfo   `json:"pendingAction"` // in-memory pending action
}

// LobbyInfo contains lobby metadata
type LobbyInfo struct {
	ID                   string  `json:"id"`
	ShortCode            string  `json:"shortCode"`
	CreatedBy            string  `json:"createdBy"`
	Status               string  `json:"status"`
	SelectedMatchOption  *int    `json:"selectedMatchOption"`
	DraftMode            string  `json:"draftMode"`
	TimerDurationSeconds int     `json:"timerDurationSeconds"`
	RoomID               *string `json:"roomId"`
	VotingEnabled        bool    `json:"votingEnabled"`
	VotingMode           string  `json:"votingMode"`
	VotingDeadline       *string `json:"votingDeadline,omitempty"`
}

// LobbyPlayerInfo contains player info
type LobbyPlayerInfo struct {
	ID           string  `json:"id"`
	UserID       string  `json:"userId"`
	DisplayName  string  `json:"displayName"`
	Team         *string `json:"team"`
	AssignedRole *string `json:"assignedRole"`
	IsReady      bool    `json:"isReady"`
	IsCaptain    bool    `json:"isCaptain"`
	JoinOrder    int     `json:"joinOrder"`
}

// MatchOptionInfo contains match option data
type MatchOptionInfo struct {
	OptionNumber   int                    `json:"optionNumber"`
	AlgorithmType  string                 `json:"algorithmType"`
	BlueTeamAvgMMR int                    `json:"blueTeamAvgMmr"`
	RedTeamAvgMMR  int                    `json:"redTeamAvgMmr"`
	MMRDifference  int                    `json:"mmrDifference"`
	BalanceScore   float64                `json:"balanceScore"`
	AvgBlueComfort float64                `json:"avgBlueComfort"`
	AvgRedComfort  float64                `json:"avgRedComfort"`
	MaxLaneDiff    int                    `json:"maxLaneDiff"`
	Assignments    []AssignmentInfo       `json:"assignments"`
}

// AssignmentInfo contains player assignment data
type AssignmentInfo struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	Team          string `json:"team"`
	AssignedRole  string `json:"assignedRole"`
	RoleMMR       int    `json:"roleMmr"`
	ComfortRating int    `json:"comfortRating"`
}

// TeamStatsInfo contains team statistics
type TeamStatsInfo struct {
	BlueTeamAvgMMR int            `json:"blueTeamAvgMmr"`
	RedTeamAvgMMR  int            `json:"redTeamAvgMmr"`
	MMRDifference  int            `json:"mmrDifference"`
	AvgBlueComfort float64        `json:"avgBlueComfort"`
	AvgRedComfort  float64        `json:"avgRedComfort"`
	LaneDiffs      map[string]int `json:"laneDiffs"`
}

// VotingStatusInfo contains voting status
type VotingStatusInfo struct {
	VotingEnabled bool                       `json:"votingEnabled"`
	VotingMode    string                     `json:"votingMode"`
	Deadline      *string                    `json:"deadline,omitempty"`
	TotalPlayers  int                        `json:"totalPlayers"`
	VotesCast     int                        `json:"votesCast"`
	VoteCounts    map[int]int                `json:"voteCounts"`
	Voters        map[int][]VoterInfoPayload `json:"voters"`
	WinningOption *int                       `json:"winningOption,omitempty"`
	CanFinalize   bool                       `json:"canFinalize"`
}

// VoterInfoPayload contains voter info
type VoterInfoPayload struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

// InMemoryActionInfo contains in-memory pending action data
type InMemoryActionInfo struct {
	ID             string    `json:"id"`
	ActionType     string    `json:"actionType"`
	Status         string    `json:"status"`
	ProposedByUser string    `json:"proposedByUser"`
	ProposedBySide string    `json:"proposedBySide"`
	Player1ID      *string   `json:"player1Id,omitempty"`
	Player2ID      *string   `json:"player2Id,omitempty"`
	MatchOptionNum *int      `json:"matchOptionNum,omitempty"`
	ApprovedByBlue bool      `json:"approvedByBlue"`
	ApprovedByRed  bool      `json:"approvedByRed"`
	ExpiresAt      string    `json:"expiresAt"`
}

// ============== Event payloads ==============

// PlayerJoinedPayload is sent when a player joins
type PlayerJoinedPayload struct {
	Player LobbyPlayerInfo `json:"player"`
}

// PlayerLeftPayload is sent when a player leaves
type PlayerLeftPayload struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

// PlayerReadyChangedPayload is sent when a player's ready status changes
type PlayerReadyChangedPayload struct {
	UserID  string `json:"userId"`
	IsReady bool   `json:"isReady"`
}

// StatusChangedPayload is sent when lobby status changes
type StatusChangedPayload struct {
	OldStatus string `json:"oldStatus"`
	NewStatus string `json:"newStatus"`
}

// MatchOptionsGeneratedPayload is sent when teams are generated
type MatchOptionsGeneratedPayload struct {
	Options []MatchOptionInfo `json:"options"`
}

// TeamSelectedPayload is sent when a team option is selected
type TeamSelectedPayload struct {
	OptionNumber int                 `json:"optionNumber"`
	Assignments  []LobbyPlayerInfo   `json:"assignments"`
	TeamStats    *TeamStatsInfo      `json:"teamStats,omitempty"`
}

// VoteCastPayload is sent when someone votes
type VoteCastPayload struct {
	UserID       string                     `json:"userId"`
	DisplayName  string                     `json:"displayName"`
	OptionNumber int                        `json:"optionNumber"`
	VoteCounts   map[int]int                `json:"voteCounts"`
	VotesCast    int                        `json:"votesCast"`
	Voters       map[int][]VoterInfoPayload `json:"voters"`
}

// ActionProposedPayload is sent when a captain proposes an action
type ActionProposedPayload struct {
	Action InMemoryActionInfo `json:"action"`
}

// ActionApprovedPayload is sent when a captain approves
type ActionApprovedPayload struct {
	ActionID       string `json:"actionId"`
	ApprovedBySide string `json:"approvedBySide"`
	ApprovedByBlue bool   `json:"approvedByBlue"`
	ApprovedByRed  bool   `json:"approvedByRed"`
}

// ActionExecutedPayload is sent when action is fully approved and executed
type ActionExecutedPayload struct {
	ActionType string          `json:"actionType"`
	Result     interface{}     `json:"result,omitempty"` // depends on action type
}

// ActionCancelledPayload is sent when action is cancelled
type ActionCancelledPayload struct {
	ActionID     string `json:"actionId"`
	CancelledBy  string `json:"cancelledBy"`
}

// DraftStartingPayload is sent when transitioning to draft
type DraftStartingPayload struct {
	RoomID    string `json:"roomId"`
	ShortCode string `json:"shortCode"`
}

// CaptainChangedPayload is sent when captain changes
type CaptainChangedPayload struct {
	Team           string `json:"team"`
	NewCaptainID   string `json:"newCaptainId"`
	NewCaptainName string `json:"newCaptainName"`
	OldCaptainID   string `json:"oldCaptainId,omitempty"`
}

// PlayerKickedPayload is sent when a player is kicked
type PlayerKickedPayload struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	KickedBy    string `json:"kickedBy"`
}

// TeamStatsUpdatedPayload is sent when team stats are recalculated
type TeamStatsUpdatedPayload struct {
	Stats TeamStatsInfo `json:"stats"`
}

// VotingStatusUpdatedPayload is sent when voting status changes
type VotingStatusUpdatedPayload struct {
	Status VotingStatusInfo `json:"status"`
}

// LobbyErrorPayload is sent on errors
type LobbyErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ============== Client -> Server ==============

// JoinLobbyPayload is sent by client to join a lobby
type JoinLobbyPayload struct {
	LobbyID string `json:"lobbyId"`
}

// ============== In-memory structures ==============

// InMemoryPendingAction stores a pending action in memory
type InMemoryPendingAction struct {
	ID             uuid.UUID
	ActionType     string
	Status         string // "pending", "approved", "cancelled", "executed"
	ProposedByUser uuid.UUID
	ProposedBySide string // "blue" or "red"
	Player1ID      *uuid.UUID
	Player2ID      *uuid.UUID
	MatchOptionNum *int
	ApprovedByBlue bool
	ApprovedByRed  bool
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// ToInfo converts to wire format
func (a *InMemoryPendingAction) ToInfo() *InMemoryActionInfo {
	if a == nil {
		return nil
	}
	info := &InMemoryActionInfo{
		ID:             a.ID.String(),
		ActionType:     a.ActionType,
		Status:         a.Status,
		ProposedByUser: a.ProposedByUser.String(),
		ProposedBySide: a.ProposedBySide,
		ApprovedByBlue: a.ApprovedByBlue,
		ApprovedByRed:  a.ApprovedByRed,
		ExpiresAt:      a.ExpiresAt.Format(time.RFC3339),
		MatchOptionNum: a.MatchOptionNum,
	}
	if a.Player1ID != nil {
		s := a.Player1ID.String()
		info.Player1ID = &s
	}
	if a.Player2ID != nil {
		s := a.Player2ID.String()
		info.Player2ID = &s
	}
	return info
}

// IsExpired checks if the action has expired
func (a *InMemoryPendingAction) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}

// InMemoryVote stores a vote in memory
type InMemoryVote struct {
	UserID       uuid.UUID
	OptionNumber int
	CastAt       time.Time
}
