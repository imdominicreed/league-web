package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type roomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *roomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(ctx context.Context, room *domain.Room) error {
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *roomRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	var room domain.Room
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("BlueSideUser").
		Preload("RedSideUser").
		First(&room, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) GetByShortCode(ctx context.Context, code string) (*domain.Room, error) {
	var room domain.Room
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("BlueSideUser").
		Preload("RedSideUser").
		First(&room, "short_code = ?", code).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) Update(ctx context.Context, room *domain.Room) error {
	return r.db.WithContext(ctx).Save(room).Error
}

func (r *roomRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error) {
	var rooms []*domain.Room
	err := r.db.WithContext(ctx).
		Where("created_by = ? OR blue_side_user_id = ? OR red_side_user_id = ?", userID, userID, userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rooms).Error
	if err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *roomRepository) GetCompletedByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error) {
	var rooms []*domain.Room
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("BlueSideUser").
		Preload("RedSideUser").
		Preload("Players").
		Where("status = ?", domain.RoomStatusCompleted).
		Where("created_by = ? OR blue_side_user_id = ? OR red_side_user_id = ?", userID, userID, userID).
		Order("completed_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rooms).Error
	if err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *roomRepository) GetByIDWithDraftState(ctx context.Context, id uuid.UUID) (*domain.Room, *domain.DraftState, error) {
	var room domain.Room
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("BlueSideUser").
		Preload("RedSideUser").
		Preload("Players").
		First(&room, "id = ?", id).Error
	if err != nil {
		return nil, nil, err
	}

	var draftState domain.DraftState
	err = r.db.WithContext(ctx).
		First(&draftState, "room_id = ?", id).Error
	if err != nil {
		// Draft state might not exist yet
		return &room, nil, nil
	}

	return &room, &draftState, nil
}
