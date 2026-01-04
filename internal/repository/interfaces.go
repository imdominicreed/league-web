package repository

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByDisplayName(ctx context.Context, displayName string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.UserSession) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserSession, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type RoomRepository interface {
	Create(ctx context.Context, room *domain.Room) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error)
	GetByShortCode(ctx context.Context, code string) (*domain.Room, error)
	Update(ctx context.Context, room *domain.Room) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error)
}

type DraftStateRepository interface {
	Create(ctx context.Context, state *domain.DraftState) error
	GetByRoomID(ctx context.Context, roomID uuid.UUID) (*domain.DraftState, error)
	Update(ctx context.Context, state *domain.DraftState) error
}

type DraftActionRepository interface {
	Create(ctx context.Context, action *domain.DraftAction) error
	GetByRoomID(ctx context.Context, roomID uuid.UUID) ([]*domain.DraftAction, error)
}

type ChampionRepository interface {
	Upsert(ctx context.Context, champion *domain.Champion) error
	UpsertMany(ctx context.Context, champions []*domain.Champion) error
	GetAll(ctx context.Context) ([]*domain.Champion, error)
	GetByID(ctx context.Context, id string) (*domain.Champion, error)
}

type FearlessBanRepository interface {
	Create(ctx context.Context, ban *domain.FearlessBan) error
	GetBySeriesID(ctx context.Context, seriesID uuid.UUID) ([]*domain.FearlessBan, error)
}

type Repositories struct {
	User        UserRepository
	Session     SessionRepository
	Room        RoomRepository
	DraftState  DraftStateRepository
	DraftAction DraftActionRepository
	Champion    ChampionRepository
	FearlessBan FearlessBanRepository
}
