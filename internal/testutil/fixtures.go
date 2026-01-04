package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UserBuilder creates test users with a builder pattern
type UserBuilder struct {
	displayName string
	password    string
}

// NewUserBuilder creates a new UserBuilder with default values
func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		displayName: fmt.Sprintf("testuser_%s", uuid.New().String()[:8]),
		password:    "testpassword123",
	}
}

// WithDisplayName sets the display name
func (b *UserBuilder) WithDisplayName(name string) *UserBuilder {
	b.displayName = name
	return b
}

// WithPassword sets the password
func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	b.password = password
	return b
}

// Build creates the user in the database and returns the user with the raw password
func (b *UserBuilder) Build(t *testing.T, db *gorm.DB) (*domain.User, string) {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(b.password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		DisplayName:  b.displayName,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return user, b.password
}

// AuthResponse matches the API auth response
type AuthResponse struct {
	User struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// BuildAndAuthenticate creates a user via API and returns the user and access token
func (b *UserBuilder) BuildAndAuthenticate(t *testing.T, ts *TestServer) (*domain.User, string) {
	t.Helper()

	reqBody := map[string]string{
		"displayName": b.displayName,
		"password":    b.password,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.APIURL("/auth/register"), "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	userID, _ := uuid.Parse(authResp.User.ID)
	user := &domain.User{
		ID:          userID,
		DisplayName: authResp.User.DisplayName,
	}

	return user, authResp.AccessToken
}

// RoomBuilder creates test rooms with a builder pattern
type RoomBuilder struct {
	creator      *domain.User
	draftMode    domain.DraftMode
	timerSeconds int
	blueSide     *domain.User
	redSide      *domain.User
}

// NewRoomBuilder creates a new RoomBuilder with default values
func NewRoomBuilder() *RoomBuilder {
	return &RoomBuilder{
		draftMode:    domain.DraftModeProPlay,
		timerSeconds: 30,
	}
}

// WithCreator sets the room creator
func (b *RoomBuilder) WithCreator(user *domain.User) *RoomBuilder {
	b.creator = user
	return b
}

// WithDraftMode sets the draft mode
func (b *RoomBuilder) WithDraftMode(mode domain.DraftMode) *RoomBuilder {
	b.draftMode = mode
	return b
}

// WithTimerDuration sets the timer duration in seconds
func (b *RoomBuilder) WithTimerDuration(seconds int) *RoomBuilder {
	b.timerSeconds = seconds
	return b
}

// WithBlueSide sets the blue side player
func (b *RoomBuilder) WithBlueSide(user *domain.User) *RoomBuilder {
	b.blueSide = user
	return b
}

// WithRedSide sets the red side player
func (b *RoomBuilder) WithRedSide(user *domain.User) *RoomBuilder {
	b.redSide = user
	return b
}

// Build creates the room in the database
func (b *RoomBuilder) Build(t *testing.T, db *gorm.DB) *domain.Room {
	t.Helper()

	if b.creator == nil {
		user, _ := NewUserBuilder().Build(t, db)
		b.creator = user
	}

	room := &domain.Room{
		ID:                   uuid.New(),
		ShortCode:            generateShortCode(),
		CreatedBy:            b.creator.ID,
		DraftMode:            b.draftMode,
		Status:               domain.RoomStatusWaiting,
		TimerDurationSeconds: b.timerSeconds,
		CreatedAt:            time.Now(),
	}

	if b.blueSide != nil {
		room.BlueSideUserID = &b.blueSide.ID
	}
	if b.redSide != nil {
		room.RedSideUserID = &b.redSide.ID
	}

	if err := db.Create(room).Error; err != nil {
		t.Fatalf("failed to create room: %v", err)
	}

	// Also create initial draft state
	emptyJSON, _ := json.Marshal([]string{})
	draftState := &domain.DraftState{
		ID:           uuid.New(),
		RoomID:       room.ID,
		CurrentPhase: 0,
		BlueBans:     datatypes.JSON(emptyJSON),
		RedBans:      datatypes.JSON(emptyJSON),
		BluePicks:    datatypes.JSON(emptyJSON),
		RedPicks:     datatypes.JSON(emptyJSON),
		IsComplete:   false,
	}

	if err := db.Create(draftState).Error; err != nil {
		t.Fatalf("failed to create draft state: %v", err)
	}

	return room
}

// BuildWithHub creates the room in both the database and WebSocket hub
func (b *RoomBuilder) BuildWithHub(t *testing.T, ts *TestServer) *domain.Room {
	t.Helper()

	room := b.Build(t, ts.DB.DB)

	// Create room in WebSocket hub
	ts.Hub.CreateRoom(room.ID, room.ShortCode, room.TimerDurationSeconds*1000)

	return room
}

func generateShortCode() string {
	return uuid.New().String()[:6]
}

// ChampionBuilder creates test champions
type ChampionBuilder struct {
	id       string
	key      string
	name     string
	title    string
	imageURL string
	tags     []string
}

// NewChampionBuilder creates a new ChampionBuilder with default values
func NewChampionBuilder() *ChampionBuilder {
	id := fmt.Sprintf("Champion%d", time.Now().UnixNano()%10000)
	return &ChampionBuilder{
		id:       id,
		key:      id,
		name:     id,
		title:    "The Test Champion",
		imageURL: fmt.Sprintf("https://ddragon.leagueoflegends.com/cdn/14.1.1/img/champion/%s.png", id),
		tags:     []string{"Fighter"},
	}
}

// WithID sets the champion ID
func (b *ChampionBuilder) WithID(id string) *ChampionBuilder {
	b.id = id
	b.key = id
	b.name = id
	b.imageURL = fmt.Sprintf("https://ddragon.leagueoflegends.com/cdn/14.1.1/img/champion/%s.png", id)
	return b
}

// WithName sets the champion name
func (b *ChampionBuilder) WithName(name string) *ChampionBuilder {
	b.name = name
	return b
}

// WithTitle sets the champion title
func (b *ChampionBuilder) WithTitle(title string) *ChampionBuilder {
	b.title = title
	return b
}

// WithTags sets the champion tags
func (b *ChampionBuilder) WithTags(tags []string) *ChampionBuilder {
	b.tags = tags
	return b
}

// Build creates the champion in the database
func (b *ChampionBuilder) Build(t *testing.T, db *gorm.DB) *domain.Champion {
	t.Helper()

	tagsJSON, _ := json.Marshal(b.tags)
	champion := &domain.Champion{
		ID:           b.id,
		Key:          b.key,
		Name:         b.name,
		Title:        b.title,
		ImageURL:     b.imageURL,
		Tags:         datatypes.JSON(tagsJSON),
		LastSyncedAt: time.Now(),
	}

	if err := db.Create(champion).Error; err != nil {
		t.Fatalf("failed to create champion: %v", err)
	}

	return champion
}

// SeedChampions creates N test champions in the database
func SeedChampions(t *testing.T, db *gorm.DB, count int) []*domain.Champion {
	t.Helper()

	champions := make([]*domain.Champion, count)
	for i := 0; i < count; i++ {
		champions[i] = NewChampionBuilder().
			WithID(fmt.Sprintf("TestChampion%d", i)).
			WithName(fmt.Sprintf("Test Champion %d", i)).
			Build(t, db)
	}
	return champions
}

// SeedRealChampions creates champions with real LoL champion names for realistic testing
func SeedRealChampions(t *testing.T, db *gorm.DB) []*domain.Champion {
	t.Helper()

	championNames := []string{
		"Aatrox", "Ahri", "Akali", "Alistar", "Amumu",
		"Anivia", "Annie", "Ashe", "Azir", "Bard",
		"Blitzcrank", "Brand", "Braum", "Caitlyn", "Camille",
		"Cassiopeia", "Darius", "Diana", "DrMundo", "Draven",
		"Ekko", "Elise", "Evelynn", "Ezreal", "Fiora",
	}

	champions := make([]*domain.Champion, len(championNames))
	for i, name := range championNames {
		champions[i] = NewChampionBuilder().
			WithID(name).
			WithName(name).
			Build(t, db)
	}
	return champions
}

// CreateAuthenticatedRequest creates an HTTP request with auth token
func CreateAuthenticatedRequest(t *testing.T, method, url string, body interface{}, token string) *http.Request {
	t.Helper()

	var bodyReader *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req
}

// LobbyBuilder creates test lobbies with a builder pattern
type LobbyBuilder struct {
	creator      *domain.User
	draftMode    domain.DraftMode
	timerSeconds int
	status       domain.LobbyStatus
}

// NewLobbyBuilder creates a new LobbyBuilder with default values
func NewLobbyBuilder() *LobbyBuilder {
	return &LobbyBuilder{
		draftMode:    domain.DraftModeProPlay,
		timerSeconds: 30,
		status:       domain.LobbyStatusWaitingForPlayers,
	}
}

// WithCreator sets the lobby creator
func (b *LobbyBuilder) WithCreator(u *domain.User) *LobbyBuilder {
	b.creator = u
	return b
}

// WithDraftMode sets the draft mode
func (b *LobbyBuilder) WithDraftMode(m domain.DraftMode) *LobbyBuilder {
	b.draftMode = m
	return b
}

// WithTimerDuration sets the timer duration in seconds
func (b *LobbyBuilder) WithTimerDuration(s int) *LobbyBuilder {
	b.timerSeconds = s
	return b
}

// WithStatus sets the lobby status
func (b *LobbyBuilder) WithStatus(s domain.LobbyStatus) *LobbyBuilder {
	b.status = s
	return b
}

// Build creates the lobby in the database
func (b *LobbyBuilder) Build(t *testing.T, db *gorm.DB) *domain.Lobby {
	t.Helper()

	if b.creator == nil {
		user, _ := NewUserBuilder().Build(t, db)
		b.creator = user
	}

	lobby := &domain.Lobby{
		ID:                   uuid.New(),
		ShortCode:            generateShortCode(),
		CreatedBy:            b.creator.ID,
		Status:               b.status,
		DraftMode:            b.draftMode,
		TimerDurationSeconds: b.timerSeconds,
		CreatedAt:            time.Now(),
	}

	if err := db.Create(lobby).Error; err != nil {
		t.Fatalf("failed to create lobby: %v", err)
	}

	// Add creator as a lobby player
	lobbyPlayer := &domain.LobbyPlayer{
		ID:       uuid.New(),
		LobbyID:  lobby.ID,
		UserID:   b.creator.ID,
		IsReady:  false,
		JoinedAt: time.Now(),
	}

	if err := db.Create(lobbyPlayer).Error; err != nil {
		t.Fatalf("failed to create lobby player: %v", err)
	}

	// Reload lobby with players
	if err := db.Preload("Players").Preload("Players.User").First(lobby, "id = ?", lobby.ID).Error; err != nil {
		t.Fatalf("failed to reload lobby: %v", err)
	}

	return lobby
}

// BuildWithPlayers creates the lobby with additional players
func (b *LobbyBuilder) BuildWithPlayers(t *testing.T, db *gorm.DB, count int) (*domain.Lobby, []*domain.User) {
	t.Helper()

	lobby := b.Build(t, db)
	users := make([]*domain.User, count)

	// First user is the creator
	if err := db.First(&users[0], "id = ?", b.creator.ID).Error; err != nil {
		t.Fatalf("failed to load creator: %v", err)
	}
	users[0] = b.creator

	// Create additional players (count-1 since creator is already included)
	for i := 1; i < count; i++ {
		user, _ := NewUserBuilder().Build(t, db)
		users[i] = user

		lobbyPlayer := &domain.LobbyPlayer{
			ID:       uuid.New(),
			LobbyID:  lobby.ID,
			UserID:   user.ID,
			IsReady:  false,
			JoinedAt: time.Now(),
		}

		if err := db.Create(lobbyPlayer).Error; err != nil {
			t.Fatalf("failed to create lobby player: %v", err)
		}
	}

	// Reload lobby with all players
	if err := db.Preload("Players").Preload("Players.User").First(lobby, "id = ?", lobby.ID).Error; err != nil {
		t.Fatalf("failed to reload lobby: %v", err)
	}

	return lobby, users
}

// UserRoleProfileBuilder creates test user role profiles with a builder pattern
type UserRoleProfileBuilder struct {
	user          *domain.User
	role          domain.Role
	leagueRank    domain.LeagueRank
	mmr           int
	comfortRating int
}

// NewUserRoleProfileBuilder creates a new UserRoleProfileBuilder with default values
func NewUserRoleProfileBuilder() *UserRoleProfileBuilder {
	return &UserRoleProfileBuilder{
		role:          domain.RoleMid,
		leagueRank:    domain.RankGold4,
		mmr:           1600,
		comfortRating: 3,
	}
}

// WithUser sets the user for the profile
func (b *UserRoleProfileBuilder) WithUser(u *domain.User) *UserRoleProfileBuilder {
	b.user = u
	return b
}

// WithRole sets the role for the profile
func (b *UserRoleProfileBuilder) WithRole(r domain.Role) *UserRoleProfileBuilder {
	b.role = r
	return b
}

// WithRank sets the league rank and updates MMR accordingly
func (b *UserRoleProfileBuilder) WithRank(r domain.LeagueRank) *UserRoleProfileBuilder {
	b.leagueRank = r
	b.mmr = r.ToMMR()
	return b
}

// WithMMR sets the MMR directly
func (b *UserRoleProfileBuilder) WithMMR(m int) *UserRoleProfileBuilder {
	b.mmr = m
	return b
}

// WithComfortRating sets the comfort rating (1-5)
func (b *UserRoleProfileBuilder) WithComfortRating(c int) *UserRoleProfileBuilder {
	b.comfortRating = c
	return b
}

// Build creates the user role profile in the database
func (b *UserRoleProfileBuilder) Build(t *testing.T, db *gorm.DB) *domain.UserRoleProfile {
	t.Helper()

	if b.user == nil {
		user, _ := NewUserBuilder().Build(t, db)
		b.user = user
	}

	profile := &domain.UserRoleProfile{
		ID:            uuid.New(),
		UserID:        b.user.ID,
		Role:          b.role,
		LeagueRank:    b.leagueRank,
		MMR:           b.mmr,
		ComfortRating: b.comfortRating,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("failed to create user role profile: %v", err)
	}

	return profile
}

// SeedLobbyWith10Players creates a lobby with 10 users each having role profiles
func SeedLobbyWith10Players(t *testing.T, db *gorm.DB) (*domain.Lobby, []*domain.User) {
	t.Helper()

	lobby, users := NewLobbyBuilder().BuildWithPlayers(t, db, 10)

	// Create role profiles for all users
	comfortRatings := []int{5, 4, 3, 2, 1} // Primary role to least preferred
	for _, user := range users {
		for i, role := range domain.AllRoles {
			NewUserRoleProfileBuilder().
				WithUser(user).
				WithRole(role).
				WithRank(domain.RankGold4).
				WithComfortRating(comfortRatings[i]).
				Build(t, db)
		}
	}

	return lobby, users
}

// SeedLobbyWith10ReadyPlayers creates a lobby with 10 ready players
func SeedLobbyWith10ReadyPlayers(t *testing.T, db *gorm.DB) (*domain.Lobby, []*domain.User) {
	t.Helper()

	lobby, users := SeedLobbyWith10Players(t, db)

	// Set all players to ready
	if err := db.Model(&domain.LobbyPlayer{}).
		Where("lobby_id = ?", lobby.ID).
		Update("is_ready", true).Error; err != nil {
		t.Fatalf("failed to update lobby players to ready: %v", err)
	}

	// Reload lobby with updated players
	if err := db.Preload("Players").Preload("Players.User").First(lobby, "id = ?", lobby.ID).Error; err != nil {
		t.Fatalf("failed to reload lobby: %v", err)
	}

	return lobby, users
}

// CreateUserWithAllRoleProfiles creates a user with profiles for all 5 roles
func CreateUserWithAllRoleProfiles(t *testing.T, db *gorm.DB, baseMMR int) *domain.User {
	t.Helper()

	user, _ := NewUserBuilder().Build(t, db)

	// Create a profile for each role with varying comfort ratings
	comfortRatings := []int{5, 4, 3, 2, 1} // Primary role to least preferred

	for i, role := range domain.AllRoles {
		profile := &domain.UserRoleProfile{
			ID:            uuid.New(),
			UserID:        user.ID,
			Role:          role,
			LeagueRank:    domain.MMRToRank(baseMMR),
			MMR:           baseMMR,
			ComfortRating: comfortRatings[i],
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := db.Create(profile).Error; err != nil {
			t.Fatalf("failed to create user role profile: %v", err)
		}
	}

	return user
}
