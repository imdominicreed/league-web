package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	mathrand "math/rand"
	"strings"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrLobbyNotFound         = errors.New("lobby not found")
	ErrLobbyFull             = errors.New("lobby is full")
	ErrAlreadyInLobby        = errors.New("user is already in lobby")
	ErrNotInLobby            = errors.New("user is not in lobby")
	ErrNotLobbyCreator       = errors.New("only lobby creator can perform this action")
	ErrNotEnoughPlayers      = errors.New("lobby needs 10 players")
	ErrPlayersNotReady       = errors.New("not all players are ready")
	ErrInvalidLobbyState     = errors.New("invalid lobby state for this action")
	ErrNoMatchOptions        = errors.New("no match options available")
	ErrInvalidMatchOption    = errors.New("invalid match option")
	ErrNotCaptain            = errors.New("only captain can perform this action")
	ErrNotOnTeam             = errors.New("player is not on your team")
	ErrPendingActionExists   = errors.New("a pending action already exists")
	ErrPendingActionNotFound = errors.New("pending action not found")
	ErrAlreadyApproved       = errors.New("you have already approved this action")
	ErrCannotApproveOwn      = errors.New("cannot approve your own proposal")
	ErrActionExpired         = errors.New("pending action has expired")
	ErrPlayerNotFound        = errors.New("player not found")
	ErrCannotKickSelf        = errors.New("cannot kick yourself")
	ErrInvalidSwap           = errors.New("invalid swap request")
	ErrVotingNotEnabled      = errors.New("voting is not enabled for this lobby")
	ErrVotingNotActive       = errors.New("voting is not currently active")
	ErrInvalidVotingMode     = errors.New("invalid voting mode")
)

type LobbyService struct {
	lobbyRepo          repository.LobbyRepository
	lobbyPlayerRepo    repository.LobbyPlayerRepository
	matchOptionRepo    repository.MatchOptionRepository
	profileRepo        repository.UserRoleProfileRepository
	roomPlayerRepo     repository.RoomPlayerRepository
	pendingActionRepo  repository.PendingActionRepository
	voteRepo           repository.VoteRepository
	roomService        *RoomService
	matchmakingService *MatchmakingService
}

func NewLobbyService(
	lobbyRepo repository.LobbyRepository,
	lobbyPlayerRepo repository.LobbyPlayerRepository,
	matchOptionRepo repository.MatchOptionRepository,
	profileRepo repository.UserRoleProfileRepository,
	roomPlayerRepo repository.RoomPlayerRepository,
	pendingActionRepo repository.PendingActionRepository,
	voteRepo repository.VoteRepository,
	roomService *RoomService,
	matchmakingService *MatchmakingService,
) *LobbyService {
	return &LobbyService{
		lobbyRepo:          lobbyRepo,
		lobbyPlayerRepo:    lobbyPlayerRepo,
		matchOptionRepo:    matchOptionRepo,
		profileRepo:        profileRepo,
		roomPlayerRepo:     roomPlayerRepo,
		pendingActionRepo:  pendingActionRepo,
		voteRepo:           voteRepo,
		roomService:        roomService,
		matchmakingService: matchmakingService,
	}
}

type CreateLobbyInput struct {
	DraftMode            domain.DraftMode
	TimerDurationSeconds int
	VotingEnabled        bool
	VotingMode           domain.VotingMode
}

