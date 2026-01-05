package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRoleProfile stores a user's rank and comfort rating for a specific role
type UserRoleProfile struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID        uuid.UUID  `json:"userId" gorm:"type:uuid;not null;uniqueIndex:idx_user_role_profiles_user_role"`
	Role          Role       `json:"role" gorm:"type:varchar(10);not null;uniqueIndex:idx_user_role_profiles_user_role"`
	LeagueRank    LeagueRank `json:"leagueRank" gorm:"type:varchar(20);not null;default:'Unranked'"`
	MMR           int        `json:"mmr" gorm:"not null;default:1200"`
	ComfortRating int        `json:"comfortRating" gorm:"not null;default:3"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`

	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName returns the table name for GORM
func (UserRoleProfile) TableName() string {
	return "user_role_profiles"
}

// Validate checks if the profile has valid values
func (p *UserRoleProfile) Validate() error {
	if !p.Role.IsValid() {
		return ErrInvalidRole
	}
	if !p.LeagueRank.IsValid() {
		return ErrInvalidRank
	}
	if p.ComfortRating < 1 || p.ComfortRating > 5 {
		return ErrInvalidComfortRating
	}
	if p.MMR < 0 {
		return ErrInvalidMMR
	}
	return nil
}

// SetRankAndUpdateMMR sets the league rank and automatically updates MMR
func (p *UserRoleProfile) SetRankAndUpdateMMR(rank LeagueRank) {
	p.LeagueRank = rank
	p.MMR = rank.ToMMR()
}

// NewDefaultUserRoleProfile creates a new profile with default values
func NewDefaultUserRoleProfile(userID uuid.UUID, role Role) *UserRoleProfile {
	return &UserRoleProfile{
		ID:            uuid.New(),
		UserID:        userID,
		Role:          role,
		LeagueRank:    RankUnranked,
		MMR:           1200,
		ComfortRating: 3,
	}
}

// CreateDefaultProfilesForUser creates default profiles for all 5 roles
func CreateDefaultProfilesForUser(userID uuid.UUID) []*UserRoleProfile {
	profiles := make([]*UserRoleProfile, len(AllRoles))
	for i, role := range AllRoles {
		profiles[i] = NewDefaultUserRoleProfile(userID, role)
	}
	return profiles
}
