package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dom/league-draft-website/internal/api/middleware"
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RoomHandler struct {
	roomService *service.RoomService
	hub         *websocket.Hub
}

func NewRoomHandler(roomService *service.RoomService, hub *websocket.Hub) *RoomHandler {
	return &RoomHandler{
		roomService: roomService,
		hub:         hub,
	}
}

type CreateRoomRequest struct {
	DraftMode     string `json:"draftMode"`
	TimerDuration int    `json:"timerDuration"`
}

type RoomResponse struct {
	ID                   string  `json:"id"`
	ShortCode            string  `json:"shortCode"`
	DraftMode            string  `json:"draftMode"`
	TimerDurationSeconds int     `json:"timerDurationSeconds"`
	Status               string  `json:"status"`
	BlueSideUserID       *string `json:"blueSideUserId"`
	RedSideUserID        *string `json:"redSideUserId"`
}

type JoinRoomRequest struct {
	Side string `json:"side"`
}

type JoinRoomResponse struct {
	Room         RoomResponse `json:"room"`
	YourSide     string       `json:"yourSide"`
	WebsocketURL string       `json:"websocketUrl"`
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	draftMode := domain.DraftModeProPlay
	if req.DraftMode == "fearless" {
		draftMode = domain.DraftModeFearless
	}

	timerDuration := 30
	if req.TimerDuration > 0 {
		timerDuration = req.TimerDuration
	}

	room, err := h.roomService.CreateRoom(r.Context(), service.CreateRoomInput{
		CreatedBy:     userID,
		DraftMode:     draftMode,
		TimerDuration: timerDuration,
	})
	if err != nil {
		http.Error(w, "Failed to create room", http.StatusInternalServerError)
		return
	}

	// Create WebSocket room
	h.hub.CreateRoom(room.ID, room.ShortCode, timerDuration*1000)

	resp := RoomResponse{
		ID:                   room.ID.String(),
		ShortCode:            room.ShortCode,
		DraftMode:            string(room.DraftMode),
		TimerDurationSeconds: room.TimerDurationSeconds,
		Status:               string(room.Status),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *RoomHandler) Get(w http.ResponseWriter, r *http.Request) {
	idOrCode := chi.URLParam(r, "idOrCode")

	room, err := h.roomService.GetRoom(r.Context(), idOrCode)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var blueSideUserID, redSideUserID *string
	if room.BlueSideUserID != nil {
		id := room.BlueSideUserID.String()
		blueSideUserID = &id
	}
	if room.RedSideUserID != nil {
		id := room.RedSideUserID.String()
		redSideUserID = &id
	}

	resp := RoomResponse{
		ID:                   room.ID.String(),
		ShortCode:            room.ShortCode,
		DraftMode:            string(room.DraftMode),
		TimerDurationSeconds: room.TimerDurationSeconds,
		Status:               string(room.Status),
		BlueSideUserID:       blueSideUserID,
		RedSideUserID:        redSideUserID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idOrCode := chi.URLParam(r, "idOrCode")

	var req JoinRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get room first
	room, err := h.roomService.GetRoom(r.Context(), idOrCode)
	if err != nil {
		if errors.Is(err, service.ErrRoomNotFound) {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	side := domain.Side(req.Side)

	// Support auto-assignment
	assignedSide := side
	if req.Side == "auto" {
		if room.BlueSideUserID == nil {
			assignedSide = domain.SideBlue
		} else if room.RedSideUserID == nil {
			assignedSide = domain.SideRed
		} else {
			assignedSide = domain.SideSpectator
		}
	} else if side != domain.SideBlue && side != domain.SideRed && side != domain.SideSpectator {
		assignedSide = domain.SideSpectator
	}

	room, err = h.roomService.JoinRoom(r.Context(), room.ID, userID, assignedSide)
	if err != nil {
		if errors.Is(err, service.ErrSideTaken) {
			http.Error(w, "Side is already taken", http.StatusConflict)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := JoinRoomResponse{
		Room: RoomResponse{
			ID:                   room.ID.String(),
			ShortCode:            room.ShortCode,
			DraftMode:            string(room.DraftMode),
			TimerDurationSeconds: room.TimerDurationSeconds,
			Status:               string(room.Status),
		},
		YourSide:     string(assignedSide),
		WebsocketURL: "/api/v1/ws",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *RoomHandler) GetUserRooms(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rooms, err := h.roomService.GetUserRooms(r.Context(), userID, 20, 0)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]RoomResponse, len(rooms))
	for i, room := range rooms {
		var blueSideUserID, redSideUserID *string
		if room.BlueSideUserID != nil {
			id := room.BlueSideUserID.String()
			blueSideUserID = &id
		}
		if room.RedSideUserID != nil {
			id := room.RedSideUserID.String()
			redSideUserID = &id
		}

		resp[i] = RoomResponse{
			ID:                   room.ID.String(),
			ShortCode:            room.ShortCode,
			DraftMode:            string(room.DraftMode),
			TimerDurationSeconds: room.TimerDurationSeconds,
			Status:               string(room.Status),
			BlueSideUserID:       blueSideUserID,
			RedSideUserID:        redSideUserID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *RoomHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	room, err := h.roomService.GetRoom(r.Context(), code)
	if err != nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	resp := RoomResponse{
		ID:                   room.ID.String(),
		ShortCode:            room.ShortCode,
		DraftMode:            string(room.DraftMode),
		TimerDurationSeconds: room.TimerDurationSeconds,
		Status:               string(room.Status),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
