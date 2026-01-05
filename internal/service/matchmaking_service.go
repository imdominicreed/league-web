package service

import (
	"context"
	"math"
	"sort"
	"strings"
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
	Assignments         []TeamAssignment
	BlueTeamMMR         int
	RedTeamMMR          int
	MMRDifference       int
	BalanceScore        float64
	AvgBlueComfort      float64
	AvgRedComfort       float64
	TotalComfortPenalty int
	MaxLaneDiff         int
	AlgorithmType       domain.AlgorithmType
}

// ScoringStrategy defines how to score a team assignment
type ScoringStrategy interface {
	Score(opt *GeneratedOption) float64
	AlgorithmType() domain.AlgorithmType
}

// MMRBalancedStrategy prioritizes minimizing team MMR difference
type MMRBalancedStrategy struct{}

func (s *MMRBalancedStrategy) AlgorithmType() domain.AlgorithmType {
	return domain.AlgorithmMMRBalanced
}

func (s *MMRBalancedStrategy) Score(opt *GeneratedOption) float64 {
	// Heavy weight on MMR difference, light on comfort
	// Calculate exponential comfort penalty from assignments
	var comfortPen float64
	for _, a := range opt.Assignments {
		comfortPen += comfortPenalty(a.ComfortRating)
	}

	score := 100.0
	score -= float64(opt.MMRDifference) / 50.0 // -2 points per 100 MMR diff
	score -= comfortPen * 0.15                 // Light comfort penalty (adjusted for exponential scale)
	return max(0, score)
}

// RoleComfortStrategy prioritizes players getting high-comfort roles
type RoleComfortStrategy struct{}

func (s *RoleComfortStrategy) AlgorithmType() domain.AlgorithmType {
	return domain.AlgorithmRoleComfort
}

func (s *RoleComfortStrategy) Score(opt *GeneratedOption) float64 {
	// Heavy weight on comfort, MMR barely matters
	// Calculate exponential comfort penalty from assignments
	var comfortPen float64
	for _, a := range opt.Assignments {
		comfortPen += comfortPenalty(a.ComfortRating)
	}

	score := 100.0
	score -= comfortPen * 1.5                  // Heavy comfort penalty (adjusted for exponential scale)
	score -= float64(opt.MMRDifference) / 500.0 // Light MMR penalty
	return max(0, score)
}

// HybridStrategy uses the current balanced approach
type HybridStrategy struct{}

func (s *HybridStrategy) AlgorithmType() domain.AlgorithmType {
	return domain.AlgorithmHybrid
}

func (s *HybridStrategy) Score(opt *GeneratedOption) float64 {
	// Balanced approach between MMR and comfort
	// Calculate exponential comfort penalty from assignments
	var comfortPen float64
	for _, a := range opt.Assignments {
		comfortPen += comfortPenalty(a.ComfortRating)
	}

	score := 100.0
	score -= float64(opt.MMRDifference) / 100.0 // -1 point per 100 MMR difference
	score -= comfortPen * 0.5                   // Balanced comfort penalty (adjusted for exponential scale)
	return max(0, score)
}

// LaneBalancedStrategy minimizes the worst lane matchup
type LaneBalancedStrategy struct{}

func (s *LaneBalancedStrategy) AlgorithmType() domain.AlgorithmType {
	return domain.AlgorithmLaneBalanced
}

