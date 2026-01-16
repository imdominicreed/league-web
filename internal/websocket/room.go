package websocket

import (
	"context"
	"log"
	"sync"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

type Room struct {
	id              uuid.UUID
	shortCode       string
	clients         map[*Client]bool
	blueClient      *Client
	redClient       *Client
	spectators      map[*Client]bool
	timerDurationMs int
	userRepo        repository.UserRepository
	championRepo    repository.ChampionRepository
	roomRepo        repository.RoomRepository
	draftActionRepo repository.DraftActionRepository

	// Team draft mode (5v5)
	isTeamDraft      bool
	roomPlayers      map[uuid.UUID]*domain.RoomPlayer // userId -> RoomPlayer
	blueCaptainID    *uuid.UUID
	redCaptainID     *uuid.UUID
	blueTeamClients  map[*Client]bool // Non-captain blue team members
	redTeamClients   map[*Client]bool // Non-captain red team members

	// Managers
	emitter  *EventEmitter
	timerMgr *TimerManager
	pauseMgr *PauseManager
	editMgr  *EditManager
	draftMgr *DraftStateManager

	// Channels
	join           chan *Client
	leave          chan *Client
	broadcast      chan *Message
	selectChampion chan *SelectChampionRequest
	lockIn         chan *Client
	hoverChampion  chan *HoverChampionRequest
	ready          chan *ReadyRequest
	startDraft     chan *Client
	syncState      chan *Client
	pauseDraft     chan *Client
	resumeDraft    chan *Client
	proposeEdit    chan *ProposeEditRequest
	confirmEdit    chan *Client
	rejectEdit     chan *Client
	readyToResume  chan *ReadyToResumeRequest

	mu sync.RWMutex
}

// ProposeEditRequest contains the client and payload for an edit proposal
type ProposeEditRequest struct {
	Client  *Client
	Payload ProposeEditPayload
}

// Note: PendingEdit is defined in edit_manager.go
// Note: DraftState is defined in draft_state.go

type SelectChampionRequest struct {
	Client     *Client
	ChampionID string
}

type HoverChampionRequest struct {
	Client     *Client
	ChampionID *string
}

type ReadyRequest struct {
	Client *Client
	Ready  bool
}

type ReadyToResumeRequest struct {
	Client *Client
	Ready  bool
}

func NewRoom(id uuid.UUID, shortCode string, timerDurationMs int, userRepo repository.UserRepository, championRepo repository.ChampionRepository, roomRepo repository.RoomRepository, draftActionRepo repository.DraftActionRepository) *Room {
	r := &Room{
		id:               id,
		shortCode:        shortCode,
		clients:          make(map[*Client]bool),
		spectators:       make(map[*Client]bool),
		blueTeamClients:  make(map[*Client]bool),
		redTeamClients:   make(map[*Client]bool),
		timerDurationMs:  timerDurationMs,
		userRepo:         userRepo,
		championRepo:     championRepo,
		roomRepo:         roomRepo,
		draftActionRepo:  draftActionRepo,
		join:             make(chan *Client),
		leave:              make(chan *Client),
		broadcast:          make(chan *Message),
		selectChampion:     make(chan *SelectChampionRequest),
		lockIn:             make(chan *Client),
		hoverChampion:      make(chan *HoverChampionRequest),
		ready:              make(chan *ReadyRequest),
		startDraft:         make(chan *Client),
		syncState:          make(chan *Client),
		pauseDraft:         make(chan *Client),
		resumeDraft:        make(chan *Client),
		proposeEdit:        make(chan *ProposeEditRequest),
		confirmEdit:        make(chan *Client),
		rejectEdit:         make(chan *Client),
		readyToResume:      make(chan *ReadyToResumeRequest),
	}

	// Initialize managers
	r.emitter = NewEventEmitter(r)
	r.timerMgr = NewTimerManager(timerDurationMs, r.emitter, r.handleTimerExpired)
	r.pauseMgr = NewPauseManager(r)
	r.editMgr = NewEditManager(r)

	// DraftStateManager
	r.draftMgr = NewDraftStateManager(r, championRepo, roomRepo, draftActionRepo, timerDurationMs)

	return r
}

// InitializeTeamDraft sets up the room for 5v5 team draft mode
func (r *Room) InitializeTeamDraft(players []*domain.RoomPlayer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.isTeamDraft = true
	r.roomPlayers = make(map[uuid.UUID]*domain.RoomPlayer)
	for _, p := range players {
		r.roomPlayers[p.UserID] = p
		if p.IsCaptain {
			if p.Team == domain.SideBlue {
				r.blueCaptainID = &p.UserID
			} else if p.Team == domain.SideRed {
				r.redCaptainID = &p.UserID
			}
		}
	}
}

// IsTeamDraft returns whether the room is in team draft mode
func (r *Room) IsTeamDraft() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isTeamDraft
}

