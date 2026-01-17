package domain

import (
	"time"

	"github.com/google/uuid"
)

// VotingMode represents the voting rule for selecting a match option
type VotingMode string

const (
	VotingModeMajority        VotingMode = "majority"         // >50% wins
	VotingModeUnanimous       VotingMode = "unanimous"        // 100% required
	VotingModeCaptainOverride VotingMode = "captain_override" // Voting + captain can force
)

// Vote represents a player's vote for a match option
type Vote struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	LobbyID        uuid.UUID `json:"lobbyId" gorm:"type:uuid;not null;index"`
	UserID         uuid.UUID `json:"userId" gorm:"type:uuid;not null"`
	MatchOptionNum int       `json:"matchOptionNum" gorm:"not null"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// Relations
	User  *User  `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Lobby *Lobby `json:"-" gorm:"foreignKey:LobbyID"`
}

// TableName returns the table name for GORM
func (Vote) TableName() string {
	return "votes"
}

// VoterInfo represents a user who cast a vote
type VoterInfo struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
}

// VotingStatus represents the current voting state for a lobby
type VotingStatus struct {
	VotingEnabled bool                  `json:"votingEnabled"`
	VotingMode    VotingMode            `json:"votingMode"`
	Deadline      *time.Time            `json:"deadline,omitempty"`
	TotalPlayers  int                   `json:"totalPlayers"`
	VotesCast     int                   `json:"votesCast"`
	VoteCounts    map[int]int           `json:"voteCounts"`
	Voters        map[int][]VoterInfo   `json:"voters"` // option number -> list of voters
	UserVotes     []int                 `json:"userVotes,omitempty"` // options the user has voted for
	WinningOption *int                  `json:"winningOption,omitempty"`
	CanFinalize   bool                  `json:"canFinalize"`
}