func (s *LobbyService) CreateLobby(ctx context.Context, creatorID uuid.UUID, input CreateLobbyInput) (*domain.Lobby, error) {
	shortCode := generateLobbyShortCode()

	timerDuration := input.TimerDurationSeconds
	if timerDuration <= 0 {
		timerDuration = 30
	}

	votingMode := input.VotingMode
	if votingMode == "" {
		votingMode = domain.VotingModeMajority
	}

	lobby := &domain.Lobby{
		ID:                   uuid.New(),
		ShortCode:            shortCode,
		CreatedBy:            creatorID,
		Status:               domain.LobbyStatusWaitingForPlayers,
		DraftMode:            input.DraftMode,
		TimerDurationSeconds: timerDuration,
		VotingEnabled:        input.VotingEnabled,
		VotingMode:           votingMode,
		CreatedAt:            time.Now(),
	}

	if err := s.lobbyRepo.Create(ctx, lobby); err != nil {
		return nil, err
	}

	// Creator automatically joins the lobby on blue side as captain
	blueSide := domain.SideBlue
	player := &domain.LobbyPlayer{
		ID:        uuid.New(),
		LobbyID:   lobby.ID,
		UserID:    creatorID,
		Team:      &blueSide,
		IsReady:   false,
		IsCaptain: true,
		JoinOrder: 0,
		JoinedAt:  time.Now(),
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
	// Otherwise treat as short code (normalize to uppercase)
	return s.lobbyRepo.GetByShortCode(ctx, strings.ToUpper(idOrCode))
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

	// Get all current players to determine side assignment
	players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	if len(players) >= domain.MaxLobbyPlayers {
		return nil, ErrLobbyFull
	}

	// Count players on each side
	blueCount, redCount := 0, 0
	hasBlueCaptain, hasRedCaptain := false, false
	for _, p := range players {
		if p.Team != nil {
			if *p.Team == domain.SideBlue {
				blueCount++
				if p.IsCaptain {
					hasBlueCaptain = true
				}
			} else if *p.Team == domain.SideRed {
				redCount++
				if p.IsCaptain {
					hasRedCaptain = true
				}
			}
		}
	}

	// Assign to side with fewer players, random if equal
	var side domain.Side
	if blueCount < redCount {
		side = domain.SideBlue
	} else if redCount < blueCount {
		side = domain.SideRed
	} else {
		// Random when equal
		if mathrand.Intn(2) == 0 {
			side = domain.SideBlue
		} else {
			side = domain.SideRed
		}
	}

	// First player on a side becomes captain
	isCaptain := (side == domain.SideBlue && !hasBlueCaptain) || (side == domain.SideRed && !hasRedCaptain)

	player := &domain.LobbyPlayer{
		ID:        uuid.New(),
		LobbyID:   lobbyID,
		UserID:    userID,
		Team:      &side,
		IsReady:   false,
		IsCaptain: isCaptain,
		JoinOrder: len(players),
		JoinedAt:  time.Now(),
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

	// Get the leaving player
	leavingPlayer, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}

	// If this player is captain, we need to promote a successor
	if leavingPlayer.IsCaptain && leavingPlayer.Team != nil {
		players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
		if err != nil {
			return err
		}

		// Find the next player on the same team by join order
		var successor *domain.LobbyPlayer
		for _, p := range players {
			if p.UserID == userID {
				continue // Skip the leaving player
			}
			if p.Team != nil && *p.Team == *leavingPlayer.Team {
				if successor == nil || p.JoinOrder < successor.JoinOrder {
					successor = p
				}
			}
		}

		// Promote successor if found
		if successor != nil {
			successor.IsCaptain = true
			if err := s.lobbyPlayerRepo.Update(ctx, successor); err != nil {
				return err
			}
		}
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

func (s *LobbyService) SelectMatchOption(ctx context.Context, lobbyID uuid.UUID, optionNumber int, userID uuid.UUID) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLobbyNotFound
		}
		return err
	}

	// Verify user is a captain
	player, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}
	if !player.IsCaptain {
		return ErrNotCaptain
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

	// Reset captains after team reassignment - one captain per team based on join order
	if err := s.resetCaptainsAfterTeamAssignment(ctx, lobbyID); err != nil {
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

	// Verify user is a captain
	player, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotInLobby
		}
		return nil, err
	}
	if !player.IsCaptain {
		return nil, ErrNotCaptain
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

	// Get lobby players - these have the CURRENT team/role assignments (including any swaps)
	lobbyPlayers, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	// Find lobby captains
	var blueCaptainID, redCaptainID *uuid.UUID
	for _, lp := range lobbyPlayers {
		if lp.IsCaptain && lp.Team != nil {
			if *lp.Team == domain.SideBlue {
				blueCaptainID = &lp.UserID
			} else if *lp.Team == domain.SideRed {
				redCaptainID = &lp.UserID
			}
		}
	}

	if blueCaptainID != nil {
		room.BlueSideUserID = blueCaptainID
	}
	if redCaptainID != nil {
		room.RedSideUserID = redCaptainID
	}

	if err := s.roomService.roomRepo.Update(ctx, room); err != nil {
		return nil, err
	}

	// Create RoomPlayer entries for all 10 players
	// Use lobbyPlayer data for team/role (which includes any swaps) instead of stale MatchOption assignments
	var roomPlayers []*domain.RoomPlayer
	for _, lp := range lobbyPlayers {
		// Skip players without team/role assignments (shouldn't happen but be safe)
		if lp.Team == nil || lp.AssignedRole == nil {
			continue
		}

		// Get display name from the user relation on the assignment
		displayName := ""
		for _, assignment := range option.Assignments {
			if assignment.UserID == lp.UserID && assignment.User != nil {
				displayName = assignment.User.DisplayName
				break
			}
		}

		roomPlayer := &domain.RoomPlayer{
			ID:           uuid.New(),
			RoomID:       room.ID,
			UserID:       lp.UserID,
			Team:         *lp.Team,         // Use current lobbyPlayer team (with swaps applied)
			AssignedRole: *lp.AssignedRole, // Use current lobbyPlayer role (with swaps applied)
			DisplayName:  displayName,
			IsCaptain:    lp.IsCaptain,     // Use current lobbyPlayer captain status (with swaps applied)
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

func generateLobbyShortCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes)[:8])
}

// ==================== Captain Management ====================