// canAct checks if a user can perform pick/ban actions for the given side
func (r *Room) canAct(userID uuid.UUID, side string) bool {
	if r.isTeamDraft {
		// In team draft, only captains can act
		if side == "blue" && r.blueCaptainID != nil {
			return userID == *r.blueCaptainID
		}
		if side == "red" && r.redCaptainID != nil {
			return userID == *r.redCaptainID
		}
		return false
	}
	// Original 1v1 logic: check if client is the side's assigned client
	return true
}

// GetPlayerTeam returns the team for a player in team draft mode
func (r *Room) GetPlayerTeam(userID uuid.UUID) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.isTeamDraft {
		return ""
	}
	if player, ok := r.roomPlayers[userID]; ok {
		return string(player.Team)
	}
	return ""
}

func (r *Room) getUserDisplayName(userID uuid.UUID) string {
	user, err := r.userRepo.GetByID(context.Background(), userID)
	if err != nil {
		log.Printf("Failed to get user %s: %v", userID, err)
		return "Unknown"
	}
	if user == nil {
		return "Unknown"
	}
	return user.DisplayName
}

// getDraftState returns the current draft state from the DraftStateManager.
func (r *Room) getDraftState() *DraftState {
	return r.draftMgr.GetState()
}

func (r *Room) Run() {
	for {
		select {
		case client := <-r.join:
			r.handleJoin(client)

		case client := <-r.leave:
			r.handleLeave(client)

		case msg := <-r.broadcast:
			r.mu.RLock()
			r.emitter.Broadcast(msg)
			r.mu.RUnlock()

		case req := <-r.selectChampion:
			r.handleSelectChampion(req)

		case client := <-r.lockIn:
			r.handleLockIn(client)

		case req := <-r.hoverChampion:
			r.handleHoverChampion(req)

		case req := <-r.ready:
			r.handleReady(req)

		case client := <-r.startDraft:
			r.handleStartDraft(client)

		case client := <-r.syncState:
			r.sendStateSync(client)

		case client := <-r.pauseDraft:
			r.handlePauseDraft(client)

		case client := <-r.resumeDraft:
			r.handleResumeDraft(client)

		case req := <-r.proposeEdit:
			r.handleProposeEdit(req)

		case client := <-r.confirmEdit:
			r.handleConfirmEdit(client)

		case client := <-r.rejectEdit:
			r.handleRejectEdit(client)

		case req := <-r.readyToResume:
			r.handleReadyToResume(req)
		}
	}
}

func (r *Room) handleJoin(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clients[client] = true

	// In team draft mode, only captains become blueClient/redClient
	// Other team members are tracked separately
	if r.isTeamDraft {
		isCaptain := false
		if r.blueCaptainID != nil && client.userID == *r.blueCaptainID {
			isCaptain = true
		}
		if r.redCaptainID != nil && client.userID == *r.redCaptainID {
			isCaptain = true
		}

		switch client.side {
		case "blue":
			if isCaptain {
				r.blueClient = client
				log.Printf("Blue captain %s connected", client.userID)
			} else {
				// Non-captain team member - track separately
				r.blueTeamClients[client] = true
				log.Printf("Blue team member %s connected (non-captain)", client.userID)
			}
		case "red":
			if isCaptain {
				r.redClient = client
				log.Printf("Red captain %s connected", client.userID)
			} else {
				// Non-captain team member - track separately
				r.redTeamClients[client] = true
				log.Printf("Red team member %s connected (non-captain)", client.userID)
			}
		default:
			r.spectators[client] = true
		}
	} else {
		// Original 1v1 behavior
		log.Printf("Room %s: Using 1v1 mode for user %s (side: %s)", r.id, client.userID, client.side)
		switch client.side {
		case "blue":
			if r.blueClient != nil && r.blueClient != client {
				client.sendError("SIDE_TAKEN", "Blue side is already taken")
				client.side = "spectator"
				r.spectators[client] = true
			} else {
				r.blueClient = client
			}
		case "red":
			if r.redClient != nil && r.redClient != client {
				client.sendError("SIDE_TAKEN", "Red side is already taken")
				client.side = "spectator"
				r.spectators[client] = true
			} else {
				r.redClient = client
			}
		default:
			r.spectators[client] = true
		}
	}

	// Send state sync to joining client
	r.sendStateSyncLocked(client)

	// Notify others
	r.emitter.PlayerUpdate(client.side, &PlayerInfo{
		UserID:      client.userID.String(),
		DisplayName: r.getUserDisplayName(client.userID),
		Ready:       client.ready,
	}, "joined")
}

