package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dom/league-draft-website/internal/api/middleware"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/dom/league-draft-website/internal/websocket"
	"github.com/google/uuid"
)

type PendingActionsHandler struct {
	lobbyRepo         repository.LobbyRepository
	pendingActionRepo repository.PendingActionRepository
	hub               *websocket.Hub
}

func NewPendingActionsHandler(
	lobbyRepo repository.LobbyRepository,
	pendingActionRepo repository.PendingActionRepository,
	hub *websocket.Hub,
) *PendingActionsHandler {
	return &PendingActionsHandler{
		lobbyRepo:         lobbyRepo,
		pendingActionRepo: pendingActionRepo,
		hub:               hub,
	}
}

// Response types
type LobbyPendingActionResponse struct {
	LobbyID           string                 `json:"lobbyId"`
	LobbyCode         string                 `json:"lobbyCode"`
	LobbyName         string                 `json:"lobbyName,omitempty"`
	Action            *PendingActionResponse `json:"action"`
	NeedsYourApproval bool                   `json:"needsYourApproval"`
}

type DraftPendingActionResponse struct {
	RoomID         string `json:"roomId"`
	RoomCode       string `json:"roomCode"`
	ActionType     string `json:"actionType"`
	IsYourTurn     bool   `json:"isYourTurn"`
	CurrentPhase   int    `json:"currentPhase,omitempty"`
	TimerRemaining int    `json:"timerRemaining,omitempty"`
}

type UnifiedPendingActionsResponse struct {
	LobbyActions []LobbyPendingActionResponse  `json:"lobbyActions"`
	DraftActions []DraftPendingActionResponse `json:"draftActions"`
}

// GetAll returns all pending actions for the current user across both lobby and draft contexts
func (h *PendingActionsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := UnifiedPendingActionsResponse{
		LobbyActions: []LobbyPendingActionResponse{},
		DraftActions: []DraftPendingActionResponse{},
	}

	// Get pending lobby actions
	pendingActions, err := h.pendingActionRepo.GetPendingForUser(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR [pendingActions.GetAll] failed to get pending actions for user %s: %v", userID, err)
		// Don't fail completely, just log and continue with empty lobby actions
	} else {
		log.Printf("DEBUG [pendingActions.GetAll] found %d pending lobby actions for user %s", len(pendingActions), userID)
		for _, action := range pendingActions {
			lobbyCode := ""
			if action.Lobby != nil {
				lobbyCode = action.Lobby.ShortCode
			}

			response.LobbyActions = append(response.LobbyActions, LobbyPendingActionResponse{
				LobbyID:   action.LobbyID.String(),
				LobbyCode: lobbyCode,
				Action: &PendingActionResponse{
					ID:             action.ID.String(),
					ActionType:     string(action.ActionType),
					Status:         string(action.Status),
					ProposedByUser: action.ProposedByUser.String(),
					ProposedBySide: string(action.ProposedBySide),
					Player1ID:      uuidPtrToStringPtrForPending(action.Player1ID),
					Player2ID:      uuidPtrToStringPtrForPending(action.Player2ID),
					ApprovedByBlue: action.ApprovedByBlue,
					ApprovedByRed:  action.ApprovedByRed,
					ExpiresAt:      action.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
				},
				NeedsYourApproval: true, // The repo query already filters for actions needing user's approval
			})
		}
	}

	// Get pending draft actions from the WebSocket hub
	draftActions := h.hub.GetPendingDraftActionsForUser(userID)
	for _, action := range draftActions {
		response.DraftActions = append(response.DraftActions, DraftPendingActionResponse{
			RoomID:         action.RoomID,
			RoomCode:       action.RoomCode,
			ActionType:     action.ActionType,
			IsYourTurn:     action.IsYourTurn,
			CurrentPhase:   action.CurrentPhase,
			TimerRemaining: action.TimerRemaining,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to convert *uuid.UUID to *string
func uuidPtrToStringPtrForPending(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