// resetCaptainsAfterTeamAssignment ensures exactly one captain per team after matchmaking
// The player with lowest join order on each team becomes captain
func (s *LobbyService) resetCaptainsAfterTeamAssignment(ctx context.Context, lobbyID uuid.UUID) error {
	players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return err
	}

	// Find first player (by join order) on each team
	var blueCaptain, redCaptain *domain.LobbyPlayer
	for _, p := range players {
		if p.Team == nil {
			continue
		}
		if *p.Team == domain.SideBlue {
			if blueCaptain == nil || p.JoinOrder < blueCaptain.JoinOrder {
				blueCaptain = p
			}
		} else if *p.Team == domain.SideRed {
			if redCaptain == nil || p.JoinOrder < redCaptain.JoinOrder {
				redCaptain = p
			}
		}
	}

	// Update all players - clear captain status then set the correct ones
	for _, p := range players {
		shouldBeCaptain := (blueCaptain != nil && p.ID == blueCaptain.ID) ||
			(redCaptain != nil && p.ID == redCaptain.ID)

		if p.IsCaptain != shouldBeCaptain {
			p.IsCaptain = shouldBeCaptain
			if err := s.lobbyPlayerRepo.Update(ctx, p); err != nil {
				return err
			}
		}
	}

	return nil
}

// TakeCaptain allows any player to take captain status from current captain
func (s *LobbyService) TakeCaptain(ctx context.Context, lobbyID, userID uuid.UUID) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLobbyNotFound
		}
		return err
	}

	// Allow taking captain in any lobby state before drafting starts
	if lobby.Status == domain.LobbyStatusDrafting || lobby.Status == domain.LobbyStatusCompleted {
		return ErrInvalidLobbyState
	}

	// Get the player who wants to take captain
	player, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}

	if player.Team == nil {
		return ErrNotOnTeam
	}

	if player.IsCaptain {
		return nil // Already captain
	}

	// Find current captain of the same team and remove their status
	players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return err
	}

	for _, p := range players {
		if p.Team != nil && *p.Team == *player.Team && p.IsCaptain {
			p.IsCaptain = false
			if err := s.lobbyPlayerRepo.Update(ctx, p); err != nil {
				return err
			}
			break
		}
	}

	// Make the requesting player captain
	player.IsCaptain = true
	return s.lobbyPlayerRepo.Update(ctx, player)
}

// PromoteCaptain allows captain to promote a teammate to captain
func (s *LobbyService) PromoteCaptain(ctx context.Context, lobbyID, captainID, targetUserID uuid.UUID) error {
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

	// Verify the caller is captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}

	if !captain.IsCaptain {
		return ErrNotCaptain
	}

	// Get the target player
	target, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPlayerNotFound
		}
		return err
	}

	// Verify target is on same team
	if target.Team == nil || captain.Team == nil || *target.Team != *captain.Team {
		return ErrNotOnTeam
	}

	// Swap captain status
	captain.IsCaptain = false
	target.IsCaptain = true

	if err := s.lobbyPlayerRepo.Update(ctx, captain); err != nil {
		return err
	}
	return s.lobbyPlayerRepo.Update(ctx, target)
}

// KickPlayer allows captain to remove a player from their team
func (s *LobbyService) KickPlayer(ctx context.Context, lobbyID, captainID, targetUserID uuid.UUID) error {
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

	// Verify the caller is captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}

	if !captain.IsCaptain {
		return ErrNotCaptain
	}

	if captainID == targetUserID {
		return ErrCannotKickSelf
	}

	// Get the target player
	target, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPlayerNotFound
		}
		return err
	}

	// Verify target is on same team
	if target.Team == nil || captain.Team == nil || *target.Team != *captain.Team {
		return ErrNotOnTeam
	}

	return s.lobbyPlayerRepo.Delete(ctx, lobbyID, targetUserID)
}

// ==================== Pending Actions ====================

// SwapRequest represents a request to swap players or roles
type SwapRequest struct {
	Player1ID uuid.UUID
	Player2ID uuid.UUID
	SwapType  string // "players" or "roles"
}

