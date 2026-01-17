package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

// JoinLobbyRequest represents a request to join a lobby
type JoinLobbyRequest struct {
	Client  *LobbyClient
	LobbyID uuid.UUID
}

// LobbyHub manages WebSocket connections for all lobbies
type LobbyHub struct {
	lobbies    map[uuid.UUID]*LobbyState
	clients    map[*LobbyClient]bool
	register   chan *LobbyClient
	unregister chan *LobbyClient
	joinLobby  chan *JoinLobbyRequest
	stop       chan struct{}
	done       chan struct{}
	stopped    bool

	lobbyRepo       repository.LobbyRepository
	lobbyPlayerRepo repository.LobbyPlayerRepository
	matchOptionRepo repository.MatchOptionRepository
	userRepo        repository.UserRepository

	mu sync.RWMutex
}

// NewLobbyHub creates a new lobby hub
func NewLobbyHub(
	lobbyRepo repository.LobbyRepository,
	lobbyPlayerRepo repository.LobbyPlayerRepository,
	matchOptionRepo repository.MatchOptionRepository,
	userRepo repository.UserRepository,
) *LobbyHub {
	return &LobbyHub{
		lobbies:         make(map[uuid.UUID]*LobbyState),
		clients:         make(map[*LobbyClient]bool),
		register:        make(chan *LobbyClient),
		unregister:      make(chan *LobbyClient),
		joinLobby:       make(chan *JoinLobbyRequest),
		stop:            make(chan struct{}),
		done:            make(chan struct{}),
		lobbyRepo:       lobbyRepo,
		lobbyPlayerRepo: lobbyPlayerRepo,
		matchOptionRepo: matchOptionRepo,
		userRepo:        userRepo,
	}
}

// Run starts the lobby hub event loop
func (h *LobbyHub) Run() {
	defer close(h.done)

	// Start cleanup goroutine for expired actions
	go h.cleanupExpiredActions()

	for {
		select {
		case <-h.stop:
			h.mu.Lock()
			h.stopped = true

			// Close all clients
			for client := range h.clients {
				client.Close()
			}
			h.clients = make(map[*LobbyClient]bool)
			h.lobbies = make(map[uuid.UUID]*LobbyState)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			if !h.stopped {
				h.clients[client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if !h.stopped {
				if _, ok := h.clients[client]; ok {
					delete(h.clients, client)
					client.Close()

					// Remove from lobby state
					lobbyID := client.LobbyID()
					if lobbyID != uuid.Nil {
						if state, exists := h.lobbies[lobbyID]; exists {
							state.RemoveClient(client)
							// Clean up empty lobbies
							if state.ClientCount() == 0 {
								delete(h.lobbies, lobbyID)
							}
						}
					}
				}
			}
			h.mu.Unlock()

		case req := <-h.joinLobby:
			h.mu.Lock()
			stopped := h.stopped
			h.mu.Unlock()
			if !stopped {
				h.handleJoinLobby(req)
			}
		}
	}
}

// Stop gracefully shuts down the hub
func (h *LobbyHub) Stop() {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return
	}
	h.mu.Unlock()

	close(h.stop)
	<-h.done
}

// Register adds a client to the hub
func (h *LobbyHub) Register(client *LobbyClient) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *LobbyHub) Unregister(client *LobbyClient) {
	h.mu.RLock()
	stopped := h.stopped
	h.mu.RUnlock()

	if stopped {
		return
	}

	select {
	case h.unregister <- client:
	default:
	}
}

// handleJoinLobby processes a join lobby request
func (h *LobbyHub) handleJoinLobby(req *JoinLobbyRequest) {
	ctx := context.Background()

	// Verify lobby exists
	lobby, err := h.lobbyRepo.GetByID(ctx, req.LobbyID)
	if err != nil {
		req.Client.sendError("LOBBY_NOT_FOUND", "Lobby not found")
		return
	}

	// Leave current lobby if in one
	oldLobbyID := req.Client.LobbyID()
	if oldLobbyID != uuid.Nil && oldLobbyID != req.LobbyID {
		h.mu.Lock()
		if state, exists := h.lobbies[oldLobbyID]; exists {
			state.RemoveClient(req.Client)
			if state.ClientCount() == 0 {
				delete(h.lobbies, oldLobbyID)
			}
		}
		h.mu.Unlock()
	}

	// Get or create lobby state
	h.mu.Lock()
	state, exists := h.lobbies[req.LobbyID]
	if !exists {
		state = NewLobbyState(req.LobbyID)
		h.lobbies[req.LobbyID] = state
	}
	h.mu.Unlock()

	// Add client to lobby
	state.AddClient(req.Client)
	req.Client.SetLobbyID(req.LobbyID)

	// Build and send state sync
	syncPayload := h.buildStateSyncPayload(ctx, lobby, state)
	req.Client.Send(NewLobbyMessage(LobbyMsgStateSync, syncPayload))

	log.Printf("LobbyHub: Client %s joined lobby %s (now %d clients)", req.Client.userID, req.LobbyID, state.ClientCount())
}

