package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/dom/league-draft-website/internal/api/middleware"
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MatchHistoryHandler struct {
	roomRepo        repository.RoomRepository
	draftStateRepo  repository.DraftStateRepository
	draftActionRepo repository.DraftActionRepository
	roomPlayerRepo  repository.RoomPlayerRepository
}

func NewMatchHistoryHandler(
	roomRepo repository.RoomRepository,
	draftStateRepo repository.DraftStateRepository,
	draftActionRepo repository.DraftActionRepository,
	roomPlayerRepo repository.RoomPlayerRepository,
) *MatchHistoryHandler {
	return &MatchHistoryHandler{
		roomRepo:        roomRepo,
		draftStateRepo:  draftStateRepo,
		draftActionRepo: draftActionRepo,
		roomPlayerRepo:  roomPlayerRepo,
	}
}

// MatchHistoryItem represents a summary of a completed match for list view
type MatchHistoryItem struct {
	ID          string           `json:"id"`
	ShortCode   string           `json:"shortCode"`
	DraftMode   string           `json:"draftMode"`
	CompletedAt string           `json:"completedAt"`
	IsTeamDraft bool             `json:"isTeamDraft"`
	YourSide    string           `json:"yourSide"`
	BluePicks   []string         `json:"bluePicks"`
	RedPicks    []string         `json:"redPicks"`
	BlueTeam    []MatchPlayerDTO `json:"blueTeam,omitempty"`
	RedTeam     []MatchPlayerDTO `json:"redTeam,omitempty"`
}

// MatchPlayerDTO represents a player in a match
type MatchPlayerDTO struct {
	UserID       string `json:"userId"`
	DisplayName  string `json:"displayName"`
	AssignedRole string `json:"assignedRole"`
	IsCaptain    bool   `json:"isCaptain"`
}

// MatchDetailResponse represents the full detail of a completed match
type MatchDetailResponse struct {
	ID                   string            `json:"id"`
	ShortCode            string            `json:"shortCode"`
	DraftMode            string            `json:"draftMode"`
	TimerDurationSeconds int               `json:"timerDurationSeconds"`
	CreatedAt            string            `json:"createdAt"`
	StartedAt            string            `json:"startedAt,omitempty"`
	CompletedAt          string            `json:"completedAt,omitempty"`
	IsTeamDraft          bool              `json:"isTeamDraft"`
	YourSide             string            `json:"yourSide"`
	BluePicks            []string          `json:"bluePicks"`
	RedPicks             []string          `json:"redPicks"`
	BlueBans             []string          `json:"blueBans"`
	RedBans              []string          `json:"redBans"`
	BlueTeam             []MatchPlayerDTO  `json:"blueTeam,omitempty"`
	RedTeam              []MatchPlayerDTO  `json:"redTeam,omitempty"`
	Actions              []DraftActionDTO  `json:"actions"`
}

// DraftActionDTO represents a single pick/ban action
type DraftActionDTO struct {
	PhaseIndex int    `json:"phaseIndex"`
	Team       string `json:"team"`
	ActionType string `json:"actionType"`
	ChampionID string `json:"championId"`
	ActionTime string `json:"actionTime"`
}

