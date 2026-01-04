package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrLobbyNotFound      = errors.New("lobby not found")
	ErrLobbyFull          = errors.New("lobby is full")
	ErrAlreadyInLobby     = errors.New("user is already in lobby")
	ErrNotInLobby         = errors.New("user is not in lobby")
	ErrNotLobbyCreator    = errors.New("only lobby creator can perform this action")
	ErrNotEnoughPlayers   = errors.New("lobby needs 10 players")
	ErrPlayersNotReady    = errors.New("not all players are ready")
	ErrInvalidLobbyState  = errors.New("invalid lobby state for this action")
	ErrNoMatchOptions     = errors.New("no match options available")
	ErrInvalidMatchOption = errors.New("invalid match option")
)

type LobbyService struct {
	lobbyRepo       repository.LobbyRepository
	lobbyPlayerRepo repository.LobbyPlayerRepository
	matchOptionRepo repository.MatchOptionRepository
	profileRepo     repository.UserRoleProfileRepository
	roomPlayerRepo  repository.RoomPlayerRepository
	roomService     *RoomService
}

func NewLobbyService(
	lobbyRepo repository.LobbyRepository,
	lobbyPlayerRepo repository.LobbyPlayerRepository,
	matchOptionRepo repository.MatchOptionRepository,
	profileRepo repository.UserRoleProfileRepository,
	roomPlayerRepo repository.RoomPlayerRepository,
	roomService *RoomService,
) *LobbyService {
	return &LobbyService{
		lobbyRepo:       lobbyRepo,
		lobbyPlayerRepo: lobbyPlayerRepo,
		matchOptionRepo: matchOptionRepo,
		profileRepo:     profileRepo,
		roomPlayerRepo:  roomPlayerRepo,
		roomService:     roomService,
	}
}

type CreateLobbyInput struct {
	DraftMode            domain.DraftMode
	TimerDurationSeconds int
}

func (s *LobbyService) CreateLobby(ctx context.Context, creatorID uuid.UUID, input CreateLobbyInput) (*domain.Lobby, error) {
	shortCode := generateLobbyShortCode()

	timerDuration := input.TimerDurationSeconds
	if timerDuration <= 0 {
		timerDuration = 30
	}

	lobby := &domain.Lobby{
		ID:                   uuid.New(),
		ShortCode:            shortCode,
		CreatedBy:            creatorID,
		Status:               domain.LobbyStatusWaitingForPlayers,
		DraftMode:            input.DraftMode,
		TimerDurationSeconds: timerDuration,
		CreatedAt:            time.Now(),
	}

	if err := s.lobbyRepo.Create(ctx, lobby); err != nil {
		return nil, err
	}

	// Creator automatically joins the lobby
	player := &domain.LobbyPlayer{
		ID:       uuid.New(),
		LobbyID:  lobby.ID,
		UserID:   creatorID,
		IsReady:  false,
		JoinedAt: time.Now(),
	}

	if err := s.lobbyPlayerRepo.Create(ctx, player); err != nil {
		return nil, err
	}

	return s.lobbyRepo.GetByID(ctx, lobby.ID)
}

func (s *LobbyService) GetLobby(ctx context.Context, idOrCode string) (*domain.Lobby, error) {
	// Try parsing as UUID first
	if id, err := uuid.Parse(idOrCode); err == nil {
		return s.lobbyRepo.GetByID(ctx, id)
	}
	// Otherwise treat as short code
	return s.lobbyRepo.GetByShortCode(ctx, idOrCode)
}

func (s *LobbyService) JoinLobby(ctx context.Context, lobbyID, userID uuid.UUID) (*domain.LobbyPlayer, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	if lobby.Status != domain.LobbyStatusWaitingForPlayers {
		return nil, ErrInvalidLobbyState
	}

	// Check if already in lobby
	existing, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err == nil && existing != nil {
		return existing, nil // Already in lobby, return existing player
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Check if lobby is full
	count, err := s.lobbyPlayerRepo.CountByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}
	if count >= domain.MaxLobbyPlayers {
		return nil, ErrLobbyFull
	}

	player := &domain.LobbyPlayer{
		ID:       uuid.New(),
		LobbyID:  lobbyID,
		UserID:   userID,
		IsReady:  false,
		JoinedAt: time.Now(),
	}

	if err := s.lobbyPlayerRepo.Create(ctx, player); err != nil {
		return nil, err
	}

	return s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
}

func (s *LobbyService) LeaveLobby(ctx context.Context, lobbyID, userID uuid.UUID) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLobbyNotFound
		}
		return err
	}

	if lobby.Status != domain.LobbyStatusWaitingForPlayers {
		return ErrInvalidLobbyState
	}

	// Check if user is in lobby
	_, err = s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}

	return s.lobbyPlayerRepo.Delete(ctx, lobbyID, userID)
}

func (s *LobbyService) SetPlayerReady(ctx context.Context, lobbyID, userID uuid.UUID, ready bool) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLobbyNotFound
		}
		return err
	}

	if lobby.Status != domain.LobbyStatusWaitingForPlayers {
		return ErrInvalidLobbyState
	}

	player, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}

	player.IsReady = ready
	return s.lobbyPlayerRepo.Update(ctx, player)
}

