package domain

import (
	"time"

	"github.com/google/uuid"
)

// AlgorithmType represents the matchmaking algorithm used to generate an option
type AlgorithmType string

const (
	AlgorithmMMRBalanced  AlgorithmType = "mmr_balanced"
	AlgorithmRoleComfort  AlgorithmType = "role_comfort"
	AlgorithmHybrid       AlgorithmType = "hybrid"
	AlgorithmLaneBalanced AlgorithmType = "lane_balanced"
	AlgorithmComfortFirst AlgorithmType = "comfort_first"
)

// AlgorithmLabels maps algorithm types to human-readable labels
var AlgorithmLabels = map[AlgorithmType]string{
	AlgorithmMMRBalanced:  "Most Balanced",
	AlgorithmRoleComfort:  "Best Role Fit",
	AlgorithmHybrid:       "Balanced Overall",
	AlgorithmLaneBalanced: "Fair Lanes",
	AlgorithmComfortFirst: "Best Comfort",
}

// MatchOption represents a possible team composition from matchmaking
type MatchOption struct {
	ID               uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LobbyID          uuid.UUID     `json:"lobbyId" gorm:"type:uuid;not null;index"`
	OptionNumber     int           `json:"optionNumber" gorm:"not null"`
	AlgorithmType    AlgorithmType `json:"algorithmType" gorm:"type:varchar(20);not null;default:'comfort_first'"`
	BlueTeamAvgMMR   int           `json:"blueTeamAvgMmr" gorm:"not null"`
	RedTeamAvgMMR    int           `json:"redTeamAvgMmr" gorm:"not null"`
	MMRDifference    int           `json:"mmrDifference" gorm:"not null"`
	BalanceScore     float64       `json:"balanceScore" gorm:"type:decimal(5,2);not null"`
	AvgBlueComfort   float64       `json:"avgBlueComfort" gorm:"type:decimal(3,2)"`
	AvgRedComfort    float64       `json:"avgRedComfort" gorm:"type:decimal(3,2)"`
	MaxLaneDiff      int           `json:"maxLaneDiff" gorm:"default:0"`
	UsedMmrThreshold int           `json:"usedMmrThreshold" gorm:"default:0"`
	CreatedAt        time.Time     `json:"createdAt"`

	// Relations
	Assignments []MatchOptionAssignment `json:"assignments,omitempty" gorm:"foreignKey:MatchOptionID"`
	Lobby       *Lobby                  `json:"-" gorm:"foreignKey:LobbyID"`
}

// TableName returns the table name for GORM
func (MatchOption) TableName() string {
	return "match_options"
}

// GetBlueTeam returns assignments for the blue team
func (m *MatchOption) GetBlueTeam() []MatchOptionAssignment {
	var blue []MatchOptionAssignment
	for _, a := range m.Assignments {
		if a.Team == SideBlue {
			blue = append(blue, a)
		}
	}
	return blue
}

// GetRedTeam returns assignments for the red team
func (m *MatchOption) GetRedTeam() []MatchOptionAssignment {
	var red []MatchOptionAssignment
	for _, a := range m.Assignments {
		if a.Team == SideRed {
			red = append(red, a)
		}
	}
	return red
}

// MatchOptionAssignment represents a player's team and role assignment within a match option
type MatchOptionAssignment struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MatchOptionID uuid.UUID `json:"matchOptionId" gorm:"type:uuid;not null;index"`
	UserID        uuid.UUID `json:"userId" gorm:"type:uuid;not null"`
	Team          Side      `json:"team" gorm:"type:varchar(10);not null"`
	AssignedRole  Role      `json:"assignedRole" gorm:"type:varchar(10);not null"`
	RoleMMR       int       `json:"roleMmr" gorm:"not null"`
	ComfortRating int       `json:"comfortRating" gorm:"not null"`

	// Relations
	User        *User        `json:"user,omitempty" gorm:"foreignKey:UserID"`
	MatchOption *MatchOption `json:"-" gorm:"foreignKey:MatchOptionID"`
}

// TableName returns the table name for GORM
func (MatchOptionAssignment) TableName() string {
	return "match_option_assignments"
}

// RoomPlayer represents a player assigned to a draft room with their role
type RoomPlayer struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoomID       uuid.UUID `json:"roomId" gorm:"type:uuid;not null;index"`
	UserID       uuid.UUID `json:"userId" gorm:"type:uuid;not null"`
	Team         Side      `json:"team" gorm:"type:varchar(10);not null"`
	AssignedRole Role      `json:"assignedRole" gorm:"type:varchar(10);not null"`
	DisplayName  string    `json:"displayName" gorm:"type:varchar(100)"`
	IsCaptain    bool      `json:"isCaptain" gorm:"not null;default:false"`
	IsReady      bool      `json:"isReady" gorm:"not null;default:false"`
	JoinedAt     time.Time `json:"joinedAt" gorm:"autoCreateTime"`

	// Relations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Room *Room `json:"-" gorm:"foreignKey:RoomID"`
}

// TableName returns the table name for GORM
func (RoomPlayer) TableName() string {
	return "room_players"
}