// ProposeSwap creates a pending action to swap players between teams or roles within a team
// Allowed in: waiting_for_players, matchmaking, team_selected (not during drafting or completed)
func (s *LobbyService) ProposeSwap(ctx context.Context, lobbyID, captainID uuid.UUID, req SwapRequest) (*domain.PendingAction, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	// Allow swaps before draft starts
	if lobby.Status != domain.LobbyStatusWaitingForPlayers &&
		lobby.Status != domain.LobbyStatusMatchmaking &&
		lobby.Status != domain.LobbyStatusTeamSelected {
		return nil, ErrInvalidLobbyState
	}

	// Check for existing pending action
	existing, _ := s.pendingActionRepo.GetPendingByLobbyID(ctx, lobbyID)
	if existing != nil && !existing.IsExpired() {
		return nil, ErrPendingActionExists
	}

	// Verify the caller is a captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotInLobby
		}
		return nil, err
	}
	if !captain.IsCaptain || captain.Team == nil {
		return nil, ErrNotCaptain
	}

	// Verify both players exist
	player1, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, req.Player1ID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	player2, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, req.Player2ID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}

	// Determine action type
	var actionType domain.PendingActionType
	if req.SwapType == "roles" {
		// Role swap: both players must be on same team
		if player1.Team == nil || player2.Team == nil || *player1.Team != *player2.Team {
			return nil, ErrInvalidSwap
		}
		actionType = domain.PendingActionSwapRoles
	} else {
		// Player swap: players must be on different teams
		if player1.Team == nil || player2.Team == nil || *player1.Team == *player2.Team {
			return nil, ErrInvalidSwap
		}
		actionType = domain.PendingActionSwapPlayers
	}

	action := domain.NewPendingAction(lobbyID, captainID, *captain.Team, actionType)
	action.Player1ID = &req.Player1ID
	action.Player2ID = &req.Player2ID

	if err := s.pendingActionRepo.Create(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// ProposeMatchmake creates a pending action to run matchmaking
func (s *LobbyService) ProposeMatchmake(ctx context.Context, lobbyID, captainID uuid.UUID) (*domain.PendingAction, error) {
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

	// Check for existing pending action
	existing, _ := s.pendingActionRepo.GetPendingByLobbyID(ctx, lobbyID)
	if existing != nil && !existing.IsExpired() {
		return nil, ErrPendingActionExists
	}

	// Verify 10 players
	if !lobby.IsFull() {
		return nil, ErrNotEnoughPlayers
	}

	// Verify the caller is a captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		return nil, err
	}
	if !captain.IsCaptain || captain.Team == nil {
		return nil, ErrNotCaptain
	}

	action := domain.NewPendingAction(lobbyID, captainID, *captain.Team, domain.PendingActionMatchmake)

	if err := s.pendingActionRepo.Create(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// ProposeStartDraft creates a pending action to start the draft
func (s *LobbyService) ProposeStartDraft(ctx context.Context, lobbyID, captainID uuid.UUID) (*domain.PendingAction, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	if lobby.Status != domain.LobbyStatusTeamSelected {
		return nil, ErrInvalidLobbyState
	}

	// Check for existing pending action
	existing, _ := s.pendingActionRepo.GetPendingByLobbyID(ctx, lobbyID)
	if existing != nil && !existing.IsExpired() {
		return nil, ErrPendingActionExists
	}

	// Verify the caller is a captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		return nil, err
	}
	if !captain.IsCaptain || captain.Team == nil {
		return nil, ErrNotCaptain
	}

	action := domain.NewPendingAction(lobbyID, captainID, *captain.Team, domain.PendingActionStartDraft)

	if err := s.pendingActionRepo.Create(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// ProposeSelectOption creates a pending action to select a match option
func (s *LobbyService) ProposeSelectOption(ctx context.Context, lobbyID, captainID uuid.UUID, optionNumber int) (*domain.PendingAction, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	if lobby.Status != domain.LobbyStatusMatchmaking {
		return nil, ErrInvalidLobbyState
	}

	// Verify the option exists
	_, err = s.matchOptionRepo.GetByLobbyIDAndNumber(ctx, lobbyID, optionNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidMatchOption
		}
		return nil, err
	}

	// Check for existing pending action
	existing, _ := s.pendingActionRepo.GetPendingByLobbyID(ctx, lobbyID)
	if existing != nil && !existing.IsExpired() {
		return nil, ErrPendingActionExists
	}

	// Verify the caller is a captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		return nil, err
	}
	if !captain.IsCaptain || captain.Team == nil {
		return nil, ErrNotCaptain
	}

	action := domain.NewPendingAction(lobbyID, captainID, *captain.Team, domain.PendingActionSelectOption)
	action.MatchOptionNum = &optionNumber

	if err := s.pendingActionRepo.Create(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// ApprovePendingAction approves a pending action by the other captain
func (s *LobbyService) ApprovePendingAction(ctx context.Context, lobbyID, captainID, actionID uuid.UUID) error {
	action, err := s.pendingActionRepo.GetByID(ctx, actionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPendingActionNotFound
		}
		return err
	}

	if action.LobbyID != lobbyID {
		return ErrPendingActionNotFound
	}

	if action.Status != domain.PendingStatusPending {
		return ErrInvalidLobbyState
	}

	if action.IsExpired() {
		action.Status = domain.PendingStatusExpired
		s.pendingActionRepo.Update(ctx, action)
		return ErrActionExpired
	}

	// Verify the caller is a captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		return err
	}
	if !captain.IsCaptain || captain.Team == nil {
		return ErrNotCaptain
	}

	// Check if already approved by this side
	if *captain.Team == domain.SideBlue && action.ApprovedByBlue {
		return ErrAlreadyApproved
	}
	if *captain.Team == domain.SideRed && action.ApprovedByRed {
		return ErrAlreadyApproved
	}

	// Mark approval
	if *captain.Team == domain.SideBlue {
		action.ApprovedByBlue = true
	} else {
		action.ApprovedByRed = true
	}

	// If fully approved, execute the action
	if action.IsFullyApproved() {
		if err := s.executePendingAction(ctx, action); err != nil {
			return err
		}
		action.Status = domain.PendingStatusApproved
	}

	return s.pendingActionRepo.Update(ctx, action)
}

// CancelPendingAction cancels a pending action
func (s *LobbyService) CancelPendingAction(ctx context.Context, lobbyID, captainID, actionID uuid.UUID) error {
	action, err := s.pendingActionRepo.GetByID(ctx, actionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPendingActionNotFound
		}
		return err
	}

	if action.LobbyID != lobbyID {
		return ErrPendingActionNotFound
	}

	if action.Status != domain.PendingStatusPending {
		return ErrInvalidLobbyState
	}

	// Verify the caller is a captain
	captain, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, captainID)
	if err != nil {
		return err
	}
	if !captain.IsCaptain {
		return ErrNotCaptain
	}

	action.Status = domain.PendingStatusCancelled
	return s.pendingActionRepo.Update(ctx, action)
}