func (s *LobbyService) GetMatchOptions(ctx context.Context, lobbyID uuid.UUID) ([]*domain.MatchOption, error) {
	return s.matchOptionRepo.GetByLobbyID(ctx, lobbyID)
}

func (s *LobbyService) SelectMatchOption(ctx context.Context, lobbyID uuid.UUID, optionNumber int, creatorID uuid.UUID) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLobbyNotFound
		}
		return err
	}

	if lobby.CreatedBy != creatorID {
		return ErrNotLobbyCreator
	}

	if lobby.Status != domain.LobbyStatusMatchmaking {
		return ErrInvalidLobbyState
	}

	// Verify the option exists
	option, err := s.matchOptionRepo.GetByLobbyIDAndNumber(ctx, lobbyID, optionNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidMatchOption
		}
		return err
	}

	// Update lobby players with team assignments
	assignments := make(map[uuid.UUID]struct {
		Team domain.Side
		Role domain.Role
	})
	for _, a := range option.Assignments {
		assignments[a.UserID] = struct {
			Team domain.Side
			Role domain.Role
		}{
			Team: a.Team,
			Role: a.AssignedRole,
		}
	}

	if err := s.lobbyPlayerRepo.UpdateTeamAssignments(ctx, lobbyID, assignments); err != nil {
		return err
	}

	// Update lobby status
	lobby.SelectedMatchOption = &optionNumber
	lobby.Status = domain.LobbyStatusTeamSelected
	return s.lobbyRepo.Update(ctx, lobby)
}

// GetPlayers returns all players in a lobby
func (s *LobbyService) GetPlayers(ctx context.Context, lobbyID uuid.UUID) ([]*domain.LobbyPlayer, error) {
	return s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
}

// StartDraft creates a Room from a lobby after team selection
func (s *LobbyService) StartDraft(ctx context.Context, lobbyID uuid.UUID, userID uuid.UUID) (*domain.Room, error) {
	// Get lobby with players
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	// Verify user is the lobby creator
	if lobby.CreatedBy != userID {
		return nil, ErrNotLobbyCreator
	}

	// Verify lobby status is team_selected
	if lobby.Status != domain.LobbyStatusTeamSelected {
		return nil, ErrInvalidLobbyState
	}

	// Get the selected match option with assignments
	if lobby.SelectedMatchOption == nil {
		return nil, ErrNoMatchOptions
	}
	option, err := s.matchOptionRepo.GetByLobbyIDAndNumber(ctx, lobbyID, *lobby.SelectedMatchOption)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidMatchOption
		}
		return nil, err
	}

	// Create the room
	room, err := s.roomService.CreateRoom(ctx, CreateRoomInput{
		CreatedBy:     userID,
		DraftMode:     lobby.DraftMode,
		TimerDuration: lobby.TimerDurationSeconds,
	})
	if err != nil {
		return nil, err
	}

	// Update the room with team draft fields
	room.IsTeamDraft = true
	room.LobbyID = &lobbyID

	// Find captains (first player per team in role order)
	blueCaptain := findCaptain(option.GetBlueTeam())
	redCaptain := findCaptain(option.GetRedTeam())

	if blueCaptain != nil {
		room.BlueSideUserID = &blueCaptain.UserID
	}
	if redCaptain != nil {
		room.RedSideUserID = &redCaptain.UserID
	}

	if err := s.roomService.roomRepo.Update(ctx, room); err != nil {
		return nil, err
	}

	// Create RoomPlayer entries for all 10 players
	var roomPlayers []*domain.RoomPlayer
	for _, assignment := range option.Assignments {
		isCaptain := false
		if blueCaptain != nil && assignment.UserID == blueCaptain.UserID {
			isCaptain = true
		}
		if redCaptain != nil && assignment.UserID == redCaptain.UserID {
			isCaptain = true
		}

		roomPlayer := &domain.RoomPlayer{
			ID:           uuid.New(),
			RoomID:       room.ID,
			UserID:       assignment.UserID,
			Team:         assignment.Team,
			AssignedRole: assignment.AssignedRole,
			IsCaptain:    isCaptain,
			IsReady:      false,
		}
		roomPlayers = append(roomPlayers, roomPlayer)
	}

	if err := s.roomPlayerRepo.CreateMany(ctx, roomPlayers); err != nil {
		return nil, err
	}

	// Update lobby status to drafting and set roomId
	now := time.Now()
	lobby.Status = domain.LobbyStatusDrafting
	lobby.RoomID = &room.ID
	lobby.StartedAt = &now
	if err := s.lobbyRepo.Update(ctx, lobby); err != nil {
		return nil, err
	}

	return room, nil
}

// findCaptain returns the first player (captain) from a team based on role order
func findCaptain(assignments []domain.MatchOptionAssignment) *domain.MatchOptionAssignment {
	if len(assignments) == 0 {
		return nil
	}
	// Role order: Top -> Jungle -> Mid -> ADC -> Support
	roleOrder := map[domain.Role]int{
		domain.RoleTop:     0,
		domain.RoleJungle:  1,
		domain.RoleMid:     2,
		domain.RoleADC:     3,
		domain.RoleSupport: 4,
	}

	captain := &assignments[0]
	for i := range assignments {
		if roleOrder[assignments[i].AssignedRole] < roleOrder[captain.AssignedRole] {
			captain = &assignments[i]
		}
	}
	return captain
}

func generateLobbyShortCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:8]
}
