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
	GetCompletedByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error)
	GetAllCompleted(ctx context.Context, limit, offset int) ([]*domain.Room, error)
	GetByIDWithDraftState(ctx context.Context, id uuid.UUID) (*domain.Room, *domain.DraftState, error)
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

type UserRoleProfileRepository interface {
	Create(ctx context.Context, profile *domain.UserRoleProfile) error
	CreateMany(ctx context.Context, profiles []*domain.UserRoleProfile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.UserRoleProfile, error)
	GetByUserIDAndRole(ctx context.Context, userID uuid.UUID, role domain.Role) (*domain.UserRoleProfile, error)
	Upsert(ctx context.Context, profile *domain.UserRoleProfile) error
	Update(ctx context.Context, profile *domain.UserRoleProfile) error
	GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]*domain.UserRoleProfile, error)
}

type LobbyRepository interface {
	Create(ctx context.Context, lobby *domain.Lobby) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Lobby, error)
	GetByShortCode(ctx context.Context, code string) (*domain.Lobby, error)
	Update(ctx context.Context, lobby *domain.Lobby) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Lobby, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type LobbyPlayerRepository interface {
	Create(ctx context.Context, player *domain.LobbyPlayer) error
	GetByLobbyID(ctx context.Context, lobbyID uuid.UUID) ([]*domain.LobbyPlayer, error)
	GetByLobbyIDAndUserID(ctx context.Context, lobbyID, userID uuid.UUID) (*domain.LobbyPlayer, error)
	Update(ctx context.Context, player *domain.LobbyPlayer) error
	Delete(ctx context.Context, lobbyID, userID uuid.UUID) error
	CountByLobbyID(ctx context.Context, lobbyID uuid.UUID) (int64, error)
	UpdateTeamAssignments(ctx context.Context, lobbyID uuid.UUID, assignments map[uuid.UUID]struct {
		Team domain.Side
		Role domain.Role
	}) error
}

type MatchOptionRepository interface {
	Create(ctx context.Context, option *domain.MatchOption) error
	CreateMany(ctx context.Context, options []*domain.MatchOption) error
	GetByLobbyID(ctx context.Context, lobbyID uuid.UUID) ([]*domain.MatchOption, error)
	GetByLobbyIDAndNumber(ctx context.Context, lobbyID uuid.UUID, optionNumber int) (*domain.MatchOption, error)
	DeleteByLobbyID(ctx context.Context, lobbyID uuid.UUID) error
}

type RoomPlayerRepository interface {
	Create(ctx context.Context, player *domain.RoomPlayer) error
	CreateMany(ctx context.Context, players []*domain.RoomPlayer) error
	GetByRoomID(ctx context.Context, roomId uuid.UUID) ([]*domain.RoomPlayer, error)
	GetByRoomAndUser(ctx context.Context, roomId, userId uuid.UUID) (*domain.RoomPlayer, error)
	GetCaptains(ctx context.Context, roomId uuid.UUID) (map[string]*domain.RoomPlayer, error)
}

type PendingActionRepository interface {
	Create(ctx context.Context, action *domain.PendingAction) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PendingAction, error)
	GetPendingByLobbyID(ctx context.Context, lobbyID uuid.UUID) (*domain.PendingAction, error)
	GetPendingForUser(ctx context.Context, userID uuid.UUID) ([]*domain.PendingAction, error)
	Update(ctx context.Context, action *domain.PendingAction) error
	CancelAllPending(ctx context.Context, lobbyID uuid.UUID) error
}

type VoteRepository interface {
	Create(ctx context.Context, vote *domain.Vote) error
	Update(ctx context.Context, vote *domain.Vote) error
	GetByLobbyAndUser(ctx context.Context, lobbyID, userID uuid.UUID) (*domain.Vote, error)
	GetVotesByLobby(ctx context.Context, lobbyID uuid.UUID) ([]*domain.Vote, error)
	GetVoteCounts(ctx context.Context, lobbyID uuid.UUID) (map[int]int, error)
	DeleteByLobby(ctx context.Context, lobbyID uuid.UUID) error
	DeleteByLobbyAndUser(ctx context.Context, lobbyID, userID uuid.UUID) error
}

type Repositories struct {
	User            UserRepository
	Session         SessionRepository
	Room            RoomRepository
	DraftState      DraftStateRepository
	DraftAction     DraftActionRepository
	Champion        ChampionRepository
	FearlessBan     FearlessBanRepository
	UserRoleProfile UserRoleProfileRepository
	Lobby           LobbyRepository
	LobbyPlayer     LobbyPlayerRepository
	MatchOption     MatchOptionRepository
	RoomPlayer      RoomPlayerRepository
	PendingAction   PendingActionRepository
	Vote            VoteRepository
}
