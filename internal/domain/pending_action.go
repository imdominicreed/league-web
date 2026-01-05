package domain

import (
	"time"

	"github.com/google/uuid"
)

// PendingActionType represents the type of action awaiting approval
type PendingActionType string

const (
	PendingActionSwapPlayers  PendingActionType = "swap_players"
	PendingActionSwapRoles    PendingActionType = "swap_roles"
	PendingActionMatchmake    PendingActionType = "matchmake"
	PendingActionSelectOption PendingActionType = "select_option"
	PendingActionStartDraft   PendingActionType = "start_draft"
)

// PendingActionStatus represents the status of a pending action
type PendingActionStatus string

const (
	PendingStatusPending   PendingActionStatus = "pending"
	PendingStatusApproved  PendingActionStatus = "approved"
	PendingStatusCancelled PendingActionStatus = "cancelled"
	PendingStatusExpired   PendingActionStatus = "expired"
)

// PendingActionExpiryDuration is how long pending actions remain valid
const PendingActionExpiryDuration = 5 * time.Minute

// PendingAction represents an action that requires both captains to approve
type PendingAction struct {
	ID             uuid.UUID           `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LobbyID        uuid.UUID           `json:"lobbyId" gorm:"type:uuid;not null;index"`
	ActionType     PendingActionType   `json:"actionType" gorm:"type:varchar(30);not null"`
	Status         PendingActionStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	ProposedByUser uuid.UUID           `json:"proposedByUser" gorm:"type:uuid;not null"`
	ProposedBySide Side                `json:"proposedBySide" gorm:"type:varchar(10);not null"`

	// For swap_players: Player1ID (team A) <-> Player2ID (team B)
	// For swap_roles: Player1ID and Player2ID on same team
	Player1ID *uuid.UUID `json:"player1Id" gorm:"type:uuid"`
	Player2ID *uuid.UUID `json:"player2Id" gorm:"type:uuid"`

	// For matchmake: Store the selected option number once approved
	MatchOptionNum *int `json:"matchOptionNum"`

	ApprovedByBlue bool `json:"approvedByBlue" gorm:"not null;default:false"`
	ApprovedByRed  bool `json:"approvedByRed" gorm:"not null;default:false"`

	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`

	// Relations
	Lobby   *Lobby `json:"-" gorm:"foreignKey:LobbyID"`
	Player1 *User  `json:"player1,omitempty" gorm:"foreignKey:Player1ID"`
	Player2 *User  `json:"player2,omitempty" gorm:"foreignKey:Player2ID"`
}

// TableName returns the table name for GORM
func (PendingAction) TableName() string {
	return "pending_actions"
}

// IsExpired returns true if the action has expired
func (p *PendingAction) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// IsFullyApproved returns true if both captains have approved
func (p *PendingAction) IsFullyApproved() bool {
	return p.ApprovedByBlue && p.ApprovedByRed
}

// NewPendingAction creates a new pending action with default expiry
func NewPendingAction(lobbyID, proposedBy uuid.UUID, side Side, actionType PendingActionType) *PendingAction {
	now := time.Now()
	action := &PendingAction{
		ID:             uuid.New(),
		LobbyID:        lobbyID,
		ActionType:     actionType,
		Status:         PendingStatusPending,
		ProposedByUser: proposedBy,
		ProposedBySide: side,
		CreatedAt:      now,
		ExpiresAt:      now.Add(PendingActionExpiryDuration),
	}

	// The proposer's side is automatically approved
	if side == SideBlue {
		action.ApprovedByBlue = true
	} else if side == SideRed {
		action.ApprovedByRed = true
	}

	return action
}
