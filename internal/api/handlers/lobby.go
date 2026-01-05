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
	IsCaptain    bool    `json:"isCaptain"`
	JoinOrder    int     `json:"joinOrder"`
}

type PendingActionResponse struct {
	ID             string  `json:"id"`
	ActionType     string  `json:"actionType"`
	Status         string  `json:"status"`
	ProposedByUser string  `json:"proposedByUser"`
	ProposedBySide string  `json:"proposedBySide"`
	Player1ID      *string `json:"player1Id,omitempty"`
	Player2ID      *string `json:"player2Id,omitempty"`
	ApprovedByBlue bool    `json:"approvedByBlue"`
	ApprovedByRed  bool    `json:"approvedByRed"`
	ExpiresAt      string  `json:"expiresAt"`
}

type TeamStatsResponse struct {
	BlueTeamAvgMMR int            `json:"blueTeamAvgMmr"`
	RedTeamAvgMMR  int            `json:"redTeamAvgMmr"`
	MMRDifference  int            `json:"mmrDifference"`
	AvgBlueComfort float64        `json:"avgBlueComfort"`
	AvgRedComfort  float64        `json:"avgRedComfort"`
	LaneDiffs      map[string]int `json:"laneDiffs"`
}

type SwapRequest struct {
	Player1ID string `json:"player1Id"`
	Player2ID string `json:"player2Id"`
	SwapType  string `json:"swapType"` // "players" or "roles"
}

type PromoteCaptainRequest struct {
	UserID string `json:"userId"`
}

type KickPlayerRequest struct {
	UserID string `json:"userId"`
}

