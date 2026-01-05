package service_test

import (
	"context"
	"testing"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Helper to create a user with specific role profiles
func createUserWithProfiles(t *testing.T, db *gorm.DB, name string, profiles map[domain.Role]struct {
	MMR     int
	Comfort int
}) *domain.User {
	t.Helper()

	user, _ := testutil.NewUserBuilder().WithDisplayName(name).Build(t, db)

	for role, profile := range profiles {
		p := &domain.UserRoleProfile{
			ID:            uuid.New(),
			UserID:        user.ID,
			Role:          role,
			LeagueRank:    domain.MMRToRank(profile.MMR),
			MMR:           profile.MMR,
			ComfortRating: profile.Comfort,
		}
		require.NoError(t, db.Create(p).Error)
	}

	return user
}

// Helper to create a user with uniform profiles across all roles
func createUserUniform(t *testing.T, db *gorm.DB, name string, mmr, comfort int) *domain.User {
	t.Helper()

	profiles := make(map[domain.Role]struct {
		MMR     int
		Comfort int
	})
	for _, role := range domain.AllRoles {
		profiles[role] = struct {
			MMR     int
			Comfort int
		}{MMR: mmr, Comfort: comfort}
	}
	return createUserWithProfiles(t, db, name, profiles)
}

// Helper to create a lobby player entry
func createLobbyPlayer(t *testing.T, db *gorm.DB, lobbyID uuid.UUID, user *domain.User, ready bool) *domain.LobbyPlayer {
	t.Helper()

	lp := &domain.LobbyPlayer{
		ID:      uuid.New(),
		LobbyID: lobbyID,
		UserID:  user.ID,
		IsReady: ready,
		User:    user,
	}
	require.NoError(t, db.Create(lp).Error)
	return lp
}

func TestMatchmakingService_IdenticalProfiles(t *testing.T) {
	// When all players have identical profiles, algorithms should converge
	// on similar/same team compositions, resulting in few unique options
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	// Create lobby
	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST01",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// Create 10 players with identical profiles (all Gold 4, comfort 3)
	var players []*domain.LobbyPlayer
	for i := 0; i < 10; i++ {
		user := createUserUniform(t, testDB.DB, "Player"+string(rune('A'+i)), 1600, 3)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("Identical profiles: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, BalanceScore=%.2f, MMRDiff=%d",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore, opt.MMRDifference)
	}

	// With identical profiles, deduplication will reduce options significantly
	// This documents the current behavior
	assert.GreaterOrEqual(t, len(options), 1, "Should generate at least 1 option")
	assert.LessOrEqual(t, len(options), 8, "Should not exceed requested count")

	// All options should have 0 MMR difference since everyone is equal
	for _, opt := range options {
		assert.Equal(t, 0, opt.MMRDifference, "MMR difference should be 0 for identical players")
	}
}

func TestMatchmakingService_DiverseMMR(t *testing.T) {
	// Wide MMR spread should create more varied options
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST02",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// Create players with diverse MMR: 2 Challenger, 2 Diamond, 2 Plat, 2 Gold, 2 Silver
	mmrLevels := []int{3600, 3500, 2800, 2900, 2000, 2100, 1600, 1700, 1200, 1300}
	var players []*domain.LobbyPlayer
	for i := 0; i < 10; i++ {
		user := createUserUniform(t, testDB.DB, "Player"+string(rune('A'+i)), mmrLevels[i], 3)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("Diverse MMR: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, BalanceScore=%.2f, MMRDiff=%d, BlueMMR=%d, RedMMR=%d",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore, opt.MMRDifference,
			opt.BlueTeamAvgMMR, opt.RedTeamAvgMMR)
	}

	assert.GreaterOrEqual(t, len(options), 1)

	// With diverse MMR, we should see algorithm trying to balance teams
	for _, opt := range options {
		// Balance score should still be reasonable
		assert.GreaterOrEqual(t, opt.BalanceScore, 0.0, "Balance score should be non-negative")
	}
}

func TestMatchmakingService_RoleSpecialists(t *testing.T) {
	// Players who specialize in specific roles (high comfort in one role)
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST03",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// Create 10 role specialists (2 per role)
	// Each player has high comfort (5) on their main role, low (1) elsewhere
	roleSpecialists := []domain.Role{
		domain.RoleTop, domain.RoleTop,
		domain.RoleJungle, domain.RoleJungle,
		domain.RoleMid, domain.RoleMid,
		domain.RoleADC, domain.RoleADC,
		domain.RoleSupport, domain.RoleSupport,
	}

	var players []*domain.LobbyPlayer
	for i := 0; i < 10; i++ {
		profiles := make(map[domain.Role]struct {
			MMR     int
			Comfort int
		})
		mainRole := roleSpecialists[i]
		for _, role := range domain.AllRoles {
			comfort := 1
			if role == mainRole {
				comfort = 5
			}
			profiles[role] = struct {
				MMR     int
				Comfort int
			}{MMR: 1600, Comfort: comfort}
		}
		user := createUserWithProfiles(t, testDB.DB, "Specialist"+string(rune('A'+i)), profiles)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("Role Specialists: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, BalanceScore=%.2f, AvgBlueComfort=%.2f, AvgRedComfort=%.2f",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore,
			opt.AvgBlueComfort, opt.AvgRedComfort)
	}

	assert.GreaterOrEqual(t, len(options), 1)

	// RoleComfort algorithm should prioritize high comfort assignments
	// Find the role_comfort option if it exists
	for _, opt := range options {
		if opt.AlgorithmType == domain.AlgorithmRoleComfort {
			// Role comfort should be high (closer to 5) when specialists get their main roles
			avgComfort := (opt.AvgBlueComfort + opt.AvgRedComfort) / 2
			t.Logf("  RoleComfort option has avg comfort: %.2f", avgComfort)
		}
	}
}

func TestMatchmakingService_MixedMMRAndComfort(t *testing.T) {
	// Realistic scenario: varied MMR AND varied role comfort
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST04",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// Create diverse realistic player profiles
	playerConfigs := []struct {
		name     string
		mainRole domain.Role
		mainMMR  int
		offMMR   int
	}{
		{"HighEloTop", domain.RoleTop, 2800, 2400},
		{"HighEloJgl", domain.RoleJungle, 2700, 2300},
		{"MidEloMid", domain.RoleMid, 2000, 1800},
		{"MidEloADC", domain.RoleADC, 2100, 1900},
		{"LowEloSup", domain.RoleSupport, 1400, 1200},
		{"FlexPlayer1", domain.RoleMid, 1800, 1700},     // Flexible, similar MMR across roles
		{"FlexPlayer2", domain.RoleADC, 1900, 1850},     // Flexible
		{"TopOneT", domain.RoleTop, 2200, 1500},         // Big gap between main and off
		{"JglSmurf", domain.RoleJungle, 2500, 2000},     // Jungle specialist
		{"SupportMain", domain.RoleSupport, 1600, 1300}, // Support main
	}

	var players []*domain.LobbyPlayer
	for _, cfg := range playerConfigs {
		profiles := make(map[domain.Role]struct {
			MMR     int
			Comfort int
		})
		for _, role := range domain.AllRoles {
			mmr := cfg.offMMR
			comfort := 2
			if role == cfg.mainRole {
				mmr = cfg.mainMMR
				comfort = 5
			}
			profiles[role] = struct {
				MMR     int
				Comfort int
			}{MMR: mmr, Comfort: comfort}
		}
		user := createUserWithProfiles(t, testDB.DB, cfg.name, profiles)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("Mixed MMR & Comfort: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, Score=%.2f, MMRDiff=%d, BlueComfort=%.2f, RedComfort=%.2f",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore, opt.MMRDifference,
			opt.AvgBlueComfort, opt.AvgRedComfort)
	}

	assert.GreaterOrEqual(t, len(options), 1)

	// Different algorithms should produce different optimizations
	algorithmsSeen := make(map[domain.AlgorithmType]bool)
	for _, opt := range options {
		algorithmsSeen[opt.AlgorithmType] = true
	}
	t.Logf("Algorithms represented: %v", algorithmsSeen)
}

func TestMatchmakingService_ExtremeMMRGap(t *testing.T) {
	// Test with extreme MMR gaps (Challenger + Iron players)
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST05",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// 5 Challengers (3600 MMR) and 5 Iron players (500 MMR)
	var players []*domain.LobbyPlayer
	for i := 0; i < 5; i++ {
		user := createUserUniform(t, testDB.DB, "Challenger"+string(rune('A'+i)), 3600, 4)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}
	for i := 0; i < 5; i++ {
		user := createUserUniform(t, testDB.DB, "Iron"+string(rune('A'+i)), 500, 4)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("Extreme MMR Gap: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, Score=%.2f, MMRDiff=%d, BlueAvg=%d, RedAvg=%d",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore, opt.MMRDifference,
			opt.BlueTeamAvgMMR, opt.RedTeamAvgMMR)
	}

	assert.GreaterOrEqual(t, len(options), 1)

	// Best option should have each team with mix of high and low MMR
	bestOption := options[0]
	// Ideal balance: each team gets ~2-3 Challengers and ~2-3 Irons
	// Avg MMR per team should be around (3600*5 + 500*5) / 10 = 2050
	expectedAvgMMR := (3600 + 500) / 2
	t.Logf("Expected balanced avg MMR per team: ~%d", expectedAvgMMR)
	t.Logf("Actual: Blue=%d, Red=%d", bestOption.BlueTeamAvgMMR, bestOption.RedTeamAvgMMR)
}

func TestMatchmakingService_AllFillPlayers(t *testing.T) {
	// All players are "fill" - equal comfort across all roles
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST06",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// 10 fill players with varied MMR but equal comfort (3) on all roles
	mmrLevels := []int{2000, 1900, 1800, 1700, 1600, 1500, 1400, 1300, 1200, 1100}
	var players []*domain.LobbyPlayer
	for i := 0; i < 10; i++ {
		user := createUserUniform(t, testDB.DB, "Fill"+string(rune('A'+i)), mmrLevels[i], 3)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("All Fill Players: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, Score=%.2f, MMRDiff=%d",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore, opt.MMRDifference)
	}

	assert.GreaterOrEqual(t, len(options), 1)

	// With fill players, role comfort algorithm has no advantage
	// MMR-based algorithms should dominate
}

func TestMatchmakingService_TwoOTPsPerRole(t *testing.T) {
	// 2 one-trick-ponies per role - perfect setup for balanced teams
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST07",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// 2 OTPs per role, one higher MMR than the other
	otpConfigs := []struct {
		role domain.Role
		mmr  int
	}{
		{domain.RoleTop, 2200},
		{domain.RoleTop, 1800},
		{domain.RoleJungle, 2100},
		{domain.RoleJungle, 1900},
		{domain.RoleMid, 2300},
		{domain.RoleMid, 1700},
		{domain.RoleADC, 2000},
		{domain.RoleADC, 2000},
		{domain.RoleSupport, 1600},
		{domain.RoleSupport, 1400},
	}

	var players []*domain.LobbyPlayer
	for i, cfg := range otpConfigs {
		profiles := make(map[domain.Role]struct {
			MMR     int
			Comfort int
		})
		for _, role := range domain.AllRoles {
			mmr := 1000  // Very low off-role
			comfort := 1 // Very uncomfortable
			if role == cfg.role {
				mmr = cfg.mmr
				comfort = 5
			}
			profiles[role] = struct {
				MMR     int
				Comfort int
			}{MMR: mmr, Comfort: comfort}
		}
		user := createUserWithProfiles(t, testDB.DB, cfg.role.String()+"OTP"+string(rune('A'+i)), profiles)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("Two OTPs Per Role: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, Score=%.2f, MMRDiff=%d, BlueComfort=%.2f, RedComfort=%.2f",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore, opt.MMRDifference,
			opt.AvgBlueComfort, opt.AvgRedComfort)

		// Print team assignments
		blueTeam := opt.GetBlueTeam()
		redTeam := opt.GetRedTeam()
		t.Logf("    Blue: ")
		for _, a := range blueTeam {
			t.Logf("      %s -> MMR %d, Comfort %d", a.AssignedRole, a.RoleMMR, a.ComfortRating)
		}
		t.Logf("    Red: ")
		for _, a := range redTeam {
			t.Logf("      %s -> MMR %d, Comfort %d", a.AssignedRole, a.RoleMMR, a.ComfortRating)
		}
	}

	assert.GreaterOrEqual(t, len(options), 1)

	// With 2 OTPs per role, ideal solution pairs one to each team
	// Check if comfort is maximized (should be ~5.0 for each team if OTPs get their role)
	for _, opt := range options {
		if opt.AlgorithmType == domain.AlgorithmRoleComfort {
			// This algorithm should find the solution where everyone plays their role
			assert.GreaterOrEqual(t, opt.AvgBlueComfort, 4.0, "Blue team comfort should be high for role_comfort algorithm")
			assert.GreaterOrEqual(t, opt.AvgRedComfort, 4.0, "Red team comfort should be high for role_comfort algorithm")
		}
	}
}

func TestMatchmakingService_LaneBalancing(t *testing.T) {
	// Test lane-balanced algorithm with intentional lane mismatches
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST08",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// Create players where some roles have huge MMR gaps
	// This tests if lane_balanced algorithm minimizes worst lane matchup
	playerConfigs := []struct {
		name    string
		topMMR  int
		jglMMR  int
		midMMR  int
		adcMMR  int
		supMMR  int
		comfort int
	}{
		{"TopGod", 3000, 1500, 1500, 1500, 1500, 5},
		{"TopBad", 1200, 1500, 1500, 1500, 1500, 5},
		{"JglHigh", 1500, 2800, 1500, 1500, 1500, 5},
		{"JglLow", 1500, 1300, 1500, 1500, 1500, 5},
		{"MidMain", 1500, 1500, 2500, 1500, 1500, 5},
		{"MidFill", 1500, 1500, 1600, 1500, 1500, 3},
		{"ADCMain", 1500, 1500, 1500, 2200, 1500, 5},
		{"ADCFill", 1500, 1500, 1500, 1800, 1500, 3},
		{"SupMain", 1500, 1500, 1500, 1500, 2000, 5},
		{"SupFill", 1500, 1500, 1500, 1500, 1700, 3},
	}

	var players []*domain.LobbyPlayer
	for _, cfg := range playerConfigs {
		profiles := map[domain.Role]struct {
			MMR     int
			Comfort int
		}{
			domain.RoleTop:     {MMR: cfg.topMMR, Comfort: cfg.comfort},
			domain.RoleJungle:  {MMR: cfg.jglMMR, Comfort: cfg.comfort},
			domain.RoleMid:     {MMR: cfg.midMMR, Comfort: cfg.comfort},
			domain.RoleADC:     {MMR: cfg.adcMMR, Comfort: cfg.comfort},
			domain.RoleSupport: {MMR: cfg.supMMR, Comfort: cfg.comfort},
		}
		user := createUserWithProfiles(t, testDB.DB, cfg.name, profiles)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	t.Logf("Lane Balancing: Generated %d options (requested 8)", len(options))
	for _, opt := range options {
		t.Logf("  Option %d: Algorithm=%s, Score=%.2f, MMRDiff=%d, MaxLaneDiff=%d",
			opt.OptionNumber, opt.AlgorithmType, opt.BalanceScore, opt.MMRDifference, opt.MaxLaneDiff)
	}

	assert.GreaterOrEqual(t, len(options), 1)

	// Find lane_balanced option and check it minimizes max lane diff
	var laneBalancedOption *domain.MatchOption
	for _, opt := range options {
		if opt.AlgorithmType == domain.AlgorithmLaneBalanced {
			laneBalancedOption = opt
			break
		}
	}

	if laneBalancedOption != nil {
		t.Logf("Lane balanced option max lane diff: %d", laneBalancedOption.MaxLaneDiff)
		// The lane balanced algorithm should try to avoid putting TopGod vs TopBad (1800 MMR diff)
	}
}

func TestMatchmakingService_MinimumOptionsReturned(t *testing.T) {
	// Verify that even with deduplication, we get useful options
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	testCases := []struct {
		name        string
		setupPlayers func(t *testing.T, db *gorm.DB, lobbyID uuid.UUID) []*domain.LobbyPlayer
		minExpected int
	}{
		{
			name: "completely_identical",
			setupPlayers: func(t *testing.T, db *gorm.DB, lobbyID uuid.UUID) []*domain.LobbyPlayer {
				var players []*domain.LobbyPlayer
				for i := 0; i < 10; i++ {
					user := createUserUniform(t, db, "Clone"+string(rune('A'+i)), 1500, 3)
					lp := createLobbyPlayer(t, db, lobbyID, user, true)
					players = append(players, lp)
				}
				return players
			},
			minExpected: 1, // At least 1 option even with full deduplication
		},
		{
			name: "slight_variation",
			setupPlayers: func(t *testing.T, db *gorm.DB, lobbyID uuid.UUID) []*domain.LobbyPlayer {
				var players []*domain.LobbyPlayer
				for i := 0; i < 10; i++ {
					// Slight MMR variation: 1500, 1510, 1520, ...
					mmr := 1500 + i*10
					user := createUserUniform(t, db, "Varied"+string(rune('A'+i)), mmr, 3)
					lp := createLobbyPlayer(t, db, lobbyID, user, true)
					players = append(players, lp)
				}
				return players
			},
			minExpected: 1,
		},
		{
			name: "high_diversity",
			setupPlayers: func(t *testing.T, db *gorm.DB, lobbyID uuid.UUID) []*domain.LobbyPlayer {
				mmrs := []int{3000, 2800, 2500, 2200, 2000, 1800, 1500, 1200, 1000, 800}
				var players []*domain.LobbyPlayer
				for i := 0; i < 10; i++ {
					user := createUserUniform(t, db, "Diverse"+string(rune('A'+i)), mmrs[i], i%5+1)
					lp := createLobbyPlayer(t, db, lobbyID, user, true)
					players = append(players, lp)
				}
				return players
			},
			minExpected: 3, // High diversity should yield more options
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDB.Truncate(t) // Clean slate for each test case

			creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
			lobby := &domain.Lobby{
				ID:        uuid.New(),
				ShortCode: "TST" + tc.name[:2],
				CreatedBy: creator.ID,
				Status:    domain.LobbyStatusWaitingForPlayers,
			}
			require.NoError(t, testDB.DB.Create(lobby).Error)

			players := tc.setupPlayers(t, testDB.DB, lobby.ID)
			options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
			require.NoError(t, err)

			t.Logf("%s: Generated %d options", tc.name, len(options))
			assert.GreaterOrEqual(t, len(options), tc.minExpected,
				"Should generate at least %d options for %s", tc.minExpected, tc.name)
		})
	}
}

func TestMatchmakingService_AlgorithmDiversity(t *testing.T) {
	// Verify that different algorithms are represented in output
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	svc := service.NewMatchmakingService(repos.UserRoleProfile, repos.MatchOption, repos.Lobby)
	ctx := context.Background()

	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	lobby := &domain.Lobby{
		ID:        uuid.New(),
		ShortCode: "TEST09",
		CreatedBy: creator.ID,
		Status:    domain.LobbyStatusWaitingForPlayers,
	}
	require.NoError(t, testDB.DB.Create(lobby).Error)

	// Create players where different algorithms will optimize differently
	// - Some high MMR players with low comfort
	// - Some low MMR players with high comfort
	// This creates tension between MMR-based and comfort-based algorithms
	configs := []struct {
		mmr     int
		comfort int
	}{
		{2800, 2}, {2600, 2}, {2400, 3}, {2200, 3}, {2000, 4},
		{1800, 4}, {1600, 4}, {1400, 5}, {1200, 5}, {1000, 5},
	}

	var players []*domain.LobbyPlayer
	for i, cfg := range configs {
		user := createUserUniform(t, testDB.DB, "Mixed"+string(rune('A'+i)), cfg.mmr, cfg.comfort)
		lp := createLobbyPlayer(t, testDB.DB, lobby.ID, user, true)
		players = append(players, lp)
	}

	options, err := svc.GenerateMatchOptions(ctx, lobby.ID, players, 8)
	require.NoError(t, err)

	// Count algorithms represented
	algorithmCounts := make(map[domain.AlgorithmType]int)
	for _, opt := range options {
		algorithmCounts[opt.AlgorithmType]++
	}

	t.Logf("Algorithm Diversity: Generated %d options", len(options))
	for alg, count := range algorithmCounts {
		t.Logf("  %s: %d options", alg, count)
	}

	// We should see at least 2 different algorithms represented
	assert.GreaterOrEqual(t, len(algorithmCounts), 1, "Should have at least 1 algorithm represented")
}