// GetPendingAction returns the current pending action for a lobby
func (s *LobbyService) GetPendingAction(ctx context.Context, lobbyID uuid.UUID) (*domain.PendingAction, error) {
	action, err := s.pendingActionRepo.GetPendingByLobbyID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// Check if expired
	if action.IsExpired() {
		action.Status = domain.PendingStatusExpired
		s.pendingActionRepo.Update(ctx, action)
		return nil, nil
	}

	return action, nil
}

// executePendingAction performs the action after both captains approve
func (s *LobbyService) executePendingAction(ctx context.Context, action *domain.PendingAction) error {
	switch action.ActionType {
	case domain.PendingActionSwapPlayers:
		return s.executeSwapPlayers(ctx, action)
	case domain.PendingActionSwapRoles:
		return s.executeSwapRoles(ctx, action)
	case domain.PendingActionMatchmake:
		return s.executeMatchmaking(ctx, action.LobbyID)
	case domain.PendingActionSelectOption:
		if action.MatchOptionNum == nil {
			return ErrInvalidMatchOption
		}
		return s.applyMatchOption(ctx, action.LobbyID, *action.MatchOptionNum)
	case domain.PendingActionStartDraft:
		_, err := s.executeStartDraft(ctx, action.LobbyID)
		return err
	default:
		return ErrInvalidLobbyState
	}
}

// executeMatchmaking runs the matchmaking algorithm and updates lobby status
func (s *LobbyService) executeMatchmaking(ctx context.Context, lobbyID uuid.UUID) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		return err
	}

	if lobby.Status != domain.LobbyStatusWaitingForPlayers {
		return ErrInvalidLobbyState
	}

	// Get players and verify readiness
	players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return err
	}

	if len(players) != 10 {
		return ErrNotEnoughPlayers
	}

	// Generate match options
	_, err = s.matchmakingService.GenerateMatchOptions(ctx, lobbyID, players, 8)
	if err != nil {
		return err
	}

	// Update lobby status to matchmaking
	lobby.Status = domain.LobbyStatusMatchmaking
	return s.lobbyRepo.Update(ctx, lobby)
}

// applyMatchOption applies the selected match option to the lobby
func (s *LobbyService) applyMatchOption(ctx context.Context, lobbyID uuid.UUID, optionNumber int) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		return err
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

	// Reset captains after team reassignment - one captain per team based on join order
	if err := s.resetCaptainsAfterTeamAssignment(ctx, lobbyID); err != nil {
		return err
	}

	// Update lobby status
	lobby.SelectedMatchOption = &optionNumber
	lobby.Status = domain.LobbyStatusTeamSelected
	return s.lobbyRepo.Update(ctx, lobby)
}

