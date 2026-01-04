package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

type MatchmakingService struct {
	profileRepo     repository.UserRoleProfileRepository
	matchOptionRepo repository.MatchOptionRepository
	lobbyRepo       repository.LobbyRepository
}

func NewMatchmakingService(
	profileRepo repository.UserRoleProfileRepository,
	matchOptionRepo repository.MatchOptionRepository,
	lobbyRepo repository.LobbyRepository,
) *MatchmakingService {
	return &MatchmakingService{
		profileRepo:     profileRepo,
		matchOptionRepo: matchOptionRepo,
		lobbyRepo:       lobbyRepo,
	}
}

// PlayerData holds all profile data for a player
type PlayerData struct {
	UserID       uuid.UUID
	DisplayName  string
	RoleProfiles map[domain.Role]*domain.UserRoleProfile
}

// TeamAssignment represents a player's assignment to a team and role
type TeamAssignment struct {
	UserID        uuid.UUID
	Team          domain.Side
	Role          domain.Role
	RoleMMR       int
	ComfortRating int
}

// GeneratedOption represents a generated match option before saving
type GeneratedOption struct {
	Assignments    []TeamAssignment
	BlueTeamMMR    int
	RedTeamMMR     int
	MMRDifference  int
	BalanceScore   float64
	AvgBlueComfort float64
	AvgRedComfort  float64
}

// GenerateMatchOptions generates balanced team options for a lobby
func (s *MatchmakingService) GenerateMatchOptions(ctx context.Context, lobbyID uuid.UUID, players []*domain.LobbyPlayer, count int) ([]*domain.MatchOption, error) {
	if len(players) != 10 {
		return nil, ErrNotEnoughPlayers
	}

	// Load role profiles for all players
	userIDs := make([]uuid.UUID, len(players))
	displayNames := make(map[uuid.UUID]string)
	for i, p := range players {
		userIDs[i] = p.UserID
		if p.User != nil {
			displayNames[p.UserID] = p.User.DisplayName
		}
	}

	profilesByUser, err := s.profileRepo.GetByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// Build player data
	playerData := make([]*PlayerData, len(players))
	for i, p := range players {
		pd := &PlayerData{
			UserID:       p.UserID,
			DisplayName:  displayNames[p.UserID],
			RoleProfiles: make(map[domain.Role]*domain.UserRoleProfile),
		}

		profiles := profilesByUser[p.UserID]
		for _, profile := range profiles {
			pd.RoleProfiles[profile.Role] = profile
		}

		// Fill missing roles with defaults
		for _, role := range domain.AllRoles {
			if _, ok := pd.RoleProfiles[role]; !ok {
				pd.RoleProfiles[role] = domain.NewDefaultUserRoleProfile(p.UserID, role)
			}
		}

		playerData[i] = pd
	}

	// Generate all possible team combinations and find the best ones
	options := s.generateBestOptions(playerData, count)

	// Delete existing options for this lobby
	if err := s.matchOptionRepo.DeleteByLobbyID(ctx, lobbyID); err != nil {
		return nil, err
	}

	// Save new options
	savedOptions := make([]*domain.MatchOption, len(options))
	for i, opt := range options {
		matchOption := &domain.MatchOption{
			ID:             uuid.New(),
			LobbyID:        lobbyID,
			OptionNumber:   i + 1,
			BlueTeamAvgMMR: opt.BlueTeamMMR / 5,
			RedTeamAvgMMR:  opt.RedTeamMMR / 5,
			MMRDifference:  opt.MMRDifference,
			BalanceScore:   opt.BalanceScore,
			CreatedAt:      time.Now(),
			Assignments:    make([]domain.MatchOptionAssignment, len(opt.Assignments)),
		}

		for j, a := range opt.Assignments {
			matchOption.Assignments[j] = domain.MatchOptionAssignment{
				ID:            uuid.New(),
				MatchOptionID: matchOption.ID,
				UserID:        a.UserID,
				Team:          a.Team,
				AssignedRole:  a.Role,
				RoleMMR:       a.RoleMMR,
				ComfortRating: a.ComfortRating,
			}
		}

		if err := s.matchOptionRepo.Create(ctx, matchOption); err != nil {
			return nil, err
		}
		savedOptions[i] = matchOption
	}

	// Update lobby status
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}
	lobby.Status = domain.LobbyStatusMatchmaking
	if err := s.lobbyRepo.Update(ctx, lobby); err != nil {
		return nil, err
	}

	return s.matchOptionRepo.GetByLobbyID(ctx, lobbyID)
}

// generateBestOptions generates the best N team compositions
func (s *MatchmakingService) generateBestOptions(players []*PlayerData, count int) []*GeneratedOption {
	// Generate team splits (C(10,5) = 252 combinations)
	teamSplits := generateTeamSplits(len(players))

	var allOptions []*GeneratedOption

	for _, split := range teamSplits {
		blueTeam := make([]*PlayerData, 5)
		redTeam := make([]*PlayerData, 5)

		blueIdx, redIdx := 0, 0
		for i, inBlue := range split {
			if inBlue {
				blueTeam[blueIdx] = players[i]
				blueIdx++
			} else {
				redTeam[redIdx] = players[i]
				redIdx++
			}
		}

		// Find optimal role assignment for this team split
		option := s.findBestRoleAssignment(blueTeam, redTeam)
		if option != nil {
			allOptions = append(allOptions, option)
		}
	}

	// Sort by balance score (higher is better)
	sort.Slice(allOptions, func(i, j int) bool {
		return allOptions[i].BalanceScore > allOptions[j].BalanceScore
	})

	// Return top N options
	if len(allOptions) > count {
		allOptions = allOptions[:count]
	}

	return allOptions
}