// buildStateSyncPayload builds the full state sync payload
func (h *LobbyHub) buildStateSyncPayload(ctx context.Context, lobby *domain.Lobby, state *LobbyState) *LobbyStateSyncPayload {
	// Get players with user info
	players, _ := h.lobbyPlayerRepo.GetByLobbyID(ctx, lobby.ID)

	// Pre-build userID -> displayName map for O(1) lookups
	userDisplayNames := make(map[uuid.UUID]string, len(players))
	for _, p := range players {
		if p.User != nil {
			userDisplayNames[p.UserID] = p.User.DisplayName
		}
	}

	// Build player info
	playerInfos := make([]LobbyPlayerInfo, len(players))
	for i, p := range players {
		displayName := userDisplayNames[p.UserID]
		var team, role *string
		if p.Team != nil {
			s := string(*p.Team)
			team = &s
		}
		if p.AssignedRole != nil {
			s := string(*p.AssignedRole)
			role = &s
		}
		playerInfos[i] = LobbyPlayerInfo{
			ID:           p.ID.String(),
			UserID:       p.UserID.String(),
			DisplayName:  displayName,
			Team:         team,
			AssignedRole: role,
			IsReady:      p.IsReady,
			IsCaptain:    p.IsCaptain,
			JoinOrder:    p.JoinOrder,
		}
	}

	// Build lobby info
	var roomID *string
	if lobby.RoomID != nil {
		s := lobby.RoomID.String()
		roomID = &s
	}
	var votingDeadline *string
	if lobby.VotingDeadline != nil {
		s := lobby.VotingDeadline.Format(time.RFC3339)
		votingDeadline = &s
	}

	lobbyInfo := &LobbyInfo{
		ID:                   lobby.ID.String(),
		ShortCode:            lobby.ShortCode,
		CreatedBy:            lobby.CreatedBy.String(),
		Status:               string(lobby.Status),
		SelectedMatchOption:  lobby.SelectedMatchOption,
		DraftMode:            string(lobby.DraftMode),
		TimerDurationSeconds: lobby.TimerDurationSeconds,
		RoomID:               roomID,
		VotingEnabled:        lobby.VotingEnabled,
		VotingMode:           string(lobby.VotingMode),
		VotingDeadline:       votingDeadline,
	}

	// Get match options if available
	var matchOptionInfos []MatchOptionInfo
	if lobby.Status == domain.LobbyStatusMatchmaking || lobby.Status == domain.LobbyStatusTeamSelected {
		options, _ := h.matchOptionRepo.GetByLobbyID(ctx, lobby.ID)
		matchOptionInfos = make([]MatchOptionInfo, len(options))
		for i, opt := range options {
			assignments := make([]AssignmentInfo, len(opt.Assignments))
			for j, a := range opt.Assignments {
				displayName := ""
				if a.User != nil {
					displayName = a.User.DisplayName
				}
				assignments[j] = AssignmentInfo{
					UserID:        a.UserID.String(),
					DisplayName:   displayName,
					Team:          string(a.Team),
					AssignedRole:  string(a.AssignedRole),
					RoleMMR:       a.RoleMMR,
					ComfortRating: a.ComfortRating,
				}
			}
			matchOptionInfos[i] = MatchOptionInfo{
				OptionNumber:   opt.OptionNumber,
				AlgorithmType:  string(opt.AlgorithmType),
				BlueTeamAvgMMR: opt.BlueTeamAvgMMR,
				RedTeamAvgMMR:  opt.RedTeamAvgMMR,
				MMRDifference:  opt.MMRDifference,
				BalanceScore:   opt.BalanceScore,
				AvgBlueComfort: opt.AvgBlueComfort,
				AvgRedComfort:  opt.AvgRedComfort,
				MaxLaneDiff:    opt.MaxLaneDiff,
				Assignments:    assignments,
			}
		}
	}

	// Build voting status if applicable
	var votingStatus *VotingStatusInfo
	if lobby.VotingEnabled && lobby.Status == domain.LobbyStatusMatchmaking {
		voteCounts := state.GetVoteCounts()
		votersByOption := state.GetVotersByOption()

		// Build voters map with display names using pre-built map (O(1) lookup)
		voters := make(map[int][]VoterInfoPayload)
		for optNum, userIDs := range votersByOption {
			voterList := make([]VoterInfoPayload, 0, len(userIDs))
			for _, uid := range userIDs {
				voterList = append(voterList, VoterInfoPayload{
					UserID:      uid.String(),
					DisplayName: userDisplayNames[uid],
				})
			}
			voters[optNum] = voterList
		}

		var deadline *string
		if lobby.VotingDeadline != nil {
			s := lobby.VotingDeadline.Format(time.RFC3339)
			deadline = &s
		}

		votingStatus = &VotingStatusInfo{
			VotingEnabled: true,
			VotingMode:    string(lobby.VotingMode),
			Deadline:      deadline,
			TotalPlayers:  len(players),
			VotesCast:     state.GetTotalVotes(),
			VoteCounts:    voteCounts,
			Voters:        voters,
			CanFinalize:   state.GetTotalVotes() == len(players),
		}
	}

	return &LobbyStateSyncPayload{
		Lobby:         lobbyInfo,
		Players:       playerInfos,
		MatchOptions:  matchOptionInfos,
		VotingStatus:  votingStatus,
		Votes:         state.GetVotes(),
		PendingAction: state.GetPendingAction().ToInfo(),
	}
}

