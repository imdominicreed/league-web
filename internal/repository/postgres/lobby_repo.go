package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type lobbyRepository struct {
	db *gorm.DB
}

func NewLobbyRepository(db *gorm.DB) *lobbyRepository {
	return &lobbyRepository{db: db}
}

func (r *lobbyRepository) Create(ctx context.Context, lobby *domain.Lobby) error {
	return r.db.WithContext(ctx).Create(lobby).Error
}

func (r *lobbyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lobby, error) {
	var lobby domain.Lobby
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Players").
		Preload("Players.User").
		First(&lobby, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &lobby, nil
}

func (r *lobbyRepository) GetByShortCode(ctx context.Context, code string) (*domain.Lobby, error) {
	var lobby domain.Lobby
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Players").
		Preload("Players.User").
		First(&lobby, "short_code = ?", code).Error
	if err != nil {
		return nil, err
	}
	return &lobby, nil
}

func (r *lobbyRepository) Update(ctx context.Context, lobby *domain.Lobby) error {
	return r.db.WithContext(ctx).Save(lobby).Error
}

func (r *lobbyRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Lobby, error) {
	var lobbies []*domain.Lobby
	err := r.db.WithContext(ctx).
		Joins("JOIN lobby_players ON lobby_players.lobby_id = lobbies.id").
		Where("lobby_players.user_id = ? OR lobbies.created_by = ?", userID, userID).
		Distinct().
		Order("lobbies.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&lobbies).Error
	if err != nil {
		return nil, err
	}
	return lobbies, nil
}

func (r *lobbyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Lobby{}, "id = ?", id).Error
}
