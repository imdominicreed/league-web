package postgres

import (
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewConnection(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate tables
	err = db.AutoMigrate(
		&domain.User{},
		&domain.UserSession{},
		&domain.Room{},
		&domain.DraftState{},
		&domain.DraftAction{},
		&domain.Champion{},
		&domain.FearlessBan{},
		&domain.UserRoleProfile{},
		&domain.Lobby{},
		&domain.LobbyPlayer{},
		&domain.MatchOption{},
		&domain.MatchOptionAssignment{},
		&domain.RoomPlayer{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func NewRepositories(db *gorm.DB) *repository.Repositories {
	return &repository.Repositories{
		User:            NewUserRepository(db),
		Session:         NewSessionRepository(db),
		Room:            NewRoomRepository(db),
		DraftState:      NewDraftStateRepository(db),
		DraftAction:     NewDraftActionRepository(db),
		Champion:        NewChampionRepository(db),
		FearlessBan:     NewFearlessBanRepository(db),
		UserRoleProfile: NewUserRoleProfileRepository(db),
		Lobby:           NewLobbyRepository(db),
		LobbyPlayer:     NewLobbyPlayerRepository(db),
		MatchOption:     NewMatchOptionRepository(db),
		RoomPlayer:      NewRoomPlayerRepository(db),
	}
}
