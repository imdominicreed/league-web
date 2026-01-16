package websocket

import (
	"context"
	"log"
	"sync"

	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

type Hub struct {
	rooms           map[string]*Room
	clients         map[*Client]bool
	register        chan *Client
	unregister      chan *Client
	joinRoom        chan *JoinRoomRequest
	userRepo        repository.UserRepository
	roomPlayerRepo  repository.RoomPlayerRepository
	championRepo    repository.ChampionRepository
	roomRepo        repository.RoomRepository
	draftActionRepo repository.DraftActionRepository
	mu              sync.RWMutex
}

type JoinRoomRequest struct {
	Client *Client
	RoomID string
	Side   string
}

func NewHub(userRepo repository.UserRepository, roomPlayerRepo repository.RoomPlayerRepository, championRepo repository.ChampionRepository, roomRepo repository.RoomRepository, draftActionRepo repository.DraftActionRepository) *Hub {
	return &Hub{
		rooms:           make(map[string]*Room),
		clients:         make(map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		joinRoom:        make(chan *JoinRoomRequest),
		userRepo:        userRepo,
		roomPlayerRepo:  roomPlayerRepo,
		championRepo:    championRepo,
		roomRepo:        roomRepo,
		draftActionRepo: draftActionRepo,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				if client.room != nil {
					client.room.leave <- client
				}
			}
			h.mu.Unlock()

		case req := <-h.joinRoom:
			h.handleJoinRoom(req)
		}
	}
}

func (h *Hub) handleJoinRoom(req *JoinRoomRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[req.RoomID]
	if !exists {
		req.Client.sendError("ROOM_NOT_FOUND", "Room does not exist")
		return
	}

	// Leave current room if in one
	if req.Client.room != nil {
		req.Client.room.leave <- req.Client
	}

	// Parse room ID as UUID for database lookups
	roomUUID, err := uuid.Parse(req.RoomID)
	if err != nil {
		// Try to find room by short code - use the room's actual ID
		roomUUID = room.id
	}

	// Check if this is a team draft room by looking up room players
	if h.roomPlayerRepo != nil {
		ctx := context.Background()

		// Look up the user's RoomPlayer to determine their team
		roomPlayer, err := h.roomPlayerRepo.GetByRoomAndUser(ctx, roomUUID, req.Client.userID)
		if err == nil && roomPlayer != nil {
			// User is a room player - assign their side based on team
			req.Client.side = string(roomPlayer.Team)
			log.Printf("Hub: Found RoomPlayer for user %s in room %s: team=%s, isCaptain=%v",
				req.Client.userID, roomUUID, roomPlayer.Team, roomPlayer.IsCaptain)

			// Initialize team draft if not already done
			if !room.IsTeamDraft() {
				players, err := h.roomPlayerRepo.GetByRoomID(ctx, roomUUID)
				if err == nil && len(players) > 0 {
					room.InitializeTeamDraft(players)
					log.Printf("Initialized team draft for room %s with %d players", roomUUID, len(players))
				} else {
					log.Printf("Hub: Failed to get room players for %s: err=%v, count=%d", roomUUID, err, len(players))
				}
			} else {
				log.Printf("Hub: Room %s already initialized as team draft", roomUUID)
			}
		} else if req.Side == "" {
			// User is not a room player and no side specified - they are a spectator
			log.Printf("Hub: No RoomPlayer found for user %s in room %s (err=%v), using spectator", req.Client.userID, roomUUID, err)
			req.Client.side = "spectator"
		} else {
			// Use the requested side (1v1 mode or explicit side selection)
			log.Printf("Hub: No RoomPlayer for user %s, using requested side: %s", req.Client.userID, req.Side)
			req.Client.side = req.Side
		}
	} else {
		// No roomPlayerRepo available - use original behavior
		log.Printf("Hub: No roomPlayerRepo available, using side: %s", req.Side)
		req.Client.side = req.Side
	}

	req.Client.room = room
	room.join <- req.Client
}

func (h *Hub) CreateRoom(roomID uuid.UUID, shortCode string, timerDurationMs int) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	room := NewRoom(roomID, shortCode, timerDurationMs, h.userRepo, h.championRepo, h.roomRepo, h.draftActionRepo)
	h.rooms[roomID.String()] = room
	h.rooms[shortCode] = room

	go room.Run()

	log.Printf("Created room: %s (code: %s)", roomID, shortCode)
	return room
}

func (h *Hub) GetRoom(roomID string) *Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rooms[roomID]
}

func (h *Hub) DeleteRoom(roomID uuid.UUID, shortCode string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.rooms, roomID.String())
	delete(h.rooms, shortCode)
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

// DraftPendingAction represents a pending action in a draft room for a user
type DraftPendingAction struct {
	RoomID         string `json:"roomId"`
	RoomCode       string `json:"roomCode"`
	ActionType     string `json:"actionType"` // "pick", "ban", "pending_edit", "ready_to_resume"
	IsYourTurn     bool   `json:"isYourTurn"`
	CurrentPhase   int    `json:"currentPhase,omitempty"`
	TimerRemaining int    `json:"timerRemaining,omitempty"`
}

// GetPendingDraftActionsForUser returns all pending draft actions for a user across all active rooms
func (h *Hub) GetPendingDraftActionsForUser(userID uuid.UUID) []DraftPendingAction {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var actions []DraftPendingAction
	seen := make(map[string]bool) // Track room IDs to avoid duplicates (rooms are stored by both ID and short code)

	for _, room := range h.rooms {
		roomID := room.GetID().String()
		if seen[roomID] {
			continue
		}
		seen[roomID] = true

		if action := room.GetPendingActionForUser(userID); action != nil {
			actions = append(actions, *action)
		}
	}

	return actions
}
