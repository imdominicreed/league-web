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
