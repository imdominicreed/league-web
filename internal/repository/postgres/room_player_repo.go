package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type roomPlayerRepository struct {
	db *gorm.DB
}

func NewRoomPlayerRepository(db *gorm.DB) *roomPlayerRepository {
	return &roomPlayerRepository{db: db}
}

func (r *roomPlayerRepository) Create(ctx context.Context, player *domain.RoomPlayer) error {
	return r.db.WithContext(ctx).Create(player).Error
}

func (r *roomPlayerRepository) CreateMany(ctx context.Context, players []*domain.RoomPlayer) error {
	if len(players) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&players).Error
}

func (r *roomPlayerRepository) GetByRoomID(ctx context.Context, roomId uuid.UUID) ([]*domain.RoomPlayer, error) {
	var players []*domain.RoomPlayer
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomId).
		Order("joined_at").
		Find(&players).Error
	if err != nil {
		return nil, err
	}
	return players, nil
}

func (r *roomPlayerRepository) GetByRoomAndUser(ctx context.Context, roomId, userId uuid.UUID) (*domain.RoomPlayer, error) {
	var player domain.RoomPlayer
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomId, userId).
		First(&player).Error
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func (r *roomPlayerRepository) GetCaptains(ctx context.Context, roomId uuid.UUID) (map[string]*domain.RoomPlayer, error) {
	var players []*domain.RoomPlayer
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND is_captain = ?", roomId, true).
		Find(&players).Error
	if err != nil {
		return nil, err
	}

	captains := make(map[string]*domain.RoomPlayer)
	for _, player := range players {
		captains[string(player.Team)] = player // Side is already a string type alias
	}
	return captains, nil
}
