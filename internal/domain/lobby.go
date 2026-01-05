package domain

import (
	"time"

	"github.com/google/uuid"
)

// LobbyStatus represents the current state of a lobby
type LobbyStatus string

const (
	LobbyStatusWaitingForPlayers LobbyStatus = "waiting_for_players"
	LobbyStatusMatchmaking       LobbyStatus = "matchmaking"
	LobbyStatusTeamSelected      LobbyStatus = "team_selected"
	LobbyStatusDrafting          LobbyStatus = "drafting"
	LobbyStatusCompleted         LobbyStatus = "completed"
)

// MaxLobbyPlayers is the maximum number of players in a lobby
const MaxLobbyPlayers = 10

// Lobby represents a 10-man lobby for matchmaking
type Lobby struct {
	ID                   uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ShortCode            string      `json:"shortCode" gorm:"uniqueIndex;size:10;not null"`
	CreatedBy            uuid.UUID   `json:"createdBy" gorm:"type:uuid;not null"`
	Status               LobbyStatus `json:"status" gorm:"type:varchar(30);not null;default:'waiting_for_players'"`
	SelectedMatchOption  *int        `json:"selectedMatchOption"`
	DraftMode            DraftMode   `json:"draftMode" gorm:"type:varchar(20);not null;default:'pro_play'"`
	TimerDurationSeconds int         `json:"timerDurationSeconds" gorm:"not null;default:30"`
	RoomID               *uuid.UUID  `json:"roomId" gorm:"type:uuid"`
	CreatedAt            time.Time   `json:"createdAt"`
	StartedAt            *time.Time  `json:"startedAt"`
	CompletedAt          *time.Time  `json:"completedAt"`

	// Relations
	Creator *User         `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Players []LobbyPlayer `json:"players,omitempty" gorm:"foreignKey:LobbyID"`
	Room    *Room         `json:"room,omitempty" gorm:"foreignKey:RoomID"`
}

// TableName returns the table name for GORM
func (Lobby) TableName() string {
	return "lobbies"
}

// IsFull returns true if the lobby has 10 players
func (l *Lobby) IsFull() bool {
	return len(l.Players) >= MaxLobbyPlayers
}

// CanStartMatchmaking returns true if matchmaking can be started
func (l *Lobby) CanStartMatchmaking() bool {
	if l.Status != LobbyStatusWaitingForPlayers {
		return false
	}
	if !l.IsFull() {
		return false
	}
	// Check all players are ready
	for _, p := range l.Players {
		if !p.IsReady {
			return false
		}
	}
	return true
}

// LobbyPlayer represents a player in a lobby
type LobbyPlayer struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LobbyID      uuid.UUID  `json:"lobbyId" gorm:"type:uuid;not null;index"`
	UserID       uuid.UUID  `json:"userId" gorm:"type:uuid;not null"`
	Team         *Side      `json:"team" gorm:"type:varchar(10)"`
	AssignedRole *Role      `json:"assignedRole" gorm:"type:varchar(10)"`
	IsReady      bool       `json:"isReady" gorm:"not null;default:false"`
	IsCaptain    bool       `json:"isCaptain" gorm:"not null;default:false"`
	JoinOrder    int        `json:"joinOrder" gorm:"not null;default:0"`
	JoinedAt     time.Time  `json:"joinedAt"`

	// Relations
	User  *User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Lobby *Lobby `json:"-" gorm:"foreignKey:LobbyID"`
}

// TableName returns the table name for GORM
func (LobbyPlayer) TableName() string {
	return "lobby_players"
}