func (r *Room) handleLeave(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.clients, client)
	delete(r.spectators, client)
	delete(r.blueTeamClients, client)
	delete(r.redTeamClients, client)

	if r.blueClient == client {
		r.blueClient = nil
		r.getDraftState().BlueReady = false
	}
	if r.redClient == client {
		r.redClient = nil
		r.getDraftState().RedReady = false
	}

	r.emitter.PlayerUpdate(client.side, nil, "left")
}

func (r *Room) handleSelectChampion(req *SelectChampionRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.getDraftState().Started || r.getDraftState().IsComplete {
		req.Client.sendError("INVALID_STATE", "Draft not in progress")
		return
	}

	phase := domain.GetPhase(r.getDraftState().CurrentPhase)
	if phase == nil {
		return
	}

	currentSide := string(phase.Team)

	// Check if it's this client's turn
	if r.isTeamDraft {
		// In team draft mode, use canAct to check if user is captain
		if !r.canAct(req.Client.userID, currentSide) {
			req.Client.sendError("NOT_YOUR_TURN", "Only the captain can pick/ban")
			return
		}
	} else {
		// Original 1v1 logic
		if (phase.Team == domain.SideBlue && req.Client != r.blueClient) ||
			(phase.Team == domain.SideRed && req.Client != r.redClient) {
			req.Client.sendError("NOT_YOUR_TURN", "It's not your turn")
			return
		}
	}

	// Check if champion is already picked/banned
	if r.draftMgr.IsChampionUsed(req.ChampionID) {
		req.Client.sendError("CHAMPION_UNAVAILABLE", "Champion is already picked or banned")
		return
	}

	// Store selection (will be confirmed on lock in)
	r.draftMgr.SetCurrentHover(currentSide, &req.ChampionID)

	// Broadcast hover
	msg, _ := NewMessage(MessageTypeChampionHovered, ChampionHoveredPayload{
		Side:       currentSide,
		ChampionID: &req.ChampionID,
	})
	r.emitter.Broadcast(msg)
}

func (r *Room) handleLockIn(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.getDraftState().Started || r.getDraftState().IsComplete {
		client.sendError("INVALID_STATE", "Draft not in progress")
		return
	}

	phase := domain.GetPhase(r.getDraftState().CurrentPhase)
	if phase == nil {
		return
	}

	currentSide := string(phase.Team)

	// Check if it's this client's turn
	if r.isTeamDraft {
		// In team draft mode, use canAct to check if user is captain
		if !r.canAct(client.userID, currentSide) {
			client.sendError("NOT_YOUR_TURN", "Only the captain can pick/ban")
			return
		}
	} else {
		// Original 1v1 logic
		if (phase.Team == domain.SideBlue && client != r.blueClient) ||
			(phase.Team == domain.SideRed && client != r.redClient) {
			client.sendError("NOT_YOUR_TURN", "It's not your turn")
			return
		}
	}

	championID := r.draftMgr.GetCurrentHover(currentSide)
	if championID == nil {
		none := "None"
		championID = &none
	}

	// Apply the selection
	r.draftMgr.applySelection(phase, *championID)

	// Stop current timer
	r.timerMgr.Stop()

	// Broadcast selection
	msg, _ := NewMessage(MessageTypeChampionSelected, ChampionSelectedPayload{
		Phase:      r.getDraftState().CurrentPhase,
		Team:       string(phase.Team),
		ActionType: string(phase.ActionType),
		ChampionID: *championID,
	})
	r.emitter.Broadcast(msg)

	// Move to next phase
	r.draftMgr.advancePhase()
}

