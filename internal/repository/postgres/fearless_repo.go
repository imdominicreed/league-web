package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fearlessBanRepository struct {
	db *gorm.DB
}

func NewFearlessBanRepository(db *gorm.DB) *fearlessBanRepository {
	return &fearlessBanRepository{db: db}
}

func (r *fearlessBanRepository) Create(ctx context.Context, ban *domain.FearlessBan) error {
	return r.db.WithContext(ctx).Create(ban).Error
}

func (r *fearlessBanRepository) GetBySeriesID(ctx context.Context, seriesID uuid.UUID) ([]*domain.FearlessBan, error) {
	var bans []*domain.FearlessBan
	err := r.db.WithContext(ctx).
		Where("series_id = ?", seriesID).
		Find(&bans).Error
	if err != nil {
		return nil, err
	}
	return bans, nil
}
