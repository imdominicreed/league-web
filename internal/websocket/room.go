package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

const bufferDurationMs = 5000 // 5 second buffer after timer hits 0

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

	// Team draft mode (5v5)
	isTeamDraft      bool
	roomPlayers      map[uuid.UUID]*domain.RoomPlayer // userId -> RoomPlayer
	blueCaptainID    *uuid.UUID
	redCaptainID     *uuid.UUID
	blueTeamClients  map[*Client]bool // Non-captain blue team members
	redTeamClients   map[*Client]bool // Non-captain red team members

	// Draft state
	draftState     *DraftState
	currentHover   map[string]*string // side -> championId
	timer          *time.Timer
	timerStartedAt time.Time

	// Pause state
	isPaused           bool
	pausedBy           *uuid.UUID
	pausedBySide       string
	pausedAt           time.Time
	pauseRemainingMs   int         // Timer value when paused
	pauseTimer         *time.Timer // For auto-resume
	maxPauseDurationMs int         // 5 minutes default

	// Edit state
	pendingEdit  *PendingEdit
	editTimeout  *time.Timer

	// Resume ready state (during pause)
	blueResumeReady       bool
	redResumeReady        bool
	resumeCountdown       int
	resumeCountdownCancel chan struct{}

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

// PendingEdit represents an edit proposal awaiting confirmation
type PendingEdit struct {
	ProposedBy    uuid.UUID
	ProposedSide  string
	SlotType      string // "ban" or "pick"
	Team          string // "blue" or "red"
	SlotIndex     int
	OldChampionID string
	NewChampionID string
	ExpiresAt     time.Time
}

// ProposeEditRequest contains the client and payload for an edit proposal
type ProposeEditRequest struct {
	Client  *Client
	Payload ProposeEditPayload
}

type DraftState struct {
	CurrentPhase int
	BlueBans     []string
	RedBans      []string
	BluePicks    []string
	RedPicks     []string
	IsComplete   bool
	BlueReady    bool
	RedReady     bool
	Started      bool
}

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