func (r *Room) handleHoverChampion(req *HoverChampionRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.getDraftState().Started {
		return
	}

	msg, _ := NewMessage(MessageTypeChampionHovered, ChampionHoveredPayload{
		Side:       req.Client.side,
		ChampionID: req.ChampionID,
	})
	r.emitter.Broadcast(msg)
}

func (r *Room) handleReady(req *ReadyRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.getDraftState().Started {
		return
	}

	switch req.Client.side {
	case "blue":
		r.getDraftState().BlueReady = req.Ready
	case "red":
		r.getDraftState().RedReady = req.Ready
	}

	req.Client.ready = req.Ready

	r.emitter.PlayerUpdate(req.Client.side, &PlayerInfo{
		UserID:      req.Client.userID.String(),
		DisplayName: r.getUserDisplayName(req.Client.userID),
		Ready:       req.Ready,
	}, "ready_changed")
}

func (r *Room) handleStartDraft(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.getDraftState().Started {
		client.sendError("ALREADY_STARTED", "Draft already started")
		return
	}

	if r.blueClient == nil || r.redClient == nil {
		client.sendError("MISSING_PLAYERS", "Both sides must have a player")
		return
	}

	if !r.getDraftState().BlueReady || !r.getDraftState().RedReady {
		client.sendError("NOT_READY", "Both players must be ready")
		return
	}

	r.getDraftState().Started = true

	phase := domain.GetPhase(0)

	msg, _ := NewMessage(MessageTypeDraftStarted, DraftStartedPayload{
		CurrentPhase:     0,
		CurrentTeam:      string(phase.Team),
		ActionType:       string(phase.ActionType),
		TimerRemainingMs: r.timerDurationMs,
	})
	r.emitter.Broadcast(msg)

	// Send STATE_SYNC to ensure room.status updates to 'in_progress'
	for client := range r.clients {
		r.sendStateSyncLocked(client)
	}

	r.timerMgr.Start()
}

func (r *Room) handleTimerExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pauseMgr.IsPaused() {
		return
	}

	r.draftMgr.HandleTimerExpired()
}

func (r *Room) sendStateSync(client *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.sendStateSyncLocked(client)
}

// syncAllClients sends state sync to all connected clients.
// Called by DraftStateManager after state changes.
func (r *Room) syncAllClients() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for client := range r.clients {
		r.sendStateSyncLocked(client)
	}
}

