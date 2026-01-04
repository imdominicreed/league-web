package service

import (
	"github.com/dom/league-draft-website/internal/config"
	"github.com/dom/league-draft-website/internal/repository"
)

type Services struct {
	Auth     *AuthService
	Room     *RoomService
	Champion *ChampionService
	Draft    *DraftService
}

func NewServices(repos *repository.Repositories, cfg *config.Config) *Services {
	return &Services{
		Auth:     NewAuthService(repos.User, repos.Session, cfg),
		Room:     NewRoomService(repos.Room, repos.DraftState),
		Champion: NewChampionService(repos.Champion, cfg),
		Draft:    NewDraftService(repos.DraftState, repos.DraftAction, repos.FearlessBan),
	}
}
