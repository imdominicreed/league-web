package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type lobbyPlayerRepository struct {
	db *gorm.DB
}

func NewLobbyPlayerRepository(db *gorm.DB) *lobbyPlayerRepository {
	return &lobbyPlayerRepository{db: db}
}

func (r *lobbyPlayerRepository) Create(ctx context.Context, player *domain.LobbyPlayer) error {
	return r.db.WithContext(ctx).Create(player).Error
}

func (r *lobbyPlayerRepository) GetByLobbyID(ctx context.Context, lobbyID uuid.UUID) ([]*domain.LobbyPlayer, error) {
	var players []*domain.LobbyPlayer
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("lobby_id = ?", lobbyID).
		Order("joined_at").
		Find(&players).Error
	if err != nil {
		return nil, err
	}
	return players, nil
}

func (r *lobbyPlayerRepository) GetByLobbyIDAndUserID(ctx context.Context, lobbyID, userID uuid.UUID) (*domain.LobbyPlayer, error) {
	var player domain.LobbyPlayer
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("lobby_id = ? AND user_id = ?", lobbyID, userID).
		First(&player).Error
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func (r *lobbyPlayerRepository) Update(ctx context.Context, player *domain.LobbyPlayer) error {
	return r.db.WithContext(ctx).Save(player).Error
}

func (r *lobbyPlayerRepository) Delete(ctx context.Context, lobbyID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("lobby_id = ? AND user_id = ?", lobbyID, userID).
		Delete(&domain.LobbyPlayer{}).Error
}

func (r *lobbyPlayerRepository) CountByLobbyID(ctx context.Context, lobbyID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.LobbyPlayer{}).
		Where("lobby_id = ?", lobbyID).
		Count(&count).Error
	return count, err
}

func (r *lobbyPlayerRepository) UpdateTeamAssignments(ctx context.Context, lobbyID uuid.UUID, assignments map[uuid.UUID]struct {
	Team domain.Side
	Role domain.Role
}) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for userID, assignment := range assignments {
			team := assignment.Team
			role := assignment.Role
			err := tx.Model(&domain.LobbyPlayer{}).
				Where("lobby_id = ? AND user_id = ?", lobbyID, userID).
				Updates(map[string]interface{}{
					"team":          team,
					"assigned_role": role,
				}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}