func (r *Room) sendStateSyncLocked(client *Client) {
	var currentTeam, actionType string
	timerRemaining := r.timerDurationMs

	if phase := domain.GetPhase(r.getDraftState().CurrentPhase); phase != nil {
		currentTeam = string(phase.Team)
		actionType = string(phase.ActionType)
	}

	if r.getDraftState().Started && !r.getDraftState().IsComplete {
		if r.pauseMgr.IsPaused() {
			// When paused, use the frozen timer value
			timerRemaining = r.pauseMgr.GetFrozenTimerMs()
		} else {
			timerRemaining = r.timerMgr.GetRemaining()
		}
	}

	var bluePlayer, redPlayer *PlayerInfo
	if r.blueClient != nil {
		bluePlayer = &PlayerInfo{
			UserID:      r.blueClient.userID.String(),
			DisplayName: r.getUserDisplayName(r.blueClient.userID),
			Ready:       r.getDraftState().BlueReady,
		}
	}
	if r.redClient != nil {
		redPlayer = &PlayerInfo{
			UserID:      r.redClient.userID.String(),
			DisplayName: r.getUserDisplayName(r.redClient.userID),
			Ready:       r.getDraftState().RedReady,
		}
	}

	status := "waiting"
	if r.getDraftState().Started {
		if r.getDraftState().IsComplete {
			status = "completed"
		} else {
			status = "in_progress"
		}
	}

	// Determine if this client is a captain
	isCaptain := false
	if r.isTeamDraft {
		if r.blueCaptainID != nil && client.userID == *r.blueCaptainID {
			isCaptain = true
		}
		if r.redCaptainID != nil && client.userID == *r.redCaptainID {
			isCaptain = true
		}
		log.Printf("STATE_SYNC: isTeamDraft=true, client=%s, side=%s, blueCaptainID=%v, redCaptainID=%v, isCaptain=%v",
			client.userID, client.side, r.blueCaptainID, r.redCaptainID, isCaptain)
	} else {
		// In 1v1 mode, both players are effectively "captains"
		isCaptain = client.side == "blue" || client.side == "red"
		log.Printf("STATE_SYNC: isTeamDraft=false (1v1 mode), client=%s, side=%s, isCaptain=%v",
			client.userID, client.side, isCaptain)
	}

	// Build team players list for team draft mode
	var teamPlayers []TeamPlayerInfo
	if r.isTeamDraft && len(r.roomPlayers) > 0 {
		teamPlayers = make([]TeamPlayerInfo, 0, len(r.roomPlayers))
		for _, p := range r.roomPlayers {
			teamPlayers = append(teamPlayers, TeamPlayerInfo{
				ID:           p.UserID.String(),
				DisplayName:  p.DisplayName,
				Team:         string(p.Team),
				AssignedRole: string(p.AssignedRole),
				IsCaptain:    p.IsCaptain,
			})
		}
	}

	// Build pending edit info from EditManager
	pendingEditInfo := r.editMgr.BuildPendingEditInfo()

	// Get paused by display name
	pausedByName := ""
	if pausedByID := r.pauseMgr.GetPausedBy(); pausedByID != nil {
		pausedByName = r.getUserDisplayName(*pausedByID)
	}

	msg, _ := NewMessage(MessageTypeStateSync, StateSyncPayload{
		Room: RoomInfo{
			ID:            r.id.String(),
			ShortCode:     r.shortCode,
			DraftMode:     "pro_play",
			Status:        status,
			TimerDuration: r.timerDurationMs,
		},
		Draft: DraftInfo{
			CurrentPhase:     r.getDraftState().CurrentPhase,
			CurrentTeam:      currentTeam,
			ActionType:       actionType,
			TimerRemainingMs: timerRemaining,
			BlueBans:         r.getDraftState().BlueBans,
			RedBans:          r.getDraftState().RedBans,
			BluePicks:        r.getDraftState().BluePicks,
			RedPicks:         r.getDraftState().RedPicks,
			IsComplete:       r.getDraftState().IsComplete,
			IsPaused:         r.pauseMgr.IsPaused(),
			PausedBy:         pausedByName,
			PausedBySide:     r.pauseMgr.GetPausedBySide(),
			PendingEdit:      pendingEditInfo,
			BlueResumeReady:  func() bool { b, _ := r.pauseMgr.GetResumeReady(); return b }(),
			RedResumeReady:   func() bool { _, r := r.pauseMgr.GetResumeReady(); return r }(),
			ResumeCountdown:  r.pauseMgr.GetResumeCountdown(),
		},
		Players: PlayersInfo{
			Blue: bluePlayer,
			Red:  redPlayer,
		},
		YourSide:       client.side,
		IsCaptain:      isCaptain,
		IsTeamDraft:    r.isTeamDraft,
		TeamPlayers:    teamPlayers,
		SpectatorCount: len(r.spectators),
	})

	client.Send(msg)
}

// isCaptain checks if a user is a captain for the given side
func (r *Room) isCaptain(userID uuid.UUID, side string) bool {
	if r.isTeamDraft {
		if side == "blue" && r.blueCaptainID != nil {
			return userID == *r.blueCaptainID
		}
		if side == "red" && r.redCaptainID != nil {
			return userID == *r.redCaptainID
		}
		return false
	}
	// In 1v1 mode, both players are effectively captains
	return side == "blue" || side == "red"
}