// executeStartDraft creates the draft room when start_draft pending action is approved
func (s *LobbyService) executeStartDraft(ctx context.Context, lobbyID uuid.UUID) (*domain.Room, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

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

	// Get lobby players - these have the CURRENT team/role assignments (including any swaps)
	lobbyPlayers, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	// Build a map of userID -> lobbyPlayer for quick lookup
	lobbyPlayerMap := make(map[uuid.UUID]*domain.LobbyPlayer)
	for _, lp := range lobbyPlayers {
		lobbyPlayerMap[lp.UserID] = lp
	}

	// Find lobby captains and use blue captain as creator
	var blueCaptainID, redCaptainID *uuid.UUID
	var creatorID uuid.UUID
	for _, lp := range lobbyPlayers {
		if lp.IsCaptain && lp.Team != nil {
			if *lp.Team == domain.SideBlue {
				blueCaptainID = &lp.UserID
				creatorID = lp.UserID
			} else if *lp.Team == domain.SideRed {
				redCaptainID = &lp.UserID
			}
		}
	}

	// Create the room
	room, err := s.roomService.CreateRoom(ctx, CreateRoomInput{
		CreatedBy:     creatorID,
		DraftMode:     lobby.DraftMode,
		TimerDuration: lobby.TimerDurationSeconds,
	})
	if err != nil {
		return nil, err
	}

	// Update the room with team draft fields
	room.IsTeamDraft = true
	room.LobbyID = &lobbyID

	if blueCaptainID != nil {
		room.BlueSideUserID = blueCaptainID
	}
	if redCaptainID != nil {
		room.RedSideUserID = redCaptainID
	}

	if err := s.roomService.roomRepo.Update(ctx, room); err != nil {
		return nil, err
	}

	// Create RoomPlayer entries for all 10 players
	// Use lobbyPlayer data for team/role (which includes any swaps) instead of stale MatchOption assignments
	var roomPlayers []*domain.RoomPlayer
	for _, lp := range lobbyPlayers {
		// Skip players without team/role assignments (shouldn't happen but be safe)
		if lp.Team == nil || lp.AssignedRole == nil {
			continue
		}

		// Get display name from the user relation on the assignment
		displayName := ""
		for _, assignment := range option.Assignments {
			if assignment.UserID == lp.UserID && assignment.User != nil {
				displayName = assignment.User.DisplayName
				break
			}
		}

		roomPlayer := &domain.RoomPlayer{
			ID:           uuid.New(),
			RoomID:       room.ID,
			UserID:       lp.UserID,
			Team:         *lp.Team,         // Use current lobbyPlayer team (with swaps applied)
			AssignedRole: *lp.AssignedRole, // Use current lobbyPlayer role (with swaps applied)
			DisplayName:  displayName,
			IsCaptain:    lp.IsCaptain,     // Use current lobbyPlayer captain status (with swaps applied)
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

func (s *LobbyService) executeSwapPlayers(ctx context.Context, action *domain.PendingAction) error {
	if action.Player1ID == nil || action.Player2ID == nil {
		return ErrInvalidSwap
	}

	player1, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, action.LobbyID, *action.Player1ID)
	if err != nil {
		return err
	}
	player2, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, action.LobbyID, *action.Player2ID)
	if err != nil {
		return err
	}

	// Swap teams, roles, and captain status together
	// (captain status stays with the team position, not the person)
	// (roles are swapped so the team compositions stay balanced)
	player1.Team, player2.Team = player2.Team, player1.Team
	player1.AssignedRole, player2.AssignedRole = player2.AssignedRole, player1.AssignedRole
	player1.IsCaptain, player2.IsCaptain = player2.IsCaptain, player1.IsCaptain

	if err := s.lobbyPlayerRepo.Update(ctx, player1); err != nil {
		return err
	}
	return s.lobbyPlayerRepo.Update(ctx, player2)
}

func (s *LobbyService) executeSwapRoles(ctx context.Context, action *domain.PendingAction) error {
	if action.Player1ID == nil || action.Player2ID == nil {
		return ErrInvalidSwap
	}

	player1, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, action.LobbyID, *action.Player1ID)
	if err != nil {
		return err
	}
	player2, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, action.LobbyID, *action.Player2ID)
	if err != nil {
		return err
	}

	// Swap roles
	player1.AssignedRole, player2.AssignedRole = player2.AssignedRole, player1.AssignedRole

	if err := s.lobbyPlayerRepo.Update(ctx, player1); err != nil {
		return err
	}
	return s.lobbyPlayerRepo.Update(ctx, player2)
}

// ==================== Team Stats ====================

// TeamStats represents the current team balance stats
type TeamStats struct {
	BlueTeamAvgMMR int                `json:"blueTeamAvgMmr"`
	RedTeamAvgMMR  int                `json:"redTeamAvgMmr"`
	MMRDifference  int                `json:"mmrDifference"`
	AvgBlueComfort float64            `json:"avgBlueComfort"`
	AvgRedComfort  float64            `json:"avgRedComfort"`
	LaneDiffs      map[domain.Role]int `json:"laneDiffs"`
}