// cleanupExpiredActions periodically cleans up expired pending actions
func (h *LobbyHub) cleanupExpiredActions() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stop:
			return
		case <-ticker.C:
			h.mu.RLock()
			for _, state := range h.lobbies {
				action := state.GetPendingAction()
				if action != nil && action.IsExpired() {
					state.ClearPendingAction()
					state.Broadcast(NewLobbyMessage(LobbyMsgActionCancelled, ActionCancelledPayload{
						ActionID:    action.ID.String(),
						CancelledBy: "system",
					}))
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ============== Broadcast Methods (called from handlers) ==============

// GetLobbyState returns the state for a lobby (creates if not exists)
func (h *LobbyHub) GetLobbyState(lobbyID uuid.UUID) *LobbyState {
	h.mu.Lock()
	defer h.mu.Unlock()

	state, exists := h.lobbies[lobbyID]
	if !exists {
		state = NewLobbyState(lobbyID)
		h.lobbies[lobbyID] = state
	}
	return state
}

// GetLobbyStateIfExists returns the state for a lobby if it exists
func (h *LobbyHub) GetLobbyStateIfExists(lobbyID uuid.UUID) *LobbyState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lobbies[lobbyID]
}

// BroadcastPlayerJoined broadcasts a player joined event
func (h *LobbyHub) BroadcastPlayerJoined(lobbyID uuid.UUID, player *domain.LobbyPlayer) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	var team, role *string
	if player.Team != nil {
		s := string(*player.Team)
		team = &s
	}
	if player.AssignedRole != nil {
		s := string(*player.AssignedRole)
		role = &s
	}

	displayName := ""
	if player.User != nil {
		displayName = player.User.DisplayName
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgPlayerJoined, PlayerJoinedPayload{
		Player: LobbyPlayerInfo{
			ID:           player.ID.String(),
			UserID:       player.UserID.String(),
			DisplayName:  displayName,
			Team:         team,
			AssignedRole: role,
			IsReady:      player.IsReady,
			IsCaptain:    player.IsCaptain,
			JoinOrder:    player.JoinOrder,
		},
	}))
}

// BroadcastPlayerLeft broadcasts a player left event
func (h *LobbyHub) BroadcastPlayerLeft(lobbyID uuid.UUID, userID uuid.UUID, displayName string) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgPlayerLeft, PlayerLeftPayload{
		UserID:      userID.String(),
		DisplayName: displayName,
	}))
}

// BroadcastPlayerReadyChanged broadcasts a ready status change
func (h *LobbyHub) BroadcastPlayerReadyChanged(lobbyID uuid.UUID, userID uuid.UUID, isReady bool) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgPlayerReadyChanged, PlayerReadyChangedPayload{
		UserID:  userID.String(),
		IsReady: isReady,
	}))
}