func (s *LaneBalancedStrategy) Score(opt *GeneratedOption) float64 {
	// Calculate per-lane differentials
	laneDiffs := make(map[domain.Role]int)
	for _, role := range domain.AllRoles {
		var blueMMR, redMMR int
		for _, a := range opt.Assignments {
			if a.Role == role {
				if a.Team == domain.SideBlue {
					blueMMR = a.RoleMMR
				} else {
					redMMR = a.RoleMMR
				}
			}
		}
		laneDiffs[role] = abs(blueMMR - redMMR)
	}

	// Find max lane diff and sum
	maxLaneDiff := 0
	sumLaneDiff := 0
	for _, diff := range laneDiffs {
		sumLaneDiff += diff
		if diff > maxLaneDiff {
			maxLaneDiff = diff
		}
	}
	opt.MaxLaneDiff = maxLaneDiff

	// Calculate exponential comfort penalty from assignments
	var comfortPen float64
	for _, a := range opt.Assignments {
		comfortPen += comfortPenalty(a.ComfortRating)
	}

	// Primarily punishes the worst lane matchup
	score := 100.0
	score -= float64(maxLaneDiff) / 100.0 // -1 point per 100 MMR max lane diff
	score -= float64(sumLaneDiff) / 500.0 // Light penalty for total lane imbalance
	score -= comfortPen * 0.1             // Minimal comfort consideration (adjusted for exponential scale)
	return max(0, score)
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
			AlgorithmType:  opt.AlgorithmType,
			BlueTeamAvgMMR: opt.BlueTeamMMR / 5,
			RedTeamAvgMMR:  opt.RedTeamMMR / 5,
			MMRDifference:  opt.MMRDifference,
			BalanceScore:   opt.BalanceScore,
			AvgBlueComfort: opt.AvgBlueComfort,
			AvgRedComfort:  opt.AvgRedComfort,
			MaxLaneDiff:    opt.MaxLaneDiff,
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

// generateBestOptions generates the best team compositions using multiple algorithms
func (s *MatchmakingService) generateBestOptions(players []*PlayerData, count int) []*GeneratedOption {
	// Calculate MMR range to detect wide skill variance
	minMMR, maxMMR := s.calculateMMRRange(players)
	mmrRange := maxMMR - minMMR

	// Generate team splits once (C(10,5) = 252 combinations)
	teamSplits := generateTeamSplits(len(players))

	// Pre-compute all base options (without algorithm-specific scoring)
	baseOptions := s.computeAllBaseOptions(players, teamSplits)

	// Define all scoring strategies
	strategies := []ScoringStrategy{
		&MMRBalancedStrategy{},
		&RoleComfortStrategy{},
		&HybridStrategy{},
		&LaneBalancedStrategy{},
	}

	// Collect best 2 options from each algorithm
	var allOptions []*GeneratedOption
	optionsPerAlgorithm := 2

	for _, strategy := range strategies {
		bestFromStrategy := s.getBestOptionsForStrategy(baseOptions, strategy, optionsPerAlgorithm)
		allOptions = append(allOptions, bestFromStrategy...)
	}

	// Deduplicate identical team compositions (keep first occurrence)
	allOptions = deduplicateOptions(allOptions)

	// Choose primary sorting strategy based on skill variance
	// When MMR range is wide (>1000), lane matchups matter more
	var primaryStrategy ScoringStrategy
	if mmrRange > 1000 {
		primaryStrategy = &LaneBalancedStrategy{}
	} else {
		primaryStrategy = &HybridStrategy{}
	}

	// Sort by primary strategy for final ranking
	sort.Slice(allOptions, func(i, j int) bool {
		return primaryStrategy.Score(allOptions[i]) > primaryStrategy.Score(allOptions[j])
	})

	// Return top N options
	if len(allOptions) > count {
		allOptions = allOptions[:count]
	}

	return allOptions
}

// computeAllBaseOptions computes all possible team compositions with their base stats
func (s *MatchmakingService) computeAllBaseOptions(players []*PlayerData, teamSplits [][]bool) []*GeneratedOption {
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

	return allOptions
}

// getBestOptionsForStrategy scores all options with a strategy and returns top N
func (s *MatchmakingService) getBestOptionsForStrategy(options []*GeneratedOption, strategy ScoringStrategy, n int) []*GeneratedOption {
	// Create scored copies
	type scoredOption struct {
		option *GeneratedOption
		score  float64
	}

	scored := make([]scoredOption, len(options))
	for i, opt := range options {
		// Create a copy to avoid modifying the original
		optCopy := *opt
		score := strategy.Score(&optCopy)
		optCopy.BalanceScore = score
		optCopy.AlgorithmType = strategy.AlgorithmType()
		scored[i] = scoredOption{option: &optCopy, score: score}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Take top N
	result := make([]*GeneratedOption, 0, n)
	for i := 0; i < n && i < len(scored); i++ {
		result = append(result, scored[i].option)
	}

	return result
}

// deduplicateOptions removes duplicate team compositions, keeping the first occurrence
func deduplicateOptions(options []*GeneratedOption) []*GeneratedOption {
	seen := make(map[string]bool)
	var unique []*GeneratedOption

	for _, opt := range options {
		key := computeAssignmentHash(opt.Assignments)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, opt)
		}
	}

	return unique
}