// GetTeamStats calculates current team balance stats
func (s *LobbyService) GetTeamStats(ctx context.Context, lobbyID uuid.UUID) (*TeamStats, error) {
	players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	// Get user IDs for profile lookup
	userIDs := make([]uuid.UUID, 0, len(players))
	for _, p := range players {
		userIDs = append(userIDs, p.UserID)
	}

	profiles, err := s.profileRepo.GetByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	stats := &TeamStats{
		LaneDiffs: make(map[domain.Role]int),
	}

	var blueTotalMMR, redTotalMMR int
	var blueTotalComfort, redTotalComfort float64
	blueCount, redCount := 0, 0
	blueRoleMMR := make(map[domain.Role]int)
	redRoleMMR := make(map[domain.Role]int)

	for _, p := range players {
		if p.Team == nil {
			continue
		}

		// Get player's best MMR and comfort for their assigned role (or average if no role)
		userProfiles := profiles[p.UserID]
		var mmr int
		var comfort int

		if p.AssignedRole != nil {
			// Find profile for assigned role
			for _, prof := range userProfiles {
				if prof.Role == *p.AssignedRole {
					mmr = prof.MMR
					comfort = prof.ComfortRating
					break
				}
			}
		}

		// If no assigned role or no profile found, use average
		if mmr == 0 && len(userProfiles) > 0 {
			totalMMR := 0
			totalComfort := 0
			for _, prof := range userProfiles {
				totalMMR += prof.MMR
				totalComfort += prof.ComfortRating
			}
			mmr = totalMMR / len(userProfiles)
			comfort = totalComfort / len(userProfiles)
		}

		if *p.Team == domain.SideBlue {
			blueTotalMMR += mmr
			blueTotalComfort += float64(comfort)
			blueCount++
			if p.AssignedRole != nil {
				blueRoleMMR[*p.AssignedRole] = mmr
			}
		} else if *p.Team == domain.SideRed {
			redTotalMMR += mmr
			redTotalComfort += float64(comfort)
			redCount++
			if p.AssignedRole != nil {
				redRoleMMR[*p.AssignedRole] = mmr
			}
		}
	}

	if blueCount > 0 {
		stats.BlueTeamAvgMMR = blueTotalMMR / blueCount
		stats.AvgBlueComfort = blueTotalComfort / float64(blueCount)
	}
	if redCount > 0 {
		stats.RedTeamAvgMMR = redTotalMMR / redCount
		stats.AvgRedComfort = redTotalComfort / float64(redCount)
	}

	stats.MMRDifference = abs(stats.BlueTeamAvgMMR - stats.RedTeamAvgMMR)

	// Calculate lane diffs
	for _, role := range domain.AllRoles {
		blueMMR := blueRoleMMR[role]
		redMMR := redRoleMMR[role]
		stats.LaneDiffs[role] = blueMMR - redMMR
	}

	return stats, nil
}

// ==================== Voting ====================

// CastVote allows a player to vote for a match option
func (s *LobbyService) CastVote(ctx context.Context, lobbyID, userID uuid.UUID, optionNumber int) (*domain.VotingStatus, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	if !lobby.VotingEnabled {
		return nil, ErrVotingNotEnabled
	}

	if lobby.Status != domain.LobbyStatusMatchmaking {
		return nil, ErrInvalidLobbyState
	}

	// Verify user is in lobby
	_, err = s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotInLobby
		}
		return nil, err
	}

	// Verify the option exists
	_, err = s.matchOptionRepo.GetByLobbyIDAndNumber(ctx, lobbyID, optionNumber)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidMatchOption
		}
		return nil, err
	}

	// Check if user already voted
	existingVote, err := s.voteRepo.GetByLobbyAndUser(ctx, lobbyID, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if existingVote != nil {
		// Update existing vote
		existingVote.MatchOptionNum = optionNumber
		existingVote.UpdatedAt = time.Now()
		if err := s.voteRepo.Update(ctx, existingVote); err != nil {
			return nil, err
		}
	} else {
		// Create new vote
		vote := &domain.Vote{
			ID:             uuid.New(),
			LobbyID:        lobbyID,
			UserID:         userID,
			MatchOptionNum: optionNumber,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := s.voteRepo.Create(ctx, vote); err != nil {
			return nil, err
		}
	}

	return s.GetVotingStatus(ctx, lobbyID, &userID)
}

// GetVotingStatus returns the current voting state for a lobby
func (s *LobbyService) GetVotingStatus(ctx context.Context, lobbyID uuid.UUID, userID *uuid.UUID) (*domain.VotingStatus, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	// Get player count
	players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	// Build map of user IDs to display names
	userDisplayNames := make(map[uuid.UUID]string)
	for _, p := range players {
		if p.User != nil {
			userDisplayNames[p.UserID] = p.User.DisplayName
		}
	}

	// Get vote counts
	voteCounts, err := s.voteRepo.GetVoteCounts(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	// Get all votes to count total
	votes, err := s.voteRepo.GetVotesByLobby(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	// Build voters map (option number -> list of voters)
	voters := make(map[int][]domain.VoterInfo)
	for _, v := range votes {
		displayName := userDisplayNames[v.UserID]
		if displayName == "" {
			displayName = "Unknown"
		}
		voters[v.MatchOptionNum] = append(voters[v.MatchOptionNum], domain.VoterInfo{
			UserID:      v.UserID,
			DisplayName: displayName,
		})
	}

	status := &domain.VotingStatus{
		VotingEnabled: lobby.VotingEnabled,
		VotingMode:    lobby.VotingMode,
		Deadline:      lobby.VotingDeadline,
		TotalPlayers:  len(players),
		VotesCast:     len(votes),
		VoteCounts:    voteCounts,
		Voters:        voters,
	}

	// Get user's vote if userID provided
	if userID != nil {
		for _, v := range votes {
			if v.UserID == *userID {
				status.UserVote = &v.MatchOptionNum
				break
			}
		}
	}

	// Calculate winning option based on voting mode
	status.WinningOption, status.CanFinalize = s.calculateVotingResult(lobby, len(players), voteCounts, len(votes))

	return status, nil
}

// calculateVotingResult determines the winning option based on voting mode
func (s *LobbyService) calculateVotingResult(lobby *domain.Lobby, totalPlayers int, voteCounts map[int]int, totalVotes int) (*int, bool) {
	if totalVotes == 0 {
		return nil, false
	}

	// Find option with most votes
	var maxVotes int
	var winningOption *int
	var tiedOptions []int

	for optionNum, count := range voteCounts {
		if count > maxVotes {
			maxVotes = count
			opt := optionNum
			winningOption = &opt
			tiedOptions = []int{optionNum}
		} else if count == maxVotes {
			tiedOptions = append(tiedOptions, optionNum)
		}
	}

	// Handle ties - lowest option number wins
	if len(tiedOptions) > 1 {
		minOption := tiedOptions[0]
		for _, opt := range tiedOptions[1:] {
			if opt < minOption {
				minOption = opt
			}
		}
		winningOption = &minOption
	}

	// Determine if can finalize based on voting mode
	var canFinalize bool
	switch lobby.VotingMode {
	case domain.VotingModeMajority:
		// Need more than 50% of total players
		canFinalize = maxVotes > totalPlayers/2
	case domain.VotingModeUnanimous:
		// All votes must be for the same option
		canFinalize = maxVotes == totalVotes && totalVotes == totalPlayers
	case domain.VotingModeCaptainOverride:
		// Captain can force, but show if majority is reached
		canFinalize = maxVotes > totalPlayers/2
	default:
		canFinalize = maxVotes > totalPlayers/2
	}

	return winningOption, canFinalize
}

// StartVoting enables voting on a lobby (captain only)
func (s *LobbyService) StartVoting(ctx context.Context, lobbyID, userID uuid.UUID, durationSeconds int) error {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLobbyNotFound
		}
		return err
	}

	// Verify user is a captain
	player, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotInLobby
		}
		return err
	}
	if !player.IsCaptain {
		return ErrNotCaptain
	}

	if lobby.Status != domain.LobbyStatusMatchmaking {
		return ErrInvalidLobbyState
	}

	// Clear any existing votes
	if err := s.voteRepo.DeleteByLobby(ctx, lobbyID); err != nil {
		return err
	}

	// Set voting deadline if duration provided
	lobby.VotingEnabled = true
	if durationSeconds > 0 {
		deadline := time.Now().Add(time.Duration(durationSeconds) * time.Second)
		lobby.VotingDeadline = &deadline
	}

	return s.lobbyRepo.Update(ctx, lobby)
}

