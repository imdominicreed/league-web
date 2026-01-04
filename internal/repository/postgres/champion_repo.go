package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type championRepository struct {
	db *gorm.DB
}

func NewChampionRepository(db *gorm.DB) *championRepository {
	return &championRepository{db: db}
}

func (r *championRepository) Upsert(ctx context.Context, champion *domain.Champion) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(champion).Error
}

func (r *championRepository) UpsertMany(ctx context.Context, champions []*domain.Champion) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(champions).Error
}

func (r *championRepository) GetAll(ctx context.Context) ([]*domain.Champion, error) {
	var champions []*domain.Champion
	err := r.db.WithContext(ctx).Order("name ASC").Find(&champions).Error
	if err != nil {
		return nil, err
	}
	return champions, nil
}

func (r *championRepository) GetByID(ctx context.Context, id string) (*domain.Champion, error) {
	var champion domain.Champion
	err := r.db.WithContext(ctx).First(&champion, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &champion, nil
}
