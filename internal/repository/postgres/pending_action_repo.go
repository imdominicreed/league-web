package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type pendingActionRepository struct {
	db *gorm.DB
}

func NewPendingActionRepository(db *gorm.DB) *pendingActionRepository {
	return &pendingActionRepository{db: db}
}

func (r *pendingActionRepository) Create(ctx context.Context, action *domain.PendingAction) error {
	return r.db.WithContext(ctx).Create(action).Error
}

func (r *pendingActionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PendingAction, error) {
	var action domain.PendingAction
	err := r.db.WithContext(ctx).
		Preload("Player1").
		Preload("Player2").
		Where("id = ?", id).
		First(&action).Error
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (r *pendingActionRepository) GetPendingByLobbyID(ctx context.Context, lobbyID uuid.UUID) (*domain.PendingAction, error) {
	var action domain.PendingAction
	err := r.db.WithContext(ctx).
		Preload("Player1").
		Preload("Player2").
		Where("lobby_id = ? AND status = ?", lobbyID, domain.PendingStatusPending).
		Order("created_at DESC").
		First(&action).Error
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (r *pendingActionRepository) Update(ctx context.Context, action *domain.PendingAction) error {
	return r.db.WithContext(ctx).Save(action).Error
}

func (r *pendingActionRepository) CancelAllPending(ctx context.Context, lobbyID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.PendingAction{}).
		Where("lobby_id = ? AND status = ?", lobbyID, domain.PendingStatusPending).
		Update("status", domain.PendingStatusCancelled).Error
}

// GetPendingForUser returns all pending actions for lobbies where the user is a captain
// and the user's side has not yet approved the action
func (r *pendingActionRepository) GetPendingForUser(ctx context.Context, userID uuid.UUID) ([]*domain.PendingAction, error) {
	var actions []*domain.PendingAction

	// First, get the lobby IDs where the user is a captain along with their team
	type captainInfo struct {
		LobbyID uuid.UUID
		Team    string
	}
	var captainLobbies []captainInfo

	err := r.db.WithContext(ctx).
		Table("lobby_players").
		Select("lobby_id, team").
		Where("user_id = ? AND is_captain = true AND team IS NOT NULL", userID).
		Scan(&captainLobbies).Error

	if err != nil {
		return nil, err
	}

	if len(captainLobbies) == 0 {
		return actions, nil
	}

	// Build conditions for each lobby based on the user's team
	for _, cl := range captainLobbies {
		var lobbyActions []*domain.PendingAction

		query := r.db.WithContext(ctx).
			Model(&domain.PendingAction{}).
			Preload("Player1").
			Preload("Player2").
			Preload("Lobby").
			Where("lobby_id = ?", cl.LobbyID).
			Where("status = ?", domain.PendingStatusPending).
			Where("expires_at > NOW()")

		// Show actions where either:
		// 1. The user's side hasn't approved yet, OR
		// 2. The user is the proposer (so they can see their own pending action)
		if cl.Team == "blue" {
			query = query.Where("approved_by_blue = false OR proposed_by_user = ?", userID)
		} else if cl.Team == "red" {
			query = query.Where("approved_by_red = false OR proposed_by_user = ?", userID)
		}

		if err := query.Find(&lobbyActions).Error; err != nil {
			return nil, err
		}

		actions = append(actions, lobbyActions...)
	}

	return actions, nil
}
