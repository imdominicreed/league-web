package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dom/league-draft-website/internal/api/middleware"
	"github.com/dom/league-draft-website/internal/config"
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type SimulationHandler struct {
	roomRepo        repository.RoomRepository
	draftStateRepo  repository.DraftStateRepository
	draftActionRepo repository.DraftActionRepository
	roomPlayerRepo  repository.RoomPlayerRepository
	cfg             *config.Config
}

func NewSimulationHandler(
	roomRepo repository.RoomRepository,
	draftStateRepo repository.DraftStateRepository,
	draftActionRepo repository.DraftActionRepository,
	roomPlayerRepo repository.RoomPlayerRepository,
	cfg *config.Config,
) *SimulationHandler {
	return &SimulationHandler{
		roomRepo:        roomRepo,
		draftStateRepo:  draftStateRepo,
		draftActionRepo: draftActionRepo,
		roomPlayerRepo:  roomPlayerRepo,
		cfg:             cfg,
	}
}

type SimulateMatchRequest struct {
	RoomID      string              `json:"roomId"`
	IsTeamDraft bool                `json:"isTeamDraft"`
	BluePicks   []string            `json:"bluePicks"`
	RedPicks    []string            `json:"redPicks"`
	BlueBans    []string            `json:"blueBans"`
	RedBans     []string            `json:"redBans"`
	DaysAgo     int                 `json:"daysAgo"`

	// For 1v1 matches
	BlueSideUserID string `json:"blueSideUserId,omitempty"`
	RedSideUserID  string `json:"redSideUserId,omitempty"`

	// For team drafts
	BlueTeam []TeamPlayerRequest `json:"blueTeam,omitempty"`
	RedTeam  []TeamPlayerRequest `json:"redTeam,omitempty"`
}

type TeamPlayerRequest struct {
	UserID       string `json:"userId"`
	DisplayName  string `json:"displayName"`
	AssignedRole string `json:"assignedRole"`
	IsCaptain    bool   `json:"isCaptain"`
}