type MatchOptionResponse struct {
	OptionNumber   int                  `json:"optionNumber"`
	AlgorithmType  string               `json:"algorithmType"`
	BlueTeamAvgMMR int                  `json:"blueTeamAvgMmr"`
	RedTeamAvgMMR  int                  `json:"redTeamAvgMmr"`
	MMRDifference  int                  `json:"mmrDifference"`
	BalanceScore   float64              `json:"balanceScore"`
	AvgBlueComfort float64              `json:"avgBlueComfort"`
	AvgRedComfort  float64              `json:"avgRedComfort"`
	MaxLaneDiff    int                  `json:"maxLaneDiff"`
	Assignments    []AssignmentResponse `json:"assignments"`
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

	// Verify user is a captain
	var isCaptain bool
	for _, p := range lobby.Players {
		if p.UserID == userID && p.IsCaptain {
			isCaptain = true
			break
		}
	}
	if !isCaptain {
		http.Error(w, "Only captain can generate teams", http.StatusForbidden)
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

	options, err := h.matchmakingService.GenerateMatchOptions(r.Context(), lobbyID, players, 8)
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
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can select teams", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrNotInLobby) {
			http.Error(w, "User not in lobby", http.StatusForbidden)
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
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can start draft", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrNotInLobby) {
			http.Error(w, "User not in lobby", http.StatusForbidden)
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

// ==================== Captain Management ====================

func (h *LobbyHandler) TakeCaptain(w http.ResponseWriter, r *http.Request) {
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

	if err := h.lobbyService.TakeCaptain(r.Context(), lobbyID, userID); err != nil {
		if errors.Is(err, service.ErrNotInLobby) {
			http.Error(w, "Not in lobby", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrNotOnTeam) {
			http.Error(w, "Not on a team", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.TakeCaptain] failed: %v", err)
		http.Error(w, "Failed to take captain", http.StatusInternalServerError)
		return
	}

	// Return updated lobby
	lobby, _ := h.lobbyService.GetLobby(r.Context(), lobbyID.String())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

func (h *LobbyHandler) PromoteCaptain(w http.ResponseWriter, r *http.Request) {
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

	var req PromoteCaptainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.lobbyService.PromoteCaptain(r.Context(), lobbyID, userID, targetUserID); err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can promote", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrNotOnTeam) {
			http.Error(w, "Player not on your team", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.PromoteCaptain] failed: %v", err)
		http.Error(w, "Failed to promote captain", http.StatusInternalServerError)
		return
	}

	lobby, _ := h.lobbyService.GetLobby(r.Context(), lobbyID.String())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

func (h *LobbyHandler) KickPlayer(w http.ResponseWriter, r *http.Request) {
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

	var req KickPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.lobbyService.KickPlayer(r.Context(), lobbyID, userID, targetUserID); err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can kick", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrNotOnTeam) {
			http.Error(w, "Player not on your team", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrCannotKickSelf) {
			http.Error(w, "Cannot kick yourself", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.KickPlayer] failed: %v", err)
		http.Error(w, "Failed to kick player", http.StatusInternalServerError)
		return
	}

	lobby, _ := h.lobbyService.GetLobby(r.Context(), lobbyID.String())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

// ==================== Pending Actions ====================

func (h *LobbyHandler) ProposeSwap(w http.ResponseWriter, r *http.Request) {
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

	var req SwapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	player1ID, err := uuid.Parse(req.Player1ID)
	if err != nil {
		http.Error(w, "Invalid player1 ID", http.StatusBadRequest)
		return
	}
	player2ID, err := uuid.Parse(req.Player2ID)
	if err != nil {
		http.Error(w, "Invalid player2 ID", http.StatusBadRequest)
		return
	}

	action, err := h.lobbyService.ProposeSwap(r.Context(), lobbyID, userID, service.SwapRequest{
		Player1ID: player1ID,
		Player2ID: player2ID,
		SwapType:  req.SwapType,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can propose swap", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrPendingActionExists) {
			http.Error(w, "A pending action already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrInvalidSwap) {
			http.Error(w, "Invalid swap request", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.ProposeSwap] failed: %v", err)
		http.Error(w, "Failed to propose swap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toPendingActionResponse(action))
}

func (h *LobbyHandler) ProposeMatchmake(w http.ResponseWriter, r *http.Request) {
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

	action, err := h.lobbyService.ProposeMatchmake(r.Context(), lobbyID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can propose matchmake", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrPendingActionExists) {
			http.Error(w, "A pending action already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrNotEnoughPlayers) {
			http.Error(w, "Need 10 players", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrPlayersNotReady) {
			http.Error(w, "All players must be ready", http.StatusBadRequest)
			return
		}
		log.Printf("ERROR [lobby.ProposeMatchmake] failed: %v", err)
		http.Error(w, "Failed to propose matchmake", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toPendingActionResponse(action))
}

func (h *LobbyHandler) ProposeStartDraft(w http.ResponseWriter, r *http.Request) {
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

	action, err := h.lobbyService.ProposeStartDraft(r.Context(), lobbyID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can propose start draft", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrPendingActionExists) {
			http.Error(w, "A pending action already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrInvalidLobbyState) {
			http.Error(w, "Teams must be selected first", http.StatusConflict)
			return
		}
		log.Printf("ERROR [lobby.ProposeStartDraft] failed: %v", err)
		http.Error(w, "Failed to propose start draft", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toPendingActionResponse(action))
}

func (h *LobbyHandler) ProposeSelectOption(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		OptionNumber int `json:"optionNumber"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	action, err := h.lobbyService.ProposeSelectOption(r.Context(), lobbyID, userID, req.OptionNumber)
	if err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can propose option selection", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrPendingActionExists) {
			http.Error(w, "A pending action already exists", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrInvalidMatchOption) {
			http.Error(w, "Invalid match option", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrInvalidLobbyState) {
			http.Error(w, "Lobby must be in matchmaking status", http.StatusConflict)
			return
		}
		log.Printf("ERROR [lobby.ProposeSelectOption] failed: %v", err)
		http.Error(w, "Failed to propose option selection", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toPendingActionResponse(action))
}

func (h *LobbyHandler) ApprovePendingAction(w http.ResponseWriter, r *http.Request) {
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

	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		http.Error(w, "Invalid action ID", http.StatusBadRequest)
		return
	}

	if err := h.lobbyService.ApprovePendingAction(r.Context(), lobbyID, userID, actionID); err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can approve", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrAlreadyApproved) {
			http.Error(w, "Already approved", http.StatusConflict)
			return
		}
		if errors.Is(err, service.ErrActionExpired) {
			http.Error(w, "Action has expired", http.StatusGone)
			return
		}
		if errors.Is(err, service.ErrPendingActionNotFound) {
			http.Error(w, "Action not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR [lobby.ApprovePendingAction] failed: %v", err)
		http.Error(w, "Failed to approve action", http.StatusInternalServerError)
		return
	}

	lobby, _ := h.lobbyService.GetLobby(r.Context(), lobbyID.String())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

func (h *LobbyHandler) CancelPendingAction(w http.ResponseWriter, r *http.Request) {
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

	actionID, err := uuid.Parse(chi.URLParam(r, "actionId"))
	if err != nil {
		http.Error(w, "Invalid action ID", http.StatusBadRequest)
		return
	}

	if err := h.lobbyService.CancelPendingAction(r.Context(), lobbyID, userID, actionID); err != nil {
		if errors.Is(err, service.ErrNotCaptain) {
			http.Error(w, "Only captain can cancel", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrPendingActionNotFound) {
			http.Error(w, "Action not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR [lobby.CancelPendingAction] failed: %v", err)
		http.Error(w, "Failed to cancel action", http.StatusInternalServerError)
		return
	}

	lobby, _ := h.lobbyService.GetLobby(r.Context(), lobbyID.String())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLobbyResponse(lobby))
}

func (h *LobbyHandler) GetPendingAction(w http.ResponseWriter, r *http.Request) {
	lobbyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid lobby ID", http.StatusBadRequest)
		return
	}

	action, err := h.lobbyService.GetPendingAction(r.Context(), lobbyID)
	if err != nil {
		log.Printf("ERROR [lobby.GetPendingAction] failed: %v", err)
		http.Error(w, "Failed to get pending action", http.StatusInternalServerError)
		return
	}

	if action == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toPendingActionResponse(action))
}

// ==================== Team Stats ====================

func (h *LobbyHandler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	lobbyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid lobby ID", http.StatusBadRequest)
		return
	}

	stats, err := h.lobbyService.GetTeamStats(r.Context(), lobbyID)
	if err != nil {
		log.Printf("ERROR [lobby.GetTeamStats] failed: %v", err)
		http.Error(w, "Failed to get team stats", http.StatusInternalServerError)
		return
	}

	// Convert Role keys to strings
	laneDiffs := make(map[string]int)
	for role, diff := range stats.LaneDiffs {
		laneDiffs[string(role)] = diff
	}

	resp := TeamStatsResponse{
		BlueTeamAvgMMR: stats.BlueTeamAvgMMR,
		RedTeamAvgMMR:  stats.RedTeamAvgMMR,
		MMRDifference:  stats.MMRDifference,
		AvgBlueComfort: stats.AvgBlueComfort,
		AvgRedComfort:  stats.AvgRedComfort,
		LaneDiffs:      laneDiffs,
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
			IsCaptain:    p.IsCaptain,
			JoinOrder:    p.JoinOrder,
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

func toPendingActionResponse(action *domain.PendingAction) PendingActionResponse {
	var player1ID, player2ID *string
	if action.Player1ID != nil {
		s := action.Player1ID.String()
		player1ID = &s
	}
	if action.Player2ID != nil {
		s := action.Player2ID.String()
		player2ID = &s
	}

	return PendingActionResponse{
		ID:             action.ID.String(),
		ActionType:     string(action.ActionType),
		Status:         string(action.Status),
		ProposedByUser: action.ProposedByUser.String(),
		ProposedBySide: string(action.ProposedBySide),
		Player1ID:      player1ID,
		Player2ID:      player2ID,
		ApprovedByBlue: action.ApprovedByBlue,
		ApprovedByRed:  action.ApprovedByRed,
		ExpiresAt:      action.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
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
