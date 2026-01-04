package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/dom/league-draft-website/internal/api/middleware"
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type LobbyHandler struct {
	lobbyService       *service.LobbyService
	matchmakingService *service.MatchmakingService
	hub                *websocket.Hub
}

func NewLobbyHandler(lobbyService *service.LobbyService, matchmakingService *service.MatchmakingService, hub *websocket.Hub) *LobbyHandler {
	return &LobbyHandler{
		lobbyService:       lobbyService,
		matchmakingService: matchmakingService,
		hub:                hub,
	}
}

// Request/Response types
type CreateLobbyRequest struct {
	DraftMode            string `json:"draftMode"`
	TimerDurationSeconds int    `json:"timerDurationSeconds"`
}

type LobbyResponse struct {
	ID                   string              `json:"id"`
	ShortCode            string              `json:"shortCode"`
	CreatedBy            string              `json:"createdBy"`
	Status               string              `json:"status"`
	SelectedMatchOption  *int                `json:"selectedMatchOption"`
	DraftMode            string              `json:"draftMode"`
	TimerDurationSeconds int                 `json:"timerDurationSeconds"`
	RoomID               *string             `json:"roomId"`
	Players              []LobbyPlayerResponse `json:"players"`
}

type LobbyPlayerResponse struct {
	ID           string  `json:"id"`
	UserID       string  `json:"userId"`
	DisplayName  string  `json:"displayName"`
	Team         *string `json:"team"`
	AssignedRole *string `json:"assignedRole"`
	IsReady      bool    `json:"isReady"`
}

type MatchOptionResponse struct {
	OptionNumber   int                         `json:"optionNumber"`
	BlueTeamAvgMMR int                         `json:"blueTeamAvgMmr"`
	RedTeamAvgMMR  int                         `json:"redTeamAvgMmr"`
	MMRDifference  int                         `json:"mmrDifference"`
	BalanceScore   float64                     `json:"balanceScore"`
	Assignments    []AssignmentResponse        `json:"assignments"`
}

type AssignmentResponse struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	Team          string `json:"team"`
	AssignedRole  string `json:"assignedRole"`
	RoleMMR       int    `json:"roleMmr"`
	ComfortRating int    `json:"comfortRating"`
}

type SetReadyRequest struct {
	Ready bool `json:"ready"`
}

type SelectOptionRequest struct {
	OptionNumber int `json:"optionNumber"`
}

type StartDraftResponse struct {
	ID                   string  `json:"id"`
	ShortCode            string  `json:"shortCode"`
	DraftMode            string  `json:"draftMode"`
	TimerDurationSeconds int     `json:"timerDurationSeconds"`
	Status               string  `json:"status"`
	BlueSideUserID       *string `json:"blueSideUserId"`
	RedSideUserID        *string `json:"redSideUserId"`
	IsTeamDraft          bool    `json:"isTeamDraft"`
}

