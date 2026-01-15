package service

import (
	"github.com/dom/league-draft-website/internal/config"
	"github.com/dom/league-draft-website/internal/repository"
)

type Services struct {
	Auth        *AuthService
	Room        *RoomService
	Champion    *ChampionService
	Draft       *DraftService
	Profile     *ProfileService
	Lobby       *LobbyService
	Matchmaking *MatchmakingService
}

func NewServices(repos *repository.Repositories, cfg *config.Config) *Services {
	roomService := NewRoomService(repos.Room, repos.DraftState)
	matchmakingService := NewMatchmakingService(
		repos.UserRoleProfile,
		repos.MatchOption,
		repos.Lobby,
	)

	return &Services{
		Auth:     NewAuthService(repos.User, repos.Session, cfg),
		Room:     roomService,
		Champion: NewChampionService(repos.Champion, cfg),
		Draft:    NewDraftService(repos.DraftState, repos.DraftAction, repos.FearlessBan),
		Profile:  NewProfileService(repos.User, repos.UserRoleProfile),
		Lobby: NewLobbyService(
			repos.Lobby,
			repos.LobbyPlayer,
			repos.MatchOption,
			repos.UserRoleProfile,
			repos.RoomPlayer,
			repos.PendingAction,
			repos.Vote,
			roomService,
			matchmakingService,
		),
		Matchmaking: matchmakingService,
	}
}
