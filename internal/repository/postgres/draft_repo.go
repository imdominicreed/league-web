package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type draftStateRepository struct {
	db *gorm.DB
}

func NewDraftStateRepository(db *gorm.DB) *draftStateRepository {
	return &draftStateRepository{db: db}
}

func (r *draftStateRepository) Create(ctx context.Context, state *domain.DraftState) error {
	return r.db.WithContext(ctx).Create(state).Error
}

func (r *draftStateRepository) GetByRoomID(ctx context.Context, roomID uuid.UUID) (*domain.DraftState, error) {
	var state domain.DraftState
	err := r.db.WithContext(ctx).First(&state, "room_id = ?", roomID).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *draftStateRepository) Update(ctx context.Context, state *domain.DraftState) error {
	return r.db.WithContext(ctx).Save(state).Error
}

type draftActionRepository struct {
	db *gorm.DB
}

func NewDraftActionRepository(db *gorm.DB) *draftActionRepository {
	return &draftActionRepository{db: db}
}

func (r *draftActionRepository) Create(ctx context.Context, action *domain.DraftAction) error {
	return r.db.WithContext(ctx).Create(action).Error
}

func (r *draftActionRepository) GetByRoomID(ctx context.Context, roomID uuid.UUID) ([]*domain.DraftAction, error) {
	var actions []*domain.DraftAction
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("phase_index ASC").
		Find(&actions).Error
	if err != nil {
		return nil, err
	}
	return actions, nil
}