// BroadcastStatusChanged broadcasts a status change
func (h *LobbyHub) BroadcastStatusChanged(lobbyID uuid.UUID, oldStatus, newStatus string) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgStatusChanged, StatusChangedPayload{
		OldStatus: oldStatus,
		NewStatus: newStatus,
	}))
}

// BroadcastMatchOptionsGenerated broadcasts match options
func (h *LobbyHub) BroadcastMatchOptionsGenerated(lobbyID uuid.UUID, options []*domain.MatchOption) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		log.Printf("LobbyHub.BroadcastMatchOptionsGenerated: no state for lobby %s", lobbyID)
		return
	}
	log.Printf("LobbyHub.BroadcastMatchOptionsGenerated: lobby %s has %d connected clients", lobbyID, state.ClientCount())

	optionInfos := make([]MatchOptionInfo, len(options))
	for i, opt := range options {
		assignments := make([]AssignmentInfo, len(opt.Assignments))
		for j, a := range opt.Assignments {
			displayName := ""
			if a.User != nil {
				displayName = a.User.DisplayName
			}
			assignments[j] = AssignmentInfo{
				UserID:        a.UserID.String(),
				DisplayName:   displayName,
				Team:          string(a.Team),
				AssignedRole:  string(a.AssignedRole),
				RoleMMR:       a.RoleMMR,
				ComfortRating: a.ComfortRating,
			}
		}
		optionInfos[i] = MatchOptionInfo{
			OptionNumber:   opt.OptionNumber,
			AlgorithmType:  string(opt.AlgorithmType),
			BlueTeamAvgMMR: opt.BlueTeamAvgMMR,
			RedTeamAvgMMR:  opt.RedTeamAvgMMR,
			MMRDifference:  opt.MMRDifference,
			BalanceScore:   opt.BalanceScore,
			AvgBlueComfort: opt.AvgBlueComfort,
			AvgRedComfort:  opt.AvgRedComfort,
			MaxLaneDiff:    opt.MaxLaneDiff,
			Assignments:    assignments,
		}
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgMatchOptionsGenerated, MatchOptionsGeneratedPayload{
		Options: optionInfos,
	}))
}

// BroadcastTeamSelected broadcasts team selection
func (h *LobbyHub) BroadcastTeamSelected(lobbyID uuid.UUID, optionNumber int, players []LobbyPlayerInfo, stats *TeamStatsInfo) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgTeamSelected, TeamSelectedPayload{
		OptionNumber: optionNumber,
		Assignments:  players,
		TeamStats:    stats,
	}))

	// Clear votes when team is selected
	state.ClearVotes()
}

// BroadcastVoteCast broadcasts a vote
func (h *LobbyHub) BroadcastVoteCast(lobbyID uuid.UUID, userID uuid.UUID, displayName string, optionNumber int, userDisplayNames map[uuid.UUID]string) {
	state := h.GetLobbyState(lobbyID)

	// Record the vote in memory
	state.CastVote(userID, optionNumber)

	// Build voters map with display names
	votersByOption := state.GetVotersByOption()
	voters := make(map[int][]VoterInfoPayload)
	for optNum, userIDs := range votersByOption {
		voterList := make([]VoterInfoPayload, 0, len(userIDs))
		for _, uid := range userIDs {
			voterList = append(voterList, VoterInfoPayload{
				UserID:      uid.String(),
				DisplayName: userDisplayNames[uid],
			})
		}
		voters[optNum] = voterList
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgVoteCast, VoteCastPayload{
		UserID:       userID.String(),
		DisplayName:  displayName,
		OptionNumber: optionNumber,
		VoteCounts:   state.GetVoteCounts(),
		VotesCast:    state.GetTotalVotes(),
		Voters:       voters,
	}))
}

// BroadcastActionProposed broadcasts a proposed action
func (h *LobbyHub) BroadcastActionProposed(lobbyID uuid.UUID, action *InMemoryPendingAction) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		log.Printf("LobbyHub.BroadcastActionProposed: no state for lobby %s", lobbyID)
		return
	}

	log.Printf("LobbyHub.BroadcastActionProposed: broadcasting to %d clients for lobby %s, action type: %s", state.ClientCount(), lobbyID, action.ActionType)
	state.Broadcast(NewLobbyMessage(LobbyMsgActionProposed, ActionProposedPayload{
		Action: *action.ToInfo(),
	}))
}

