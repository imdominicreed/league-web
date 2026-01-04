package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type matchOptionRepository struct {
	db *gorm.DB
}

func NewMatchOptionRepository(db *gorm.DB) *matchOptionRepository {
	return &matchOptionRepository{db: db}
}

func (r *matchOptionRepository) Create(ctx context.Context, option *domain.MatchOption) error {
	return r.db.WithContext(ctx).Create(option).Error
}

func (r *matchOptionRepository) CreateMany(ctx context.Context, options []*domain.MatchOption) error {
	if len(options) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(options).Error
}

func (r *matchOptionRepository) GetByLobbyID(ctx context.Context, lobbyID uuid.UUID) ([]*domain.MatchOption, error) {
	var options []*domain.MatchOption
	err := r.db.WithContext(ctx).
		Preload("Assignments").
		Preload("Assignments.User").
		Where("lobby_id = ?", lobbyID).
		Order("option_number").
		Find(&options).Error
	if err != nil {
		return nil, err
	}
	return options, nil
}

func (r *matchOptionRepository) GetByLobbyIDAndNumber(ctx context.Context, lobbyID uuid.UUID, optionNumber int) (*domain.MatchOption, error) {
	var option domain.MatchOption
	err := r.db.WithContext(ctx).
		Preload("Assignments").
		Preload("Assignments.User").
		Where("lobby_id = ? AND option_number = ?", lobbyID, optionNumber).
		First(&option).Error
	if err != nil {
		return nil, err
	}
	return &option, nil
}

func (r *matchOptionRepository) DeleteByLobbyID(ctx context.Context, lobbyID uuid.UUID) error {
	// First delete assignments, then options
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get option IDs
		var optionIDs []uuid.UUID
		err := tx.Model(&domain.MatchOption{}).
			Where("lobby_id = ?", lobbyID).
			Pluck("id", &optionIDs).Error
		if err != nil {
			return err
		}

		if len(optionIDs) > 0 {
			// Delete assignments
			err = tx.Where("match_option_id IN ?", optionIDs).
				Delete(&domain.MatchOptionAssignment{}).Error
			if err != nil {
				return err
			}
		}

		// Delete options
		return tx.Where("lobby_id = ?", lobbyID).
			Delete(&domain.MatchOption{}).Error
	})
}