func (h *LobbyHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateLobbyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR [lobby.Create] failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	draftMode := domain.DraftModeProPlay
	if req.DraftMode == "fearless" {
		draftMode = domain.DraftModeFearless
	}

	lobby, err := h.lobbyService.CreateLobby(r.Context(), userID, service.CreateLobbyInput{
		DraftMode:            draftMode,
		TimerDurationSeconds: req.TimerDurationSeconds,
	})
	if err != nil {
		log.Printf("ERROR [lobby.Create] failed to create lobby: %v", err)
		http.Error(w, "Failed to create lobby", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

func (h *LobbyHandler) Get(w http.ResponseWriter, r *http.Request) {
	idOrCode := chi.URLParam(r, "idOrCode")

	lobby, err := h.lobbyService.GetLobby(r.Context(), idOrCode)
	if err != nil {
		if errors.Is(err, service.ErrLobbyNotFound) {
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR [lobby.Get] failed to get lobby: %v", err)
		http.Error(w, "Failed to get lobby", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

func (h *LobbyHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idOrCode := chi.URLParam(r, "idOrCode")
	lobby, err := h.lobbyService.GetLobby(r.Context(), idOrCode)
	if err != nil {
		if errors.Is(err, service.ErrLobbyNotFound) {
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR [lobby.Join] failed to get lobby: %v", err)
		http.Error(w, "Failed to get lobby", http.StatusInternalServerError)
		return
	}

	player, err := h.lobbyService.JoinLobby(r.Context(), lobby.ID, userID)
	if err != nil {
		if errors.Is(err, service.ErrLobbyFull) {
			http.Error(w, "Lobby is full", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrInvalidLobbyState) {
			http.Error(w, "Cannot join lobby in current state", http.StatusConflict)
			return
		}
		log.Printf("ERROR [lobby.Join] failed to join lobby: %v", err)
		http.Error(w, "Failed to join lobby", http.StatusInternalServerError)
		return
	}

	displayName := ""
	if player.User != nil {
		displayName = player.User.DisplayName
	}

	resp := LobbyPlayerResponse{
		ID:          player.ID.String(),
		UserID:      player.UserID.String(),
		DisplayName: displayName,
		IsReady:     player.IsReady,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *LobbyHandler) Leave(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idOrCode := chi.URLParam(r, "idOrCode")
	lobby, err := h.lobbyService.GetLobby(r.Context(), idOrCode)
	if err != nil {
		if errors.Is(err, service.ErrLobbyNotFound) {
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR [lobby.Leave] failed to get lobby: %v", err)
		http.Error(w, "Failed to get lobby", http.StatusInternalServerError)
		return
	}

	if err := h.lobbyService.LeaveLobby(r.Context(), lobby.ID, userID); err != nil {
		if errors.Is(err, service.ErrNotInLobby) {
			http.Error(w, "Not in lobby", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.Leave] failed to leave lobby: %v", err)
		http.Error(w, "Failed to leave lobby", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *LobbyHandler) SetReady(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idOrCode := chi.URLParam(r, "idOrCode")
	lobby, err := h.lobbyService.GetLobby(r.Context(), idOrCode)
	if err != nil {
		if errors.Is(err, service.ErrLobbyNotFound) {
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR [lobby.SetReady] failed to get lobby: %v", err)
		http.Error(w, "Failed to get lobby", http.StatusInternalServerError)
		return
	}

	var req SetReadyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.lobbyService.SetPlayerReady(r.Context(), lobby.ID, userID, req.Ready); err != nil {
		if errors.Is(err, service.ErrNotInLobby) {
			http.Error(w, "Not in lobby", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.SetReady] failed to set ready: %v", err)
		http.Error(w, "Failed to set ready status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ready": req.Ready})
}

func (h *LobbyHandler) GenerateTeams(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lobbyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid lobby ID", http.StatusBadRequest)
		return
	}

	lobby, err := h.lobbyService.GetLobby(r.Context(), lobbyID.String())
	if err != nil {
		if errors.Is(err, service.ErrLobbyNotFound) {
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR [lobby.GenerateTeams] failed to get lobby: %v", err)
		http.Error(w, "Failed to get lobby", http.StatusInternalServerError)
		return
	}

	if lobby.CreatedBy != userID {
		http.Error(w, "Only lobby creator can generate teams", http.StatusForbidden)
		return
	}

	if len(lobby.Players) != 10 {
		http.Error(w, "Lobby needs exactly 10 players", http.StatusBadRequest)
		return
	}

	// Check all players are ready and convert to pointer slice
	players := make([]*domain.LobbyPlayer, len(lobby.Players))
	for i := range lobby.Players {
		if !lobby.Players[i].IsReady {
			http.Error(w, "All players must be ready", http.StatusBadRequest)
			return
		}
		players[i] = &lobby.Players[i]
	}

	options, err := h.matchmakingService.GenerateMatchOptions(r.Context(), lobbyID, players, 5)
	if err != nil {
		log.Printf("ERROR [lobby.GenerateTeams] failed to generate teams: %v", err)
		http.Error(w, "Failed to generate teams", http.StatusInternalServerError)
		return
	}

	resp := make([]MatchOptionResponse, len(options))
	for i, opt := range options {
		resp[i] = toMatchOptionResponse(opt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *LobbyHandler) GetMatchOptions(w http.ResponseWriter, r *http.Request) {
	lobbyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid lobby ID", http.StatusBadRequest)
		return
	}

	options, err := h.lobbyService.GetMatchOptions(r.Context(), lobbyID)
	if err != nil {
		log.Printf("ERROR [lobby.GetMatchOptions] failed to get options: %v", err)
		http.Error(w, "Failed to get match options", http.StatusInternalServerError)
		return
	}

	resp := make([]MatchOptionResponse, len(options))
	for i, opt := range options {
		resp[i] = toMatchOptionResponse(opt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *LobbyHandler) SelectOption(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lobbyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid lobby ID", http.StatusBadRequest)
		return
	}

	var req SelectOptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.lobbyService.SelectMatchOption(r.Context(), lobbyID, req.OptionNumber, userID); err != nil {
		if errors.Is(err, service.ErrNotLobbyCreator) {
			http.Error(w, "Only lobby creator can select teams", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrInvalidMatchOption) {
			http.Error(w, "Invalid match option", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.SelectOption] failed to select option: %v", err)
		http.Error(w, "Failed to select option", http.StatusInternalServerError)
		return
	}

	// Get updated lobby
	lobby, err := h.lobbyService.GetLobby(r.Context(), lobbyID.String())
	if err != nil {
		log.Printf("ERROR [lobby.SelectOption] failed to get lobby: %v", err)
		http.Error(w, "Failed to get lobby", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

func (h *LobbyHandler) StartDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lobbyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid lobby ID", http.StatusBadRequest)
		return
	}

	room, err := h.lobbyService.StartDraft(r.Context(), lobbyID, userID)
	if err != nil {
		if errors.Is(err, service.ErrLobbyNotFound) {
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrNotLobbyCreator) {
			http.Error(w, "Only lobby creator can start draft", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrInvalidLobbyState) {
			http.Error(w, "Lobby must be in team_selected state to start draft", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrNoMatchOptions) {
			http.Error(w, "No match option selected", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.StartDraft] failed to start draft: %v", err)
		http.Error(w, "Failed to start draft", http.StatusInternalServerError)
		return
	}

	// Create WebSocket room for the draft
	h.hub.CreateRoom(room.ID, room.ShortCode, room.TimerDurationSeconds*1000)

	// Build response
	var blueSideUserID, redSideUserID *string
	if room.BlueSideUserID != nil {
		id := room.BlueSideUserID.String()
		blueSideUserID = &id
	}
	if room.RedSideUserID != nil {
		id := room.RedSideUserID.String()
		redSideUserID = &id
	}

	resp := StartDraftResponse{
		ID:                   room.ID.String(),
		ShortCode:            room.ShortCode,
		DraftMode:            string(room.DraftMode),
		TimerDurationSeconds: room.TimerDurationSeconds,
		Status:               string(room.Status),
		BlueSideUserID:       blueSideUserID,
		RedSideUserID:        redSideUserID,
		IsTeamDraft:          room.IsTeamDraft,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Helper functions
func toLobbyResponse(lobby *domain.Lobby) LobbyResponse {
	var roomID *string
	if lobby.RoomID != nil {
		s := lobby.RoomID.String()
		roomID = &s
	}

	players := make([]LobbyPlayerResponse, len(lobby.Players))
	for i, p := range lobby.Players {
		var team, role *string
		if p.Team != nil {
			s := string(*p.Team)
			team = &s
		}
		if p.AssignedRole != nil {
			s := string(*p.AssignedRole)
			role = &s
		}

		displayName := ""
		if p.User != nil {
			displayName = p.User.DisplayName
		}

		players[i] = LobbyPlayerResponse{
			ID:           p.ID.String(),
			UserID:       p.UserID.String(),
			DisplayName:  displayName,
			Team:         team,
			AssignedRole: role,
			IsReady:      p.IsReady,
		}
	}

	return LobbyResponse{
		ID:                   lobby.ID.String(),
		ShortCode:            lobby.ShortCode,
		CreatedBy:            lobby.CreatedBy.String(),
		Status:               string(lobby.Status),
		SelectedMatchOption:  lobby.SelectedMatchOption,
		DraftMode:            string(lobby.DraftMode),
		TimerDurationSeconds: lobby.TimerDurationSeconds,
		RoomID:               roomID,
		Players:              players,
	}
}

func toMatchOptionResponse(opt *domain.MatchOption) MatchOptionResponse {
	assignments := make([]AssignmentResponse, len(opt.Assignments))
	for i, a := range opt.Assignments {
		displayName := ""
		if a.User != nil {
			displayName = a.User.DisplayName
		}
		assignments[i] = AssignmentResponse{
			UserID:        a.UserID.String(),
			DisplayName:   displayName,
			Team:          string(a.Team),
			AssignedRole:  string(a.AssignedRole),
			RoleMMR:       a.RoleMMR,
			ComfortRating: a.ComfortRating,
		}
	}

	return MatchOptionResponse{
		OptionNumber:   opt.OptionNumber,
		BlueTeamAvgMMR: opt.BlueTeamAvgMMR,
		RedTeamAvgMMR:  opt.RedTeamAvgMMR,
		MMRDifference:  opt.MMRDifference,
		BalanceScore:   opt.BalanceScore,
		Assignments:    assignments,
	}
}