// handlePauseDraft handles a pause request from a client
func (r *Room) handlePauseDraft(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate: draft started, not complete, not already paused
	if !r.getDraftState().Started || r.getDraftState().IsComplete || r.pauseMgr.IsPaused() {
		client.sendError("INVALID_STATE", "Cannot pause at this time")
		return
	}

	// Authorization: captains only in team mode
	if r.isTeamDraft && !r.isCaptain(client.userID, client.side) {
		client.sendError("UNAUTHORIZED", "Only captains can pause")
		return
	}

	// In 1v1 mode, only players (not spectators) can pause
	if !r.isTeamDraft && client.side != "blue" && client.side != "red" {
		client.sendError("UNAUTHORIZED", "Only players can pause")
		return
	}

	// Delegate to pause manager
	if err := r.pauseMgr.Pause(client.userID, client.side); err != nil {
		client.sendError("PAUSE_FAILED", err.Error())
		return
	}
}

// handleResumeDraft handles a resume request from a client
// Now forwards to the ready-to-resume system (clicking Resume = set ready)
func (r *Room) handleResumeDraft(client *Client) {
	r.readyToResume <- &ReadyToResumeRequest{
		Client: client,
		Ready:  true,
	}
}

// handleReadyToResume handles a ready-to-resume toggle from a client
func (r *Room) handleReadyToResume(req *ReadyToResumeRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := req.Client

	// Must be paused
	if !r.pauseMgr.IsPaused() {
		client.sendError("INVALID_STATE", "Draft is not paused")
		return
	}

	// Must be a captain
	if r.isTeamDraft && !r.isCaptain(client.userID, client.side) {
		client.sendError("UNAUTHORIZED", "Only captains can ready to resume")
		return
	}

	// In 1v1 mode, must be a player
	if !r.isTeamDraft && client.side != "blue" && client.side != "red" {
		client.sendError("UNAUTHORIZED", "Only players can ready to resume")
		return
	}

	// Delegate to pause manager
	if err := r.pauseMgr.SetResumeReady(client.userID, client.side, req.Ready); err != nil {
		client.sendError("RESUME_READY_FAILED", err.Error())
		return
	}
}

// handleProposeEdit handles an edit proposal from a client
func (r *Room) handleProposeEdit(req *ProposeEditRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := req.Client
	payload := req.Payload

	// Must be paused
	if !r.pauseMgr.IsPaused() {
		client.sendError("INVALID_STATE", "Must pause to edit")
		return
	}

	// Must be captain
	if r.isTeamDraft && !r.isCaptain(client.userID, client.side) {
		client.sendError("UNAUTHORIZED", "Only captains can propose edits")
		return
	}

	// In 1v1 mode, only players can propose edits
	if !r.isTeamDraft && client.side != "blue" && client.side != "red" {
		client.sendError("UNAUTHORIZED", "Only players can propose edits")
		return
	}

	// Delegate to EditManager
	if err := r.editMgr.ProposeEdit(client.userID, client.side, payload); err != nil {
		if editErr, ok := err.(*EditError); ok {
			client.sendError(editErr.Code, editErr.Message)
		} else {
			client.sendError("EDIT_ERROR", err.Error())
		}
		return
	}
}

// handleConfirmEdit handles an edit confirmation from a client
func (r *Room) handleConfirmEdit(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Authorization: must be captain/player
	if r.isTeamDraft && !r.isCaptain(client.userID, client.side) {
		client.sendError("UNAUTHORIZED", "Only captains can confirm edits")
		return
	}
	if !r.isTeamDraft && client.side != "blue" && client.side != "red" {
		client.sendError("UNAUTHORIZED", "Only players can confirm edits")
		return
	}

	// Delegate to EditManager
	if err := r.editMgr.ConfirmEdit(client.userID, client.side); err != nil {
		if editErr, ok := err.(*EditError); ok {
			client.sendError(editErr.Code, editErr.Message)
		} else {
			client.sendError("EDIT_ERROR", err.Error())
		}
		return
	}
}

// handleRejectEdit handles an edit rejection from a client
func (r *Room) handleRejectEdit(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Delegate to EditManager
	if err := r.editMgr.RejectEdit(client.userID, client.side); err != nil {
		if editErr, ok := err.(*EditError); ok {
			client.sendError(editErr.Code, editErr.Message)
		} else {
			client.sendError("EDIT_ERROR", err.Error())
		}
		return
	}
}