// BroadcastActionApproved broadcasts an action approval
func (h *LobbyHub) BroadcastActionApproved(lobbyID uuid.UUID, actionID uuid.UUID, side string, approvedByBlue, approvedByRed bool) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgActionApproved, ActionApprovedPayload{
		ActionID:       actionID.String(),
		ApprovedBySide: side,
		ApprovedByBlue: approvedByBlue,
		ApprovedByRed:  approvedByRed,
	}))
}

// BroadcastActionExecuted broadcasts action execution
func (h *LobbyHub) BroadcastActionExecuted(lobbyID uuid.UUID, actionType string, result interface{}) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.ClearPendingAction()
	state.Broadcast(NewLobbyMessage(LobbyMsgActionExecuted, ActionExecutedPayload{
		ActionType: actionType,
		Result:     result,
	}))
}

// BroadcastActionCancelled broadcasts action cancellation
func (h *LobbyHub) BroadcastActionCancelled(lobbyID uuid.UUID, actionID uuid.UUID, cancelledBy string) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.ClearPendingAction()
	state.Broadcast(NewLobbyMessage(LobbyMsgActionCancelled, ActionCancelledPayload{
		ActionID:    actionID.String(),
		CancelledBy: cancelledBy,
	}))
}

// BroadcastDraftStarting broadcasts draft starting
func (h *LobbyHub) BroadcastDraftStarting(lobbyID uuid.UUID, roomID uuid.UUID, shortCode string) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgDraftStarting, DraftStartingPayload{
		RoomID:    roomID.String(),
		ShortCode: shortCode,
	}))
}

// BroadcastCaptainChanged broadcasts captain change
func (h *LobbyHub) BroadcastCaptainChanged(lobbyID uuid.UUID, team string, newCaptainID uuid.UUID, newCaptainName string, oldCaptainID *uuid.UUID) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	payload := CaptainChangedPayload{
		Team:           team,
		NewCaptainID:   newCaptainID.String(),
		NewCaptainName: newCaptainName,
	}
	if oldCaptainID != nil {
		payload.OldCaptainID = oldCaptainID.String()
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgCaptainChanged, payload))
}

// BroadcastPlayerKicked broadcasts player kick
func (h *LobbyHub) BroadcastPlayerKicked(lobbyID uuid.UUID, userID uuid.UUID, displayName string, kickedBy string) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgPlayerKicked, PlayerKickedPayload{
		UserID:      userID.String(),
		DisplayName: displayName,
		KickedBy:    kickedBy,
	}))
}

// BroadcastTeamStatsUpdated broadcasts team stats update
func (h *LobbyHub) BroadcastTeamStatsUpdated(lobbyID uuid.UUID, stats *TeamStatsInfo) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgTeamStatsUpdated, TeamStatsUpdatedPayload{
		Stats: *stats,
	}))
}

// BroadcastVotingStatusUpdated broadcasts voting status update
func (h *LobbyHub) BroadcastVotingStatusUpdated(lobbyID uuid.UUID, status *VotingStatusInfo) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		return
	}

	state.Broadcast(NewLobbyMessage(LobbyMsgVotingStatusUpdated, VotingStatusUpdatedPayload{
		Status: *status,
	}))
}

// BroadcastLobbyUpdate broadcasts a full lobby update (convenience method)
func (h *LobbyHub) BroadcastLobbyUpdate(ctx context.Context, lobbyID uuid.UUID) {
	state := h.GetLobbyStateIfExists(lobbyID)
	if state == nil {
		log.Printf("LobbyHub.BroadcastLobbyUpdate: no state for lobby %s", lobbyID)
		return
	}

	clientCount := state.ClientCount()
	log.Printf("LobbyHub.BroadcastLobbyUpdate: lobby %s has %d connected clients", lobbyID, clientCount)

	lobby, err := h.lobbyRepo.GetByID(ctx, lobbyID)
	if err != nil {
		log.Printf("LobbyHub.BroadcastLobbyUpdate: failed to get lobby %s: %v", lobbyID, err)
		return
	}

	// Build state sync payload ONCE and reuse for all clients
	syncPayload := h.buildStateSyncPayload(ctx, lobby, state)
	msg := NewLobbyMessage(LobbyMsgStateSync, syncPayload)

	// Broadcast to all clients
	state.Broadcast(msg)
	log.Printf("LobbyHub.BroadcastLobbyUpdate: broadcasted state sync to %d clients for lobby %s", clientCount, lobbyID)
}