// findBestRoleAssignment finds the optimal role assignment for two teams
func (s *MatchmakingService) findBestRoleAssignment(blueTeam, redTeam []*PlayerData) *GeneratedOption {
	// Try all role permutations for blue team (5! = 120)
	// Red team gets the remaining roles automatically
	roles := domain.AllRoles

	var bestOption *GeneratedOption
	bestScore := float64(-1000)

	// Generate all permutations of roles
	permutations := generatePermutations(5)

	for _, bluePerm := range permutations {
		// Calculate team MMRs and comfort
		blueMMR, blueComfortPenalty := 0, 0
		redMMR, redComfortPenalty := 0, 0

		blueAssignments := make([]TeamAssignment, 5)
		redAssignments := make([]TeamAssignment, 5)

		for i := 0; i < 5; i++ {
			blueRole := roles[bluePerm[i]]
			redRole := roles[i]

			blueProfile := blueTeam[i].RoleProfiles[blueRole]
			redProfile := redTeam[i].RoleProfiles[redRole]

			blueMMR += blueProfile.MMR
			redMMR += redProfile.MMR

			blueComfortPenalty += 5 - blueProfile.ComfortRating
			redComfortPenalty += 5 - redProfile.ComfortRating

			blueAssignments[i] = TeamAssignment{
				UserID:        blueTeam[i].UserID,
				Team:          domain.SideBlue,
				Role:          blueRole,
				RoleMMR:       blueProfile.MMR,
				ComfortRating: blueProfile.ComfortRating,
			}

			redAssignments[i] = TeamAssignment{
				UserID:        redTeam[i].UserID,
				Team:          domain.SideRed,
				Role:          redRole,
				RoleMMR:       redProfile.MMR,
				ComfortRating: redProfile.ComfortRating,
			}
		}

		mmrDiff := abs(blueMMR - redMMR)
		totalComfortPenalty := blueComfortPenalty + redComfortPenalty

		// Calculate balance score (0-100, higher is better)
		// - Lose points for MMR difference
		// - Lose points for comfort penalties
		score := 100.0
		score -= float64(mmrDiff) / 100.0        // -1 point per 100 MMR difference
		score -= float64(totalComfortPenalty) * 1.5 // -1.5 points per comfort deficit

		if score > bestScore {
			bestScore = score

			allAssignments := make([]TeamAssignment, 10)
			copy(allAssignments[:5], blueAssignments)
			copy(allAssignments[5:], redAssignments)

			bestOption = &GeneratedOption{
				Assignments:    allAssignments,
				BlueTeamMMR:    blueMMR,
				RedTeamMMR:     redMMR,
				MMRDifference:  mmrDiff,
				BalanceScore:   math.Max(0, score),
				AvgBlueComfort: float64(25-blueComfortPenalty) / 5.0,
				AvgRedComfort:  float64(25-redComfortPenalty) / 5.0,
			}
		}
	}

	return bestOption
}

// generateTeamSplits generates all C(10,5) team split combinations
func generateTeamSplits(n int) [][]bool {
	// We need exactly 5 true values (blue team) in a slice of 10
	var results [][]bool
	var generate func(pos, trueCount int, current []bool)

	generate = func(pos, trueCount int, current []bool) {
		remaining := n - pos
		neededTrue := 5 - trueCount
		neededFalse := 5 - (pos - trueCount)

		// Pruning: can't complete if not enough positions left
		if neededTrue > remaining || neededFalse > remaining {
			return
		}

		if pos == n {
			if trueCount == 5 {
				result := make([]bool, n)
				copy(result, current)
				results = append(results, result)
			}
			return
		}

		// Try adding true (blue team)
		if trueCount < 5 {
			current[pos] = true
			generate(pos+1, trueCount+1, current)
		}

		// Try adding false (red team)
		if pos-trueCount < 5 {
			current[pos] = false
			generate(pos+1, trueCount, current)
		}
	}

	generate(0, 0, make([]bool, n))
	return results
}

// generatePermutations generates all permutations of indices [0, n)
func generatePermutations(n int) [][]int {
	var results [][]int
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}

	var generate func(k int)
	generate = func(k int) {
		if k == 1 {
			result := make([]int, n)
			copy(result, perm)
			results = append(results, result)
			return
		}

		generate(k - 1)
		for i := 0; i < k-1; i++ {
			if k%2 == 0 {
				perm[i], perm[k-1] = perm[k-1], perm[i]
			} else {
				perm[0], perm[k-1] = perm[k-1], perm[0]
			}
			generate(k - 1)
		}
	}

	generate(n)
	return results
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