func NewRoom(id uuid.UUID, shortCode string, timerDurationMs int, userRepo repository.UserRepository, championRepo repository.ChampionRepository) *Room {
	return &Room{
		id:               id,
		shortCode:        shortCode,
		clients:          make(map[*Client]bool),
		spectators:       make(map[*Client]bool),
		blueTeamClients:  make(map[*Client]bool),
		redTeamClients:   make(map[*Client]bool),
		timerDurationMs:  timerDurationMs,
		userRepo:         userRepo,
		championRepo:     championRepo,
		draftState: &DraftState{
			CurrentPhase: 0,
			BlueBans:     []string{},
			RedBans:      []string{},
			BluePicks:    []string{},
			RedPicks:     []string{},
		},
		currentHover:       make(map[string]*string),
		maxPauseDurationMs: 300000, // 5 minutes
		join:               make(chan *Client),
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

func (r *Room) Run() {
	for {
		select {
		case client := <-r.join:
			r.handleJoin(client)

		case client := <-r.leave:
			r.handleLeave(client)

		case msg := <-r.broadcast:
			r.broadcastMessage(msg)

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
	r.broadcastPlayerUpdate(client.side, &PlayerInfo{
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
		r.draftState.BlueReady = false
	}
	if r.redClient == client {
		r.redClient = nil
		r.draftState.RedReady = false
	}

	r.broadcastPlayerUpdate(client.side, nil, "left")
}

func (r *Room) handleSelectChampion(req *SelectChampionRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.draftState.Started || r.draftState.IsComplete {
		req.Client.sendError("INVALID_STATE", "Draft not in progress")
		return
	}

	phase := domain.GetPhase(r.draftState.CurrentPhase)
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
	if r.isChampionUsed(req.ChampionID) {
		req.Client.sendError("CHAMPION_UNAVAILABLE", "Champion is already picked or banned")
		return
	}

	// Store selection (will be confirmed on lock in)
	r.currentHover[currentSide] = &req.ChampionID

	// Broadcast hover
	msg, _ := NewMessage(MessageTypeChampionHovered, ChampionHoveredPayload{
		Side:       currentSide,
		ChampionID: &req.ChampionID,
	})
	r.broadcastMessageLocked(msg)
}

func (r *Room) handleLockIn(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.draftState.Started || r.draftState.IsComplete {
		client.sendError("INVALID_STATE", "Draft not in progress")
		return
	}

	phase := domain.GetPhase(r.draftState.CurrentPhase)
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

	championID := r.currentHover[currentSide]
	if championID == nil {
		none := "None"
		championID = &none

	}

	// Apply the selection
	r.applySelection(phase, *championID)

	// Stop current timer
	if r.timer != nil {
		r.timer.Stop()
	}

	// Broadcast selection
	msg, _ := NewMessage(MessageTypeChampionSelected, ChampionSelectedPayload{
		Phase:      r.draftState.CurrentPhase,
		Team:       string(phase.Team),
		ActionType: string(phase.ActionType),
		ChampionID: *championID,
	})
	r.broadcastMessageLocked(msg)

	// Move to next phase
	r.advancePhase()
}

func (r *Room) handleHoverChampion(req *HoverChampionRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.draftState.Started {
		return
	}

	msg, _ := NewMessage(MessageTypeChampionHovered, ChampionHoveredPayload{
		Side:       req.Client.side,
		ChampionID: req.ChampionID,
	})
	r.broadcastMessageLocked(msg)
}

func (r *Room) handleReady(req *ReadyRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.draftState.Started {
		return
	}

	switch req.Client.side {
	case "blue":
		r.draftState.BlueReady = req.Ready
	case "red":
		r.draftState.RedReady = req.Ready
	}

	req.Client.ready = req.Ready

	r.broadcastPlayerUpdate(req.Client.side, &PlayerInfo{
		UserID:      req.Client.userID.String(),
		DisplayName: r.getUserDisplayName(req.Client.userID),
		Ready:       req.Ready,
	}, "ready_changed")
}

func (r *Room) handleStartDraft(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.draftState.Started {
		client.sendError("ALREADY_STARTED", "Draft already started")
		return
	}

	if r.blueClient == nil || r.redClient == nil {
		client.sendError("MISSING_PLAYERS", "Both sides must have a player")
		return
	}

	if !r.draftState.BlueReady || !r.draftState.RedReady {
		client.sendError("NOT_READY", "Both players must be ready")
		return
	}

	r.draftState.Started = true
	r.timerStartedAt = time.Now()

	phase := domain.GetPhase(0)

	msg, _ := NewMessage(MessageTypeDraftStarted, DraftStartedPayload{
		CurrentPhase:     0,
		CurrentTeam:      string(phase.Team),
		ActionType:       string(phase.ActionType),
		TimerRemainingMs: r.timerDurationMs,
	})
	r.broadcastMessageLocked(msg)

	// Send STATE_SYNC to ensure room.status updates to 'in_progress'
	for client := range r.clients {
		r.sendStateSyncLocked(client)
	}

	r.startTimer()
}

func (r *Room) startTimer() {
	r.timerStartedAt = time.Now()

	// Timer fires after main duration + buffer period
	totalDuration := r.timerDurationMs + bufferDurationMs
	r.timer = time.AfterFunc(time.Duration(totalDuration)*time.Millisecond, func() {
		r.handleTimerExpired()
	})

	// Start ticker for timer updates
	go r.runTimerTicker()
}

func (r *Room) runTimerTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.RLock()
		if !r.draftState.Started || r.draftState.IsComplete {
			r.mu.RUnlock()
			return
		}

		// Skip tick if paused
		if r.isPaused {
			r.mu.RUnlock()
			continue
		}

		elapsed := time.Since(r.timerStartedAt)
		remaining := r.timerDurationMs - int(elapsed.Milliseconds())

		// Check if we're in the buffer period (past main timer but before auto-lock)
		isBufferPeriod := remaining <= 0

		// Display 0 during buffer period (don't show negative)
		displayRemaining := remaining
		if displayRemaining < 0 {
			displayRemaining = 0
		}
		r.mu.RUnlock()

		msg, _ := NewMessage(MessageTypeTimerTick, TimerTickPayload{
			RemainingMs:    displayRemaining,
			IsBufferPeriod: isBufferPeriod,
		})
		r.broadcast <- msg

		// Stop ticker after buffer period expires
		if remaining <= -bufferDurationMs {
			return
		}
	}
}

func (r *Room) handleTimerExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.draftState.IsComplete || r.isPaused {
		return
	}

	phase := domain.GetPhase(r.draftState.CurrentPhase)
	if phase == nil {
		return
	}

	var championID string

	if phase.ActionType == domain.ActionTypeBan {
		// Missed ban - use "None" (skip the ban)
		championID = "None"
	} else {
		// Missed pick - select a random available champion
		championID = r.getRandomAvailableChampion()
	}

	r.applySelection(phase, championID)

	// Stop current timer
	if r.timer != nil {
		r.timer.Stop()
	}

	// Broadcast selection
	msg, _ := NewMessage(MessageTypeChampionSelected, ChampionSelectedPayload{
		Phase:      r.draftState.CurrentPhase,
		Team:       string(phase.Team),
		ActionType: string(phase.ActionType),
		ChampionID: championID,
	})
	r.broadcastMessageLocked(msg)

	r.advancePhase()
}

func (r *Room) advancePhase() {
	r.draftState.CurrentPhase++

	// Clear hover for next phase
	r.currentHover = make(map[string]*string)

	if r.draftState.CurrentPhase >= domain.TotalPhases() {
		r.draftState.IsComplete = true

		msg, _ := NewMessage(MessageTypeDraftCompleted, DraftCompletedPayload{
			BlueBans:  r.draftState.BlueBans,
			RedBans:   r.draftState.RedBans,
			BluePicks: r.draftState.BluePicks,
			RedPicks:  r.draftState.RedPicks,
		})
		r.broadcastMessageLocked(msg)

		// Send STATE_SYNC to ensure room.status updates to 'completed'
		for client := range r.clients {
			r.sendStateSyncLocked(client)
		}

		return
	}

	phase := domain.GetPhase(r.draftState.CurrentPhase)

	msg, _ := NewMessage(MessageTypePhaseChanged, PhaseChangedPayload{
		CurrentPhase:     r.draftState.CurrentPhase,
		CurrentTeam:      string(phase.Team),
		ActionType:       string(phase.ActionType),
		TimerRemainingMs: r.timerDurationMs,
	})
	r.broadcastMessageLocked(msg)

	r.startTimer()
}

func (r *Room) applySelection(phase *domain.Phase, championID string) {
	switch phase.ActionType {
	case domain.ActionTypeBan:
		if phase.Team == domain.SideBlue {
			r.draftState.BlueBans = append(r.draftState.BlueBans, championID)
		} else {
			r.draftState.RedBans = append(r.draftState.RedBans, championID)
		}
	case domain.ActionTypePick:
		if phase.Team == domain.SideBlue {
			r.draftState.BluePicks = append(r.draftState.BluePicks, championID)
		} else {
			r.draftState.RedPicks = append(r.draftState.RedPicks, championID)
		}
	}
}

func (r *Room) isChampionUsed(championID string) bool {
	for _, id := range r.draftState.BlueBans {
		if id == championID {
			return true
		}
	}
	for _, id := range r.draftState.RedBans {
		if id == championID {
			return true
		}
	}
	for _, id := range r.draftState.BluePicks {
		if id == championID {
			return true
		}
	}
	for _, id := range r.draftState.RedPicks {
		if id == championID {
			return true
		}
	}
	return false
}

// getRandomAvailableChampion returns a random champion that hasn't been picked or banned
func (r *Room) getRandomAvailableChampion() string {
	if r.championRepo == nil {
		log.Printf("Warning: championRepo is nil, cannot get random champion")
		return "None"
	}

	champions, err := r.championRepo.GetAll(context.Background())
	if err != nil {
		log.Printf("Error getting champions: %v", err)
		return "None"
	}

	// Filter out used champions
	var available []string
	for _, c := range champions {
		if !r.isChampionUsed(c.ID) {
			available = append(available, c.ID)
		}
	}

	if len(available) == 0 {
		log.Printf("Warning: no available champions for random pick")
		return "None"
	}

	// Pick a random one
	return available[rand.Intn(len(available))]
}

func (r *Room) broadcastMessage(msg *Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.broadcastMessageLocked(msg)
}

func (r *Room) broadcastMessageLocked(msg *Message) {
	data, _ := json.Marshal(msg)
	for client := range r.clients {
		r.trySend(client, data)
	}
}

// trySend attempts to send to a client, safely handling closed channels
func (r *Room) trySend(client *Client, data []byte) {
	defer func() {
		if recover() != nil {
			// Channel closed, client is disconnecting - skip silently
		}
	}()

	select {
	case client.send <- data:
	default:
		// Buffer full, skip
	}
}

func (r *Room) broadcastPlayerUpdate(side string, player *PlayerInfo, action string) {
	msg, _ := NewMessage(MessageTypePlayerUpdate, PlayerUpdatePayload{
		Side:   side,
		Player: player,
		Action: action,
	})
	r.broadcastMessageLocked(msg)
}

func (r *Room) sendStateSync(client *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.sendStateSyncLocked(client)
}

func (r *Room) sendStateSyncLocked(client *Client) {
	var currentTeam, actionType string
	timerRemaining := r.timerDurationMs

	if phase := domain.GetPhase(r.draftState.CurrentPhase); phase != nil {
		currentTeam = string(phase.Team)
		actionType = string(phase.ActionType)
	}

	if r.draftState.Started && !r.draftState.IsComplete {
		if r.isPaused {
			// When paused, use the frozen timer value
			timerRemaining = r.pauseRemainingMs
		} else {
			elapsed := time.Since(r.timerStartedAt)
			timerRemaining = r.timerDurationMs - int(elapsed.Milliseconds())
			if timerRemaining < 0 {
				timerRemaining = 0
			}
		}
	}

	var bluePlayer, redPlayer *PlayerInfo
	if r.blueClient != nil {
		bluePlayer = &PlayerInfo{
			UserID:      r.blueClient.userID.String(),
			DisplayName: r.getUserDisplayName(r.blueClient.userID),
			Ready:       r.draftState.BlueReady,
		}
	}
	if r.redClient != nil {
		redPlayer = &PlayerInfo{
			UserID:      r.redClient.userID.String(),
			DisplayName: r.getUserDisplayName(r.redClient.userID),
			Ready:       r.draftState.RedReady,
		}
	}

	status := "waiting"
	if r.draftState.Started {
		if r.draftState.IsComplete {
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

	// Build pending edit info if exists
	var pendingEditInfo *PendingEditInfo
	if r.pendingEdit != nil {
		pendingEditInfo = &PendingEditInfo{
			ProposedBy:    r.getUserDisplayName(r.pendingEdit.ProposedBy),
			ProposedSide:  r.pendingEdit.ProposedSide,
			SlotType:      r.pendingEdit.SlotType,
			Team:          r.pendingEdit.Team,
			SlotIndex:     r.pendingEdit.SlotIndex,
			OldChampionID: r.pendingEdit.OldChampionID,
			NewChampionID: r.pendingEdit.NewChampionID,
			ExpiresAt:     r.pendingEdit.ExpiresAt.UnixMilli(),
		}
	}

	// Get paused by display name
	pausedByName := ""
	if r.pausedBy != nil {
		pausedByName = r.getUserDisplayName(*r.pausedBy)
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
			CurrentPhase:     r.draftState.CurrentPhase,
			CurrentTeam:      currentTeam,
			ActionType:       actionType,
			TimerRemainingMs: timerRemaining,
			BlueBans:         r.draftState.BlueBans,
			RedBans:          r.draftState.RedBans,
			BluePicks:        r.draftState.BluePicks,
			RedPicks:         r.draftState.RedPicks,
			IsComplete:       r.draftState.IsComplete,
			IsPaused:         r.isPaused,
			PausedBy:         pausedByName,
			PausedBySide:     r.pausedBySide,
			PendingEdit:      pendingEditInfo,
			BlueResumeReady:  r.blueResumeReady,
			RedResumeReady:   r.redResumeReady,
			ResumeCountdown:  r.resumeCountdown,
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
	if !r.draftState.Started || r.draftState.IsComplete || r.isPaused {
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

	// Stop current timer and calculate remaining
	if r.timer != nil {
		r.timer.Stop()
	}
	elapsed := time.Since(r.timerStartedAt)
	r.pauseRemainingMs = r.timerDurationMs - int(elapsed.Milliseconds())
	if r.pauseRemainingMs < 0 {
		r.pauseRemainingMs = 0
	}

	// Set pause state
	r.isPaused = true
	r.pausedBy = &client.userID
	r.pausedBySide = client.side
	r.pausedAt = time.Now()

	// Reset resume-ready state
	r.blueResumeReady = false
	r.redResumeReady = false
	r.resumeCountdown = 0

	// Start auto-resume timer (5 minutes)
	r.pauseTimer = time.AfterFunc(
		time.Duration(r.maxPauseDurationMs)*time.Millisecond,
		func() { r.handleAutoResume() },
	)

	log.Printf("Draft paused by %s (%s side), timer frozen at %dms", client.userID, client.side, r.pauseRemainingMs)

	// Broadcast pause
	msg, _ := NewMessage(MessageTypeDraftPaused, DraftPausedPayload{
		PausedBy:         r.getUserDisplayName(client.userID),
		PausedBySide:     client.side,
		TimerFrozenAt:    r.pauseRemainingMs,
		MaxPauseDuration: r.maxPauseDurationMs,
	})
	r.broadcastMessageLocked(msg)
}

// handleResumeDraft handles a resume request from a client
// Now forwards to the ready-to-resume system (clicking Resume = set ready)
func (r *Room) handleResumeDraft(client *Client) {
	r.readyToResume <- &ReadyToResumeRequest{
		Client: client,
		Ready:  true,
	}
}

// handleAutoResume is called when the pause timer expires
func (r *Room) handleAutoResume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isPaused {
		return
	}

	// Cancel any ongoing countdown
	if r.resumeCountdownCancel != nil {
		close(r.resumeCountdownCancel)
		r.resumeCountdownCancel = nil
	}

	// Clear any pending edit
	r.clearPendingEdit()

	// Save remaining time before clearing pause state
	remainingMs := r.pauseRemainingMs

	// Clear pause state and resume-ready state
	r.isPaused = false
	r.pausedBy = nil
	r.pausedBySide = ""
	r.blueResumeReady = false
	r.redResumeReady = false
	r.resumeCountdown = 0

	log.Printf("Draft auto-resumed after pause timeout, timer restarting from %dms", remainingMs)

	// Broadcast resume
	msg, _ := NewMessage(MessageTypeDraftResumed, DraftResumedPayload{
		ResumedBy:        "System (timeout)",
		TimerRemainingMs: remainingMs,
	})
	r.broadcastMessageLocked(msg)

	// Restart timer from saved position
	r.timerStartedAt = time.Now()
	r.timerDurationMs = remainingMs
	r.startTimer()
}

// handleReadyToResume handles a ready-to-resume toggle from a client
func (r *Room) handleReadyToResume(req *ReadyToResumeRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := req.Client

	// Must be paused
	if !r.isPaused {
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

	// If countdown is in progress and someone un-readies, cancel it
	if r.resumeCountdown > 0 && !req.Ready {
		r.cancelResumeCountdown(client)
		return
	}

	// Update ready state
	if client.side == "blue" {
		r.blueResumeReady = req.Ready
	} else {
		r.redResumeReady = req.Ready
	}

	// Broadcast ready update
	msg, _ := NewMessage(MessageTypeResumeReadyUpdate, ResumeReadyUpdatePayload{
		BlueReady: r.blueResumeReady,
		RedReady:  r.redResumeReady,
	})
	r.broadcastMessageLocked(msg)

	// Check if both ready - start countdown
	if r.blueResumeReady && r.redResumeReady && r.resumeCountdown == 0 {
		r.startResumeCountdown()
	}
}

// startResumeCountdown starts the 5-second countdown before resuming
func (r *Room) startResumeCountdown() {
	r.resumeCountdown = 5
	r.resumeCountdownCancel = make(chan struct{})

	// Broadcast initial countdown
	msg, _ := NewMessage(MessageTypeResumeCountdown, ResumeCountdownPayload{
		SecondsRemaining: 5,
	})
	r.broadcastMessageLocked(msg)

	log.Printf("Resume countdown started (5 seconds)")

	// Start countdown in goroutine
	go r.runResumeCountdownTicker()
}

// runResumeCountdownTicker ticks down the resume countdown
func (r *Room) runResumeCountdownTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.resumeCountdownCancel:
			return
		case <-ticker.C:
			r.mu.Lock()

			if r.resumeCountdown <= 0 {
				r.mu.Unlock()
				return
			}

			r.resumeCountdown--

			if r.resumeCountdown <= 0 {
				// Countdown complete - resume draft
				r.doResumeDraft()
				r.mu.Unlock()
				return
			}

			// Broadcast countdown tick
			msg, _ := NewMessage(MessageTypeResumeCountdown, ResumeCountdownPayload{
				SecondsRemaining: r.resumeCountdown,
			})
			r.broadcastMessageLocked(msg)
			r.mu.Unlock()
		}
	}
}

// cancelResumeCountdown cancels an ongoing resume countdown
func (r *Room) cancelResumeCountdown(client *Client) {
	// Stop countdown goroutine
	if r.resumeCountdownCancel != nil {
		close(r.resumeCountdownCancel)
		r.resumeCountdownCancel = nil
	}

	// Reset state
	r.resumeCountdown = 0
	r.blueResumeReady = false
	r.redResumeReady = false

	log.Printf("Resume countdown cancelled by %s", client.userID)

	// Broadcast cancellation
	msg, _ := NewMessage(MessageTypeResumeCountdown, ResumeCountdownPayload{
		SecondsRemaining: 0,
		CancelledBy:      r.getUserDisplayName(client.userID),
	})
	r.broadcastMessageLocked(msg)

	// Broadcast ready update (both false)
	readyMsg, _ := NewMessage(MessageTypeResumeReadyUpdate, ResumeReadyUpdatePayload{
		BlueReady: false,
		RedReady:  false,
	})
	r.broadcastMessageLocked(readyMsg)
}

// doResumeDraft actually resumes the draft after countdown completes
func (r *Room) doResumeDraft() {
	// Clear any pending edit
	r.clearPendingEdit()

	// Stop auto-resume timer
	if r.pauseTimer != nil {
		r.pauseTimer.Stop()
	}

	// Save remaining time
	remainingMs := r.pauseRemainingMs

	// Clear pause and resume-ready state
	r.isPaused = false
	r.pausedBy = nil
	r.pausedBySide = ""
	r.blueResumeReady = false
	r.redResumeReady = false
	r.resumeCountdown = 0

	log.Printf("Draft resumed after countdown, timer restarting from %dms", remainingMs)

	// Broadcast resume
	msg, _ := NewMessage(MessageTypeDraftResumed, DraftResumedPayload{
		ResumedBy:        "Both players ready",
		TimerRemainingMs: remainingMs,
	})
	r.broadcastMessageLocked(msg)

	// Restart timer from saved position
	r.timerStartedAt = time.Now()
	r.timerDurationMs = remainingMs
	r.startTimer()
}

// handleProposeEdit handles an edit proposal from a client
func (r *Room) handleProposeEdit(req *ProposeEditRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := req.Client
	payload := req.Payload

	// Must be paused
	if !r.isPaused {
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

	// No pending edit already
	if r.pendingEdit != nil {
		client.sendError("EDIT_PENDING", "An edit is already pending")
		return
	}

	// Validate slot exists and get old champion
	oldChampionID, err := r.getChampionAtSlot(payload.SlotType, payload.Team, payload.SlotIndex)
	if err != nil {
		client.sendError("INVALID_SLOT", err.Error())
		return
	}

	// Validate new champion is available (not already picked/banned elsewhere)
	if r.isChampionUsedExcept(payload.ChampionID, payload.SlotType, payload.Team, payload.SlotIndex) {
		client.sendError("CHAMPION_UNAVAILABLE", "Champion already used")
		return
	}

	// Create pending edit
	r.pendingEdit = &PendingEdit{
		ProposedBy:    client.userID,
		ProposedSide:  client.side,
		SlotType:      payload.SlotType,
		Team:          payload.Team,
		SlotIndex:     payload.SlotIndex,
		OldChampionID: oldChampionID,
		NewChampionID: payload.ChampionID,
		ExpiresAt:     time.Now().Add(30 * time.Second),
	}

	// Start timeout timer
	r.editTimeout = time.AfterFunc(30*time.Second, func() {
		r.handleEditTimeout()
	})

	log.Printf("Edit proposed by %s: %s %s slot %d: %s -> %s",
		client.userID, payload.Team, payload.SlotType, payload.SlotIndex,
		oldChampionID, payload.ChampionID)

	// Broadcast proposal
	msg, _ := NewMessage(MessageTypeEditProposed, EditProposedPayload{
		ProposedBy:    r.getUserDisplayName(client.userID),
		ProposedSide:  client.side,
		SlotType:      payload.SlotType,
		Team:          payload.Team,
		SlotIndex:     payload.SlotIndex,
		OldChampionID: oldChampionID,
		NewChampionID: payload.ChampionID,
		ExpiresAt:     r.pendingEdit.ExpiresAt.UnixMilli(),
	})
	r.broadcastMessageLocked(msg)
}

// handleConfirmEdit handles an edit confirmation from a client
func (r *Room) handleConfirmEdit(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pendingEdit == nil {
		client.sendError("NO_EDIT", "No edit pending")
		return
	}

	// Confirmer must be captain of opposite side
	if client.side == r.pendingEdit.ProposedSide {
		client.sendError("INVALID_CONFIRM", "Cannot confirm your own edit")
		return
	}
	if r.isTeamDraft && !r.isCaptain(client.userID, client.side) {
		client.sendError("UNAUTHORIZED", "Only captains can confirm edits")
		return
	}

	// In 1v1 mode, only the opposite player can confirm
	if !r.isTeamDraft && client.side != "blue" && client.side != "red" {
		client.sendError("UNAUTHORIZED", "Only players can confirm edits")
		return
	}

	// Apply the edit
	r.applyEdit(r.pendingEdit)

	log.Printf("Edit confirmed by %s: %s %s slot %d: %s -> %s",
		client.userID, r.pendingEdit.Team, r.pendingEdit.SlotType, r.pendingEdit.SlotIndex,
		r.pendingEdit.OldChampionID, r.pendingEdit.NewChampionID)

	// Clear pending edit
	edit := r.pendingEdit
	r.clearPendingEdit()

	// Broadcast edit applied
	msg, _ := NewMessage(MessageTypeEditApplied, EditAppliedPayload{
		SlotType:      edit.SlotType,
		Team:          edit.Team,
		SlotIndex:     edit.SlotIndex,
		OldChampionID: edit.OldChampionID,
		NewChampionID: edit.NewChampionID,
		BlueBans:      r.draftState.BlueBans,
		RedBans:       r.draftState.RedBans,
		BluePicks:     r.draftState.BluePicks,
		RedPicks:      r.draftState.RedPicks,
	})
	r.broadcastMessageLocked(msg)
}

// handleRejectEdit handles an edit rejection from a client
func (r *Room) handleRejectEdit(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pendingEdit == nil {
		client.sendError("NO_EDIT", "No edit pending")
		return
	}

	// Rejecter must be captain of opposite side
	if client.side == r.pendingEdit.ProposedSide {
		client.sendError("INVALID_REJECT", "Cannot reject your own edit")
		return
	}

	log.Printf("Edit rejected by %s", client.userID)

	// Clear pending edit
	r.clearPendingEdit()

	// Broadcast rejection
	msg, _ := NewMessage(MessageTypeEditRejected, EditRejectedPayload{
		RejectedBy:   r.getUserDisplayName(client.userID),
		RejectedSide: client.side,
	})
	r.broadcastMessageLocked(msg)
}

// handleEditTimeout is called when an edit proposal times out
func (r *Room) handleEditTimeout() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pendingEdit == nil {
		return
	}

	log.Printf("Edit proposal timed out")

	// Clear pending edit
	r.clearPendingEdit()

	// Broadcast rejection (timeout)
	msg, _ := NewMessage(MessageTypeEditRejected, EditRejectedPayload{
		RejectedBy:   "System (timeout)",
		RejectedSide: "",
	})
	r.broadcastMessageLocked(msg)
}

// clearPendingEdit clears the pending edit and stops the timeout timer
func (r *Room) clearPendingEdit() {
	if r.editTimeout != nil {
		r.editTimeout.Stop()
		r.editTimeout = nil
	}
	r.pendingEdit = nil
}

// applyEdit applies a pending edit to the draft state
func (r *Room) applyEdit(edit *PendingEdit) {
	var arr *[]string
	switch {
	case edit.SlotType == "ban" && edit.Team == "blue":
		arr = &r.draftState.BlueBans
	case edit.SlotType == "ban" && edit.Team == "red":
		arr = &r.draftState.RedBans
	case edit.SlotType == "pick" && edit.Team == "blue":
		arr = &r.draftState.BluePicks
	case edit.SlotType == "pick" && edit.Team == "red":
		arr = &r.draftState.RedPicks
	}

	if arr != nil && edit.SlotIndex < len(*arr) {
		(*arr)[edit.SlotIndex] = edit.NewChampionID
	}
}

// getChampionAtSlot returns the champion at the specified slot
func (r *Room) getChampionAtSlot(slotType, team string, slotIndex int) (string, error) {
	var arr []string
	switch {
	case slotType == "ban" && team == "blue":
		arr = r.draftState.BlueBans
	case slotType == "ban" && team == "red":
		arr = r.draftState.RedBans
	case slotType == "pick" && team == "blue":
		arr = r.draftState.BluePicks
	case slotType == "pick" && team == "red":
		arr = r.draftState.RedPicks
	default:
		return "", fmt.Errorf("invalid slot type or team")
	}

	if slotIndex < 0 || slotIndex >= len(arr) {
		return "", fmt.Errorf("slot index out of range")
	}

	return arr[slotIndex], nil
}

// isChampionUsedExcept checks if a champion is used anywhere except the specified slot
func (r *Room) isChampionUsedExcept(championID, exceptSlotType, exceptTeam string, exceptSlotIndex int) bool {
	// Check blue bans
	for i, id := range r.draftState.BlueBans {
		if id == championID {
			if exceptSlotType == "ban" && exceptTeam == "blue" && i == exceptSlotIndex {
				continue // This is the slot being edited
			}
			return true
		}
	}
	// Check red bans
	for i, id := range r.draftState.RedBans {
		if id == championID {
			if exceptSlotType == "ban" && exceptTeam == "red" && i == exceptSlotIndex {
				continue
			}
			return true
		}
	}
	// Check blue picks
	for i, id := range r.draftState.BluePicks {
		if id == championID {
			if exceptSlotType == "pick" && exceptTeam == "blue" && i == exceptSlotIndex {
				continue
			}
			return true
		}
	}
	// Check red picks
	for i, id := range r.draftState.RedPicks {
		if id == championID {
			if exceptSlotType == "pick" && exceptTeam == "red" && i == exceptSlotIndex {
				continue
			}
			return true
		}
	}
	return false
}