// computeAssignmentHash creates a unique hash for a team composition
func computeAssignmentHash(assignments []TeamAssignment) string {
	// Sort by UserID and create a string representation
	sorted := make([]TeamAssignment, len(assignments))
	copy(sorted, assignments)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UserID.String() < sorted[j].UserID.String()
	})

	var parts []string
	for _, a := range sorted {
		parts = append(parts, a.UserID.String()+":"+string(a.Team)+":"+string(a.Role))
	}

	return strings.Join(parts, "|")
}

// findBestRoleAssignment finds the optimal role assignment for two teams
// Tries all 120 × 120 = 14,400 permutations for both teams
func (s *MatchmakingService) findBestRoleAssignment(blueTeam, redTeam []*PlayerData) *GeneratedOption {
	roles := domain.AllRoles

	var bestOption *GeneratedOption
	bestScore := float64(-1000)

	// Generate all permutations of roles (5! = 120)
	permutations := generatePermutations(5)

	// Try all permutations for BOTH teams (120 × 120 = 14,400)
	for _, bluePerm := range permutations {
		for _, redPerm := range permutations {
			// Calculate team MMRs and comfort
			blueMMR, redMMR := 0, 0
			var blueComfortPen, redComfortPen float64

			blueAssignments := make([]TeamAssignment, 5)
			redAssignments := make([]TeamAssignment, 5)

			for i := range 5 {
				blueRole := roles[bluePerm[i]]
				redRole := roles[redPerm[i]] // Now permuted!

				blueProfile := blueTeam[i].RoleProfiles[blueRole]
				redProfile := redTeam[i].RoleProfiles[redRole]

				blueMMR += blueProfile.MMR
				redMMR += redProfile.MMR

				blueComfortPen += comfortPenalty(blueProfile.ComfortRating)
				redComfortPen += comfortPenalty(redProfile.ComfortRating)

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
			totalPenalty := blueComfortPen + redComfortPen

			// Calculate balance score using hybrid formula with exponential comfort
			// - Lose points for MMR difference
			// - Lose points for comfort penalties (exponential scale)
			score := 100.0
			score -= float64(mmrDiff) / 100.0 // -1 point per 100 MMR difference
			score -= totalPenalty * 0.5       // Adjusted for exponential scale

			if score > bestScore {
				bestScore = score

				allAssignments := make([]TeamAssignment, 10)
				copy(allAssignments[:5], blueAssignments)
				copy(allAssignments[5:], redAssignments)

				// Calculate average comfort (convert from penalty back to rating)
				// Max penalty per player is 15, so max per team is 75
				avgBlueComfort := 5.0 - (blueComfortPen / 5.0)
				avgRedComfort := 5.0 - (redComfortPen / 5.0)

				bestOption = &GeneratedOption{
					Assignments:         allAssignments,
					BlueTeamMMR:         blueMMR,
					RedTeamMMR:          redMMR,
					MMRDifference:       mmrDiff,
					BalanceScore:        math.Max(0, score),
					AvgBlueComfort:      math.Max(1, math.Min(5, avgBlueComfort)),
					AvgRedComfort:       math.Max(1, math.Min(5, avgRedComfort)),
					TotalComfortPenalty: int(totalPenalty),
				}
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

// comfortPenalty calculates exponential penalty for low comfort ratings.
// This makes playing at comfort 1-2 much more heavily penalized than comfort 3-4.
// comfort 5 → 0, comfort 4 → 1, comfort 3 → 3, comfort 2 → 7, comfort 1 → 15
func comfortPenalty(comfort int) float64 {
	return math.Pow(2, float64(5-comfort)) - 1
}

// calculateMMRRange finds the min and max MMR across all players and roles
func (s *MatchmakingService) calculateMMRRange(players []*PlayerData) (minMMR, maxMMR int) {
	minMMR = math.MaxInt
	maxMMR = 0
	for _, p := range players {
		for _, profile := range p.RoleProfiles {
			if profile.MMR < minMMR {
				minMMR = profile.MMR
			}
			if profile.MMR > maxMMR {
				maxMMR = profile.MMR
			}
		}
	}
	return minMMR, maxMMR
}