// SimulateMatch creates a completed match for testing/demo purposes
func (h *SimulationHandler) SimulateMatch(w http.ResponseWriter, r *http.Request) {
	// Only allow in development
	if h.cfg.Environment == "production" {
		http.Error(w, "Not available in production", http.StatusForbidden)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SimulateMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse room ID
	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	// Get the room
	room, err := h.roomRepo.GetByID(r.Context(), roomID)
	if err != nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	// Calculate timestamps
	completedAt := time.Now().Add(-time.Duration(req.DaysAgo) * 24 * time.Hour)
	startedAt := completedAt.Add(-15 * time.Minute) // Draft took ~15 minutes

	// Update room to completed status
	room.Status = domain.RoomStatusCompleted
	room.IsTeamDraft = req.IsTeamDraft
	room.StartedAt = &startedAt
	room.CompletedAt = &completedAt

	if !req.IsTeamDraft {
		// Set 1v1 sides
		if req.BlueSideUserID != "" {
			blueID, _ := uuid.Parse(req.BlueSideUserID)
			room.BlueSideUserID = &blueID
		} else {
			room.BlueSideUserID = &userID
		}
		if req.RedSideUserID != "" {
			redID, _ := uuid.Parse(req.RedSideUserID)
			room.RedSideUserID = &redID
		}
	}

	if err := h.roomRepo.Update(r.Context(), room); err != nil {
		log.Printf("ERROR [simulation] failed to update room: %v", err)
		http.Error(w, "Failed to update room", http.StatusInternalServerError)
		return
	}

	// Get existing draft state and update it
	draftState, err := h.draftStateRepo.GetByRoomID(r.Context(), roomID)
	if err != nil {
		log.Printf("ERROR [simulation] failed to get draft state: %v", err)
		http.Error(w, "Failed to get draft state", http.StatusInternalServerError)
		return
	}

	bluePicks, _ := json.Marshal(req.BluePicks)
	redPicks, _ := json.Marshal(req.RedPicks)
	blueBans, _ := json.Marshal(req.BlueBans)
	redBans, _ := json.Marshal(req.RedBans)

	draftState.CurrentPhase = 20
	draftState.BluePicks = datatypes.JSON(bluePicks)
	draftState.RedPicks = datatypes.JSON(redPicks)
	draftState.BlueBans = datatypes.JSON(blueBans)
	draftState.RedBans = datatypes.JSON(redBans)
	draftState.IsComplete = true

	if err := h.draftStateRepo.Update(r.Context(), draftState); err != nil {
		log.Printf("ERROR [simulation] failed to update draft state: %v", err)
		http.Error(w, "Failed to update draft state", http.StatusInternalServerError)
		return
	}

	// Create draft actions for timeline
	actions := buildDraftActions(roomID, req.BluePicks, req.RedPicks, req.BlueBans, req.RedBans, startedAt)
	for _, action := range actions {
		if err := h.draftActionRepo.Create(r.Context(), action); err != nil {
			log.Printf("WARN [simulation] failed to create draft action: %v", err)
		}
	}

	// Create room players for team drafts
	if req.IsTeamDraft && len(req.BlueTeam) > 0 && len(req.RedTeam) > 0 {
		for _, player := range req.BlueTeam {
			playerUserID, _ := uuid.Parse(player.UserID)
			roomPlayer := &domain.RoomPlayer{
				RoomID:       roomID,
				UserID:       playerUserID,
				Team:         domain.SideBlue,
				AssignedRole: domain.Role(player.AssignedRole),
				DisplayName:  player.DisplayName,
				IsCaptain:    player.IsCaptain,
				IsReady:      true,
			}
			if err := h.roomPlayerRepo.Create(r.Context(), roomPlayer); err != nil {
				log.Printf("WARN [simulation] failed to create room player: %v", err)
			}
		}

		for _, player := range req.RedTeam {
			playerUserID, _ := uuid.Parse(player.UserID)
			roomPlayer := &domain.RoomPlayer{
				RoomID:       roomID,
				UserID:       playerUserID,
				Team:         domain.SideRed,
				AssignedRole: domain.Role(player.AssignedRole),
				DisplayName:  player.DisplayName,
				IsCaptain:    player.IsCaptain,
				IsReady:      true,
			}
			if err := h.roomPlayerRepo.Create(r.Context(), roomPlayer); err != nil {
				log.Printf("WARN [simulation] failed to create room player: %v", err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"roomId":  roomID.String(),
	})
}

// buildDraftActions creates the action timeline based on pro play phase order
func buildDraftActions(roomID uuid.UUID, bluePicks, redPicks, blueBans, redBans []string, startTime time.Time) []*domain.DraftAction {
	actions := make([]*domain.DraftAction, 0, 20)
	actionTime := startTime

	// Pro play draft order (20 phases)
	// Ban Phase 1 (6 bans): B-R-B-R-B-R
	// Pick Phase 1 (6 picks): B-R-R-B-B-R
	// Ban Phase 2 (4 bans): R-B-R-B
	// Pick Phase 2 (4 picks): R-B-B-R

	blueBanIdx, redBanIdx := 0, 0
	bluePickIdx, redPickIdx := 0, 0

	for _, phase := range domain.ProPlayPhases {
		actionTime = actionTime.Add(20 * time.Second) // Each action takes ~20 seconds

		var championID string
		if phase.ActionType == domain.ActionTypeBan {
			if phase.Team == domain.SideBlue && blueBanIdx < len(blueBans) {
				championID = blueBans[blueBanIdx]
				blueBanIdx++
			} else if phase.Team == domain.SideRed && redBanIdx < len(redBans) {
				championID = redBans[redBanIdx]
				redBanIdx++
			}
		} else {
			if phase.Team == domain.SideBlue && bluePickIdx < len(bluePicks) {
				championID = bluePicks[bluePickIdx]
				bluePickIdx++
			} else if phase.Team == domain.SideRed && redPickIdx < len(redPicks) {
				championID = redPicks[redPickIdx]
				redPickIdx++
			}
		}

		if championID == "" {
			championID = "None"
		}

		actions = append(actions, &domain.DraftAction{
			RoomID:     roomID,
			PhaseIndex: phase.Index,
			Team:       phase.Team,
			ActionType: phase.ActionType,
			ChampionID: championID,
			ActionTime: actionTime,
		})
	}

	return actions
}