// EndVoting ends voting and applies the winning option (captain only)
func (s *LobbyService) EndVoting(ctx context.Context, lobbyID, userID uuid.UUID, forceOption *int) (*domain.Lobby, error) {
	lobby, err := s.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLobbyNotFound
		}
		return nil, err
	}

	// Verify user is a captain
	player, err := s.lobbyPlayerRepo.GetByLobbyIDAndUserID(ctx, lobbyID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotInLobby
		}
		return nil, err
	}
	if !player.IsCaptain {
		return nil, ErrNotCaptain
	}

	if lobby.Status != domain.LobbyStatusMatchmaking {
		return nil, ErrInvalidLobbyState
	}

	if !lobby.VotingEnabled {
		return nil, ErrVotingNotEnabled
	}

	// Get voting status to determine winner
	voteCounts, err := s.voteRepo.GetVoteCounts(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	votes, err := s.voteRepo.GetVotesByLobby(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	players, err := s.lobbyPlayerRepo.GetByLobbyID(ctx, lobbyID)
	if err != nil {
		return nil, err
	}

	var optionToApply int

	if forceOption != nil && lobby.VotingMode == domain.VotingModeCaptainOverride {
		// Captain can force any option in captain_override mode
		optionToApply = *forceOption
	} else {
		// Use the winning option
		winningOption, canFinalize := s.calculateVotingResult(lobby, len(players), voteCounts, len(votes))

		if !canFinalize && lobby.VotingMode != domain.VotingModeCaptainOverride {
			return nil, errors.New("voting criteria not met")
		}

		if winningOption == nil {
			return nil, errors.New("no winning option determined")
		}

		optionToApply = *winningOption
	}

	// Verify the option exists
	_, err = s.matchOptionRepo.GetByLobbyIDAndNumber(ctx, lobbyID, optionToApply)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidMatchOption
		}
		return nil, err
	}

	// Apply the match option
	if err := s.applyMatchOption(ctx, lobbyID, optionToApply); err != nil {
		return nil, err
	}

	// Clear voting state
	lobby.VotingEnabled = false
	lobby.VotingDeadline = nil

	// Clean up votes
	if err := s.voteRepo.DeleteByLobby(ctx, lobbyID); err != nil {
		return nil, err
	}

	return s.lobbyRepo.GetByID(ctx, lobbyID)
}

// RemovePlayerVote removes a player's vote when they leave the lobby
func (s *LobbyService) RemovePlayerVote(ctx context.Context, lobbyID, userID uuid.UUID) error {
	return s.voteRepo.DeleteByLobbyAndUser(ctx, lobbyID, userID)
}
