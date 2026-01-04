package websocket

import (
	"log"
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	rooms      map[string]*Room
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	joinRoom   chan *JoinRoomRequest
	mu         sync.RWMutex
}

type JoinRoomRequest struct {
	Client *Client
	RoomID string
	Side   string
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		joinRoom:   make(chan *JoinRoomRequest),
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

	req.Client.side = req.Side
	req.Client.room = room
	room.join <- req.Client
}

func (h *Hub) CreateRoom(roomID uuid.UUID, shortCode string, timerDurationMs int) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	room := NewRoom(roomID, shortCode, timerDurationMs)
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
