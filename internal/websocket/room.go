package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

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

	// Draft state
	draftState     *DraftState
	currentHover   map[string]*string // side -> championId
	timer          *time.Timer
	timerStartedAt time.Time

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

	mu sync.RWMutex
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

func NewRoom(id uuid.UUID, shortCode string, timerDurationMs int, userRepo repository.UserRepository) *Room {
	return &Room{
		id:              id,
		shortCode:       shortCode,
		clients:         make(map[*Client]bool),
		spectators:      make(map[*Client]bool),
		timerDurationMs: timerDurationMs,
		userRepo:        userRepo,
		draftState: &DraftState{
			CurrentPhase: 0,
			BlueBans:     []string{},
			RedBans:      []string{},
			BluePicks:    []string{},
			RedPicks:     []string{},
		},
		currentHover:   make(map[string]*string),
		join:           make(chan *Client),
		leave:          make(chan *Client),
		broadcast:      make(chan *Message),
		selectChampion: make(chan *SelectChampionRequest),
		lockIn:         make(chan *Client),
		hoverChampion:  make(chan *HoverChampionRequest),
		ready:          make(chan *ReadyRequest),
		startDraft:     make(chan *Client),
		syncState:      make(chan *Client),
	}
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
		}
	}
}

func (r *Room) handleJoin(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clients[client] = true

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

	// Check if it's this client's turn
	if (phase.Team == domain.SideBlue && req.Client != r.blueClient) ||
		(phase.Team == domain.SideRed && req.Client != r.redClient) {
		req.Client.sendError("NOT_YOUR_TURN", "It's not your turn")
		return
	}

	// Check if champion is already picked/banned
	if r.isChampionUsed(req.ChampionID) {
		req.Client.sendError("CHAMPION_UNAVAILABLE", "Champion is already picked or banned")
		return
	}

	// Store selection (will be confirmed on lock in)
	r.currentHover[string(phase.Team)] = &req.ChampionID

	// Broadcast hover
	msg, _ := NewMessage(MessageTypeChampionHovered, ChampionHoveredPayload{
		Side:       string(phase.Team),
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

	// Check if it's this client's turn
	if (phase.Team == domain.SideBlue && client != r.blueClient) ||
		(phase.Team == domain.SideRed && client != r.redClient) {
		client.sendError("NOT_YOUR_TURN", "It's not your turn")
		return
	}

	championID := r.currentHover[string(phase.Team)]
	if championID == nil {
		client.sendError("NO_SELECTION", "No champion selected")
		return
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

	r.startTimer()
}

func (r *Room) startTimer() {
	r.timerStartedAt = time.Now()

	r.timer = time.AfterFunc(time.Duration(r.timerDurationMs)*time.Millisecond, func() {
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

		elapsed := time.Since(r.timerStartedAt)
		remaining := r.timerDurationMs - int(elapsed.Milliseconds())
		if remaining < 0 {
			remaining = 0
		}
		r.mu.RUnlock()

		msg, _ := NewMessage(MessageTypeTimerTick, TimerTickPayload{
			RemainingMs: remaining,
		})
		r.broadcast <- msg

		if remaining <= 0 {
			return
		}
	}
}

func (r *Room) handleTimerExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.draftState.IsComplete {
		return
	}

	phase := domain.GetPhase(r.draftState.CurrentPhase)
	if phase == nil {
		return
	}

	// Auto-select random available champion
	// For now, just skip (in production, pick random)
	msg, _ := NewMessage(MessageTypeTimerExpired, TimerExpiredPayload{
		Phase:        r.draftState.CurrentPhase,
		AutoSelected: nil,
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

func (r *Room) broadcastMessage(msg *Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.broadcastMessageLocked(msg)
}

func (r *Room) broadcastMessageLocked(msg *Message) {
	data, _ := json.Marshal(msg)
	for client := range r.clients {
		select {
		case client.send <- data:
		default:
			log.Printf("client send buffer full, skipping")
		}
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
		elapsed := time.Since(r.timerStartedAt)
		timerRemaining = r.timerDurationMs - int(elapsed.Milliseconds())
		if timerRemaining < 0 {
			timerRemaining = 0
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
		},
		Players: PlayersInfo{
			Blue: bluePlayer,
			Red:  redPlayer,
		},
		YourSide:       client.side,
		SpectatorCount: len(r.spectators),
	})

	client.Send(msg)
}
