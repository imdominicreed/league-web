package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ActionType string

const (
	ActionTypeBan  ActionType = "ban"
	ActionTypePick ActionType = "pick"
)

type DraftState struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoomID         uuid.UUID      `json:"roomId" gorm:"type:uuid;uniqueIndex;not null"`
	CurrentPhase   int            `json:"currentPhase" gorm:"not null;default:0"`
	CurrentTeam    *Side          `json:"currentTeam"`
	ActionType     *ActionType    `json:"actionType"`
	TimerStartedAt *time.Time     `json:"timerStartedAt"`
	TimerRemaining int            `json:"timerRemaining"` // milliseconds
	BlueBans       datatypes.JSON `json:"blueBans" gorm:"type:jsonb;default:'[]'"`
	RedBans        datatypes.JSON `json:"redBans" gorm:"type:jsonb;default:'[]'"`
	BluePicks      datatypes.JSON `json:"bluePicks" gorm:"type:jsonb;default:'[]'"`
	RedPicks       datatypes.JSON `json:"redPicks" gorm:"type:jsonb;default:'[]'"`
	IsComplete     bool           `json:"isComplete" gorm:"not null;default:false"`
	UpdatedAt      time.Time      `json:"updatedAt"`

	// Relations
	Room *Room `json:"room,omitempty" gorm:"foreignKey:RoomID"`
}

type DraftAction struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoomID     uuid.UUID  `json:"roomId" gorm:"type:uuid;index;not null"`
	PhaseIndex int        `json:"phaseIndex" gorm:"not null"`
	Team       Side       `json:"team" gorm:"not null"`
	ActionType ActionType `json:"actionType" gorm:"not null"`
	ChampionID string     `json:"championId" gorm:"not null"`
	UserID     *uuid.UUID `json:"userId" gorm:"type:uuid"`
	ActionTime time.Time  `json:"actionTime"`
}

type FearlessBan struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SeriesID     uuid.UUID `json:"seriesId" gorm:"type:uuid;index;not null"`
	ChampionID   string    `json:"championId" gorm:"not null"`
	BannedInGame int       `json:"bannedInGame" gorm:"not null"`
	PickedByTeam Side      `json:"pickedByTeam" gorm:"not null"`
}

// Phase represents a single step in the draft
type Phase struct {
	Index      int
	Team       Side
	ActionType ActionType
}

// ProPlayPhases defines the 20-phase pro play draft order
var ProPlayPhases = []Phase{
	// Ban Phase 1 (6 bans)
	{0, SideBlue, ActionTypeBan},
	{1, SideRed, ActionTypeBan},
	{2, SideBlue, ActionTypeBan},
	{3, SideRed, ActionTypeBan},
	{4, SideBlue, ActionTypeBan},
	{5, SideRed, ActionTypeBan},
	// Pick Phase 1 (6 picks: B, RR, BB, R)
	{6, SideBlue, ActionTypePick},
	{7, SideRed, ActionTypePick},
	{8, SideRed, ActionTypePick},
	{9, SideBlue, ActionTypePick},
	{10, SideBlue, ActionTypePick},
	{11, SideRed, ActionTypePick},
	// Ban Phase 2 (4 bans: R, B, R, B)
	{12, SideRed, ActionTypeBan},
	{13, SideBlue, ActionTypeBan},
	{14, SideRed, ActionTypeBan},
	{15, SideBlue, ActionTypeBan},
	// Pick Phase 2 (4 picks: R, BB, R)
	{16, SideRed, ActionTypePick},
	{17, SideBlue, ActionTypePick},
	{18, SideBlue, ActionTypePick},
	{19, SideRed, ActionTypePick},
}

// GetPhase returns the phase configuration for a given phase index
func GetPhase(index int) *Phase {
	if index < 0 || index >= len(ProPlayPhases) {
		return nil
	}
	return &ProPlayPhases[index]
}

// TotalPhases returns the total number of phases in a pro play draft
func TotalPhases() int {
	return len(ProPlayPhases)
}
