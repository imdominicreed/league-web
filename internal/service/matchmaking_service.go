package service

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

// MMR threshold constants for comfort-first matchmaking
const (
	InitialMmrThreshold   = 100
	MmrThresholdIncrement = 100
	MaxMmrThreshold       = 500
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
	UsedMmrThreshold    int
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
			ID:               uuid.New(),
			LobbyID:          lobbyID,
			OptionNumber:     i + 1,
			AlgorithmType:    opt.AlgorithmType,
			BlueTeamAvgMMR:   opt.BlueTeamMMR / 5,
			RedTeamAvgMMR:    opt.RedTeamMMR / 5,
			MMRDifference:    opt.MMRDifference,
			BalanceScore:     opt.BalanceScore,
			AvgBlueComfort:   opt.AvgBlueComfort,
			AvgRedComfort:    opt.AvgRedComfort,
			MaxLaneDiff:      opt.MaxLaneDiff,
			UsedMmrThreshold: opt.UsedMmrThreshold,
			CreatedAt:        time.Now(),
			Assignments:      make([]domain.MatchOptionAssignment, len(opt.Assignments)),
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

// GenerateMoreMatchOptions generates additional team options and appends them to existing ones
func (s *MatchmakingService) GenerateMoreMatchOptions(ctx context.Context, lobbyID uuid.UUID, players []*domain.LobbyPlayer, count int) ([]*domain.MatchOption, error) {
	if len(players) != 10 {
		return nil, ErrNotEnoughPlayers
	}

	// Get existing options to find max option number and existing compositions
	existingOptions, err := s.matchOptionRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	// Find the next option number and max threshold used
	maxOptionNum := 0
	maxThresholdUsed := InitialMmrThreshold
	existingHashes := make(map[string]bool)
	for _, opt := range existingOptions {
		if opt.OptionNumber > maxOptionNum {
			maxOptionNum = opt.OptionNumber
		}
		if opt.UsedMmrThreshold > maxThresholdUsed {
			maxThresholdUsed = opt.UsedMmrThreshold
		}
		// Build hash of existing compositions to avoid duplicates
		hash := s.buildOptionHash(opt)
		existingHashes[hash] = true
	}

	// Increase threshold by 100 for loading more teams
	newThreshold := maxThresholdUsed + MmrThresholdIncrement

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

	// Generate ALL possible options with the increased threshold
	// There are 252 possible team splits, so we request all of them
	options := s.generateBestOptionsWithMinThreshold(playerData, 252, newThreshold)

	// Filter out duplicates - use a fresh set to track within this batch too
	seenInBatch := make(map[string]bool)
	var newOptions []*GeneratedOption
	for _, opt := range options {
		hash := computeAssignmentHash(opt.Assignments)
		// Skip if already exists in DB or already added in this batch
		if existingHashes[hash] || seenInBatch[hash] {
			continue
		}
		seenInBatch[hash] = true
		newOptions = append(newOptions, opt)
		if len(newOptions) >= count {
			break
		}
	}

	// Save new options with incremented option numbers
	for i, opt := range newOptions {
		matchOption := &domain.MatchOption{
			ID:               uuid.New(),
			LobbyID:          lobbyID,
			OptionNumber:     maxOptionNum + i + 1,
			AlgorithmType:    opt.AlgorithmType,
			BlueTeamAvgMMR:   opt.BlueTeamMMR / 5,
			RedTeamAvgMMR:    opt.RedTeamMMR / 5,
			MMRDifference:    opt.MMRDifference,
			BalanceScore:     opt.BalanceScore,
			AvgBlueComfort:   opt.AvgBlueComfort,
			AvgRedComfort:    opt.AvgRedComfort,
			MaxLaneDiff:      opt.MaxLaneDiff,
			UsedMmrThreshold: opt.UsedMmrThreshold,
			CreatedAt:        time.Now(),
			Assignments:      make([]domain.MatchOptionAssignment, len(opt.Assignments)),
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
	}

	return s.matchOptionRepo.GetByLobbyID(ctx, lobbyID)
}

// buildOptionHash creates a hash from a domain.MatchOption for deduplication
func (s *MatchmakingService) buildOptionHash(opt *domain.MatchOption) string {
	assignments := make([]TeamAssignment, len(opt.Assignments))
	for i, a := range opt.Assignments {
		assignments[i] = TeamAssignment{
			UserID: a.UserID,
			Team:   a.Team,
			Role:   a.AssignedRole,
		}
	}
	return computeAssignmentHash(assignments)
}

// generateBestOptions generates the best team compositions using comfort-first algorithm
// with progressive MMR threshold expansion and randomization for variety
func (s *MatchmakingService) generateBestOptions(players []*PlayerData, count int) []*GeneratedOption {
	return s.generateBestOptionsWithMinThreshold(players, count, InitialMmrThreshold)
}

// generateBestOptionsWithMinThreshold generates options starting from a minimum threshold
func (s *MatchmakingService) generateBestOptionsWithMinThreshold(players []*PlayerData, count int, minThreshold int) []*GeneratedOption {
	// Seed random for variety on each call
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate team splits once (C(10,5) = 252 combinations)
	teamSplits := generateTeamSplits(len(players))

	// Pre-compute all base options with comfort-optimized role assignments
	baseOptions := s.computeAllBaseOptions(players, teamSplits)

	// Score all options by comfort (lower penalty = better)
	for _, opt := range baseOptions {
		opt.BalanceScore = s.scoreByComfort(opt)
		opt.AlgorithmType = domain.AlgorithmComfortFirst
	}

	// Shuffle first to randomize tie-breaking among options with identical scores
	rng.Shuffle(len(baseOptions), func(i, j int) {
		baseOptions[i], baseOptions[j] = baseOptions[j], baseOptions[i]
	})

	// Sort by comfort score (descending - higher score is better)
	// Use stable sort to preserve random order among equal scores
	sort.SliceStable(baseOptions, func(i, j int) bool {
		return baseOptions[i].BalanceScore > baseOptions[j].BalanceScore
	})

	// Try progressive MMR thresholds starting from minThreshold
	for threshold := minThreshold; threshold <= MaxMmrThreshold; threshold += MmrThresholdIncrement {
		var filtered []*GeneratedOption
		for _, opt := range baseOptions {
			if opt.MMRDifference <= threshold {
				optCopy := *opt
				optCopy.UsedMmrThreshold = threshold
				filtered = append(filtered, &optCopy)
			}
		}

		// Deduplicate
		filtered = deduplicateOptions(filtered)

		if len(filtered) >= 1 {
			// Return top N options
			if len(filtered) > count {
				filtered = filtered[:count]
			}
			return filtered
		}
	}

	// No threshold worked - return best available (best effort)
	deduped := deduplicateOptions(baseOptions)
	for _, opt := range deduped {
		opt.UsedMmrThreshold = -1 // Indicates best effort, no threshold met
	}
	if len(deduped) > count {
		deduped = deduped[:count]
	}
	return deduped
}

// scoreByComfort scores an option based purely on comfort, with tiny MMR tiebreaker
func (s *MatchmakingService) scoreByComfort(opt *GeneratedOption) float64 {
	var totalComfortPenalty float64
	for _, a := range opt.Assignments {
		totalComfortPenalty += comfortPenalty(a.ComfortRating)
	}

	// Higher score is better
	// Start at 100, subtract comfort penalty
	// Add tiny MMR tiebreaker (smaller diff = slightly higher score)
	score := 100.0
	score -= totalComfortPenalty * 1.0               // Primary: comfort
	score -= float64(opt.MMRDifference) / 10000.0    // Tiny tiebreaker for identical comfort
	return max(0, score)
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

			// Comfort-first scoring: prioritize minimizing comfort penalty
			// Use tiny MMR tiebreaker for identical comfort scores
			score := 100.0
			score -= totalPenalty * 1.0              // Primary: comfort penalty
			score -= float64(mmrDiff) / 10000.0      // Tiny tiebreaker for identical comfort

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