// List returns the user's completed matches
func (h *MatchHistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse pagination params
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	rooms, err := h.roomRepo.GetCompletedByUserID(r.Context(), userID, limit, offset)
	if err != nil {
		log.Printf("ERROR [matchHistory.List] failed to get completed rooms: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	items := make([]MatchHistoryItem, 0, len(rooms))
	for _, room := range rooms {
		// Get draft state for picks
		draftState, err := h.draftStateRepo.GetByRoomID(r.Context(), room.ID)
		if err != nil {
			log.Printf("WARN [matchHistory.List] failed to get draft state for room %s: %v", room.ID, err)
			continue
		}

		// Determine user's side
		yourSide := determineSide(userID, room)

		item := MatchHistoryItem{
			ID:          room.ID.String(),
			ShortCode:   room.ShortCode,
			DraftMode:   string(room.DraftMode),
			IsTeamDraft: room.IsTeamDraft,
			YourSide:    yourSide,
			BluePicks:   jsonToStringSlice(draftState.BluePicks),
			RedPicks:    jsonToStringSlice(draftState.RedPicks),
		}

		if room.CompletedAt != nil {
			item.CompletedAt = room.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		}

		// Add team players for team draft
		if room.IsTeamDraft && len(room.Players) > 0 {
			item.BlueTeam, item.RedTeam = categorizeTeamPlayers(room.Players)
		}

		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetDetail returns full details of a specific completed match
func (h *MatchHistoryHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roomIDStr := chi.URLParam(r, "roomId")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	room, draftState, err := h.roomRepo.GetByIDWithDraftState(r.Context(), roomID)
	if err != nil {
		log.Printf("ERROR [matchHistory.GetDetail] failed to get room: %v", err)
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	// Check if user has access to this match
	if !userHasAccessToRoom(userID, room) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get draft actions for timeline
	actions, err := h.draftActionRepo.GetByRoomID(r.Context(), roomID)
	if err != nil {
		log.Printf("WARN [matchHistory.GetDetail] failed to get draft actions for room %s: %v", roomID, err)
		actions = []*domain.DraftAction{}
	}

	yourSide := determineSide(userID, room)

	resp := MatchDetailResponse{
		ID:                   room.ID.String(),
		ShortCode:            room.ShortCode,
		DraftMode:            string(room.DraftMode),
		TimerDurationSeconds: room.TimerDurationSeconds,
		CreatedAt:            room.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsTeamDraft:          room.IsTeamDraft,
		YourSide:             yourSide,
	}

	if room.StartedAt != nil {
		resp.StartedAt = room.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if room.CompletedAt != nil {
		resp.CompletedAt = room.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	if draftState != nil {
		resp.BluePicks = jsonToStringSlice(draftState.BluePicks)
		resp.RedPicks = jsonToStringSlice(draftState.RedPicks)
		resp.BlueBans = jsonToStringSlice(draftState.BlueBans)
		resp.RedBans = jsonToStringSlice(draftState.RedBans)
	}

	if room.IsTeamDraft && len(room.Players) > 0 {
		resp.BlueTeam, resp.RedTeam = categorizeTeamPlayers(room.Players)
	}

	resp.Actions = make([]DraftActionDTO, 0, len(actions))
	for _, action := range actions {
		resp.Actions = append(resp.Actions, DraftActionDTO{
			PhaseIndex: action.PhaseIndex,
			Team:       string(action.Team),
			ActionType: string(action.ActionType),
			ChampionID: action.ChampionID,
			ActionTime: action.ActionTime.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Helper functions

func determineSide(userID uuid.UUID, room *domain.Room) string {
	// Check if user is in team players
	for _, p := range room.Players {
		if p.UserID == userID {
			return string(p.Team)
		}
	}

	// Check 1v1 sides
	if room.BlueSideUserID != nil && *room.BlueSideUserID == userID {
		return "blue"
	}
	if room.RedSideUserID != nil && *room.RedSideUserID == userID {
		return "red"
	}
	if room.CreatedBy == userID {
		return "spectator"
	}
	return "spectator"
}

func userHasAccessToRoom(userID uuid.UUID, room *domain.Room) bool {
	// Creator always has access
	if room.CreatedBy == userID {
		return true
	}
	// Check 1v1 participants
	if room.BlueSideUserID != nil && *room.BlueSideUserID == userID {
		return true
	}
	if room.RedSideUserID != nil && *room.RedSideUserID == userID {
		return true
	}
	// Check team players
	for _, p := range room.Players {
		if p.UserID == userID {
			return true
		}
	}
	return false
}

func categorizeTeamPlayers(players []domain.RoomPlayer) ([]MatchPlayerDTO, []MatchPlayerDTO) {
	var blueTeam, redTeam []MatchPlayerDTO

	for _, p := range players {
		dto := MatchPlayerDTO{
			UserID:       p.UserID.String(),
			DisplayName:  p.DisplayName,
			AssignedRole: string(p.AssignedRole),
			IsCaptain:    p.IsCaptain,
		}
		if p.Team == domain.SideBlue {
			blueTeam = append(blueTeam, dto)
		} else if p.Team == domain.SideRed {
			redTeam = append(redTeam, dto)
		}
	}

	return blueTeam, redTeam
}

func jsonToStringSlice(data []byte) []string {
	if data == nil {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return []string{}
	}
	return result
}
