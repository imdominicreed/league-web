package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/dom/league-draft-website/internal/api/middleware"
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/go-chi/chi/v5"
)

type ProfileHandler struct {
	profileService *service.ProfileService
}

func NewProfileHandler(profileService *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

// RoleProfileResponse is the API response format for a role profile
type RoleProfileResponse struct {
	Role          string `json:"role"`
	LeagueRank    string `json:"leagueRank"`
	MMR           int    `json:"mmr"`
	ComfortRating int    `json:"comfortRating"`
}

// ProfileResponse is the full profile response
type ProfileResponse struct {
	User         UserResponse          `json:"user"`
	RoleProfiles []RoleProfileResponse `json:"roleProfiles"`
}

// UpdateRoleProfileRequest is the request body for updating a role profile
type UpdateRoleProfileRequest struct {
	LeagueRank    *string `json:"leagueRank"`
	MMR           *int    `json:"mmr"`
	ComfortRating *int    `json:"comfortRating"`
}

// GetProfile returns the current user's profile with all role data
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	profile, err := h.profileService.GetUserProfile(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR [profile.GetProfile] failed to get profile: %v", err)
		if errors.Is(err, service.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	roleProfiles := make([]RoleProfileResponse, 0, len(domain.AllRoles))
	for _, role := range domain.AllRoles {
		if rp, ok := profile.RoleProfiles[role]; ok {
			roleProfiles = append(roleProfiles, RoleProfileResponse{
				Role:          string(rp.Role),
				LeagueRank:    string(rp.LeagueRank),
				MMR:           rp.MMR,
				ComfortRating: rp.ComfortRating,
			})
		}
	}

	resp := ProfileResponse{
		User: UserResponse{
			ID:          profile.User.ID.String(),
			DisplayName: profile.User.DisplayName,
		},
		RoleProfiles: roleProfiles,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetRoleProfiles returns all role profiles for the current user
func (h *ProfileHandler) GetRoleProfiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	profiles, err := h.profileService.GetRoleProfiles(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR [profile.GetRoleProfiles] failed to get role profiles: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	resp := make([]RoleProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		resp = append(resp, RoleProfileResponse{
			Role:          string(p.Role),
			LeagueRank:    string(p.LeagueRank),
			MMR:           p.MMR,
			ComfortRating: p.ComfortRating,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// UpdateRoleProfile updates a specific role profile
func (h *ProfileHandler) UpdateRoleProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roleStr := chi.URLParam(r, "role")
	role := domain.Role(roleStr)
	if !role.IsValid() {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	var req UpdateRoleProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR [profile.UpdateRoleProfile] failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert request to service input
	input := service.UpdateRoleProfileInput{}
	if req.LeagueRank != nil {
		rank := domain.LeagueRank(*req.LeagueRank)
		input.LeagueRank = &rank
	}
	if req.MMR != nil {
		input.MMR = req.MMR
	}
	if req.ComfortRating != nil {
		input.ComfortRating = req.ComfortRating
	}

	profile, err := h.profileService.UpdateRoleProfile(r.Context(), userID, role, input)
	if err != nil {
		log.Printf("ERROR [profile.UpdateRoleProfile] failed to update role profile: %v", err)
		if errors.Is(err, domain.ErrInvalidRank) {
			http.Error(w, "Invalid rank", http.StatusBadRequest)
			return
		}
		if errors.Is(err, domain.ErrInvalidComfortRating) {
			http.Error(w, "Comfort rating must be between 1 and 5", http.StatusBadRequest)
			return
		}
		if errors.Is(err, domain.ErrInvalidMMR) {
			http.Error(w, "MMR must be non-negative", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := RoleProfileResponse{
		Role:          string(profile.Role),
		LeagueRank:    string(profile.LeagueRank),
		MMR:           profile.MMR,
		ComfortRating: profile.ComfortRating,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// InitializeProfiles creates default profiles for all roles
func (h *ProfileHandler) InitializeProfiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.profileService.InitializeProfiles(r.Context(), userID); err != nil {
		log.Printf("ERROR [profile.InitializeProfiles] failed to initialize profiles: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the newly created profiles
	profiles, err := h.profileService.GetRoleProfiles(r.Context(), userID)
	if err != nil {
		log.Printf("ERROR [profile.InitializeProfiles] failed to get profiles after init: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]RoleProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		resp = append(resp, RoleProfileResponse{
			Role:          string(p.Role),
			LeagueRank:    string(p.LeagueRank),
			MMR:           p.MMR,
			ComfortRating: p.ComfortRating,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
