package service

import (
	"context"
	"errors"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrInvalidRole     = errors.New("invalid role")
)

type ProfileService struct {
	userRepo        repository.UserRepository
	roleProfileRepo repository.UserRoleProfileRepository
}

func NewProfileService(userRepo repository.UserRepository, roleProfileRepo repository.UserRoleProfileRepository) *ProfileService {
	return &ProfileService{
		userRepo:        userRepo,
		roleProfileRepo: roleProfileRepo,
	}
}

// UserProfileResponse contains user info with all role profiles
type UserProfileResponse struct {
	User         *domain.User                       `json:"user"`
	RoleProfiles map[domain.Role]*RoleProfileData   `json:"roleProfiles"`
}

// RoleProfileData is the response format for a single role profile
type RoleProfileData struct {
	Role          domain.Role       `json:"role"`
	LeagueRank    domain.LeagueRank `json:"leagueRank"`
	MMR           int               `json:"mmr"`
	ComfortRating int               `json:"comfortRating"`
}

// UpdateRoleProfileInput contains the data for updating a role profile
type UpdateRoleProfileInput struct {
	LeagueRank    *domain.LeagueRank `json:"leagueRank"`
	MMR           *int               `json:"mmr"`
	ComfortRating *int               `json:"comfortRating"`
}

// GetUserProfile returns the user with all their role profiles
func (s *ProfileService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfileResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	profiles, err := s.roleProfileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert to map
	roleProfiles := make(map[domain.Role]*RoleProfileData)
	for _, p := range profiles {
		roleProfiles[p.Role] = &RoleProfileData{
			Role:          p.Role,
			LeagueRank:    p.LeagueRank,
			MMR:           p.MMR,
			ComfortRating: p.ComfortRating,
		}
	}

	// Ensure all roles exist with defaults if not present
	for _, role := range domain.AllRoles {
		if _, ok := roleProfiles[role]; !ok {
			roleProfiles[role] = &RoleProfileData{
				Role:          role,
				LeagueRank:    domain.RankUnranked,
				MMR:           1200,
				ComfortRating: 3,
			}
		}
	}

	return &UserProfileResponse{
		User:         user,
		RoleProfiles: roleProfiles,
	}, nil
}

// GetRoleProfiles returns all role profiles for a user
func (s *ProfileService) GetRoleProfiles(ctx context.Context, userID uuid.UUID) ([]*domain.UserRoleProfile, error) {
	profiles, err := s.roleProfileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If no profiles exist, initialize them
	if len(profiles) == 0 {
		if err := s.InitializeProfiles(ctx, userID); err != nil {
			return nil, err
		}
		return s.roleProfileRepo.GetByUserID(ctx, userID)
	}

	return profiles, nil
}

// UpdateRoleProfile updates the rank and comfort for a specific role
func (s *ProfileService) UpdateRoleProfile(ctx context.Context, userID uuid.UUID, role domain.Role, input UpdateRoleProfileInput) (*domain.UserRoleProfile, error) {
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}

	profile, err := s.roleProfileRepo.GetByUserIDAndRole(ctx, userID, role)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new profile
			profile = domain.NewDefaultUserRoleProfile(userID, role)
		} else {
			return nil, err
		}
	}

	// Apply updates
	if input.LeagueRank != nil {
		if !input.LeagueRank.IsValid() {
			return nil, domain.ErrInvalidRank
		}
		profile.LeagueRank = *input.LeagueRank
		// Update MMR from rank unless MMR is also provided
		if input.MMR == nil {
			profile.MMR = input.LeagueRank.ToMMR()
		}
	}

	if input.MMR != nil {
		if *input.MMR < 0 {
			return nil, domain.ErrInvalidMMR
		}
		profile.MMR = *input.MMR
	}

	if input.ComfortRating != nil {
		if *input.ComfortRating < 1 || *input.ComfortRating > 5 {
			return nil, domain.ErrInvalidComfortRating
		}
		profile.ComfortRating = *input.ComfortRating
	}

	// Upsert the profile
	if err := s.roleProfileRepo.Upsert(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// InitializeProfiles creates default profiles for all 5 roles
func (s *ProfileService) InitializeProfiles(ctx context.Context, userID uuid.UUID) error {
	// Check if profiles already exist
	existing, err := s.roleProfileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	existingRoles := make(map[domain.Role]bool)
	for _, p := range existing {
		existingRoles[p.Role] = true
	}

	// Create missing profiles
	var profilesToCreate []*domain.UserRoleProfile
	for _, role := range domain.AllRoles {
		if !existingRoles[role] {
			profilesToCreate = append(profilesToCreate, domain.NewDefaultUserRoleProfile(userID, role))
		}
	}

	if len(profilesToCreate) > 0 {
		return s.roleProfileRepo.CreateMany(ctx, profilesToCreate)
	}

	return nil
}

// GetProfilesForUsers returns role profiles for multiple users (used for matchmaking)
func (s *ProfileService) GetProfilesForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]map[domain.Role]*domain.UserRoleProfile, error) {
	profilesByUser, err := s.roleProfileRepo.GetByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]map[domain.Role]*domain.UserRoleProfile)
	for userID, profiles := range profilesByUser {
		result[userID] = make(map[domain.Role]*domain.UserRoleProfile)
		for _, p := range profiles {
			result[userID][p.Role] = p
		}
	}

	// Ensure all users have all roles (with defaults)
	for _, userID := range userIDs {
		if _, ok := result[userID]; !ok {
			result[userID] = make(map[domain.Role]*domain.UserRoleProfile)
		}
		for _, role := range domain.AllRoles {
			if _, ok := result[userID][role]; !ok {
				result[userID][role] = domain.NewDefaultUserRoleProfile(userID, role)
			}
		}
	}

	return result, nil
}
