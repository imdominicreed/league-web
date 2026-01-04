package service_test

import (
	"context"
	"testing"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoomService_CreateRoom(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	roomService := service.NewRoomService(repos.Room, repos.DraftState)
	ctx := context.Background()

	// Create a user
	user, _ := testutil.NewUserBuilder().Build(t, testDB.DB)

	tests := []struct {
		name    string
		input   service.CreateRoomInput
		wantErr bool
	}{
		{
			name: "successful creation with pro play mode",
			input: service.CreateRoomInput{
				CreatedBy:     user.ID,
				DraftMode:     domain.DraftModeProPlay,
				TimerDuration: 30,
			},
			wantErr: false,
		},
		{
			name: "successful creation with fearless mode",
			input: service.CreateRoomInput{
				CreatedBy:     user.ID,
				DraftMode:     domain.DraftModeFearless,
				TimerDuration: 45,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room, err := roomService.CreateRoom(ctx, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, room)
			assert.Equal(t, tt.input.CreatedBy, room.CreatedBy)
			assert.Equal(t, tt.input.DraftMode, room.DraftMode)
			assert.Equal(t, tt.input.TimerDuration, room.TimerDurationSeconds)
			assert.NotEmpty(t, room.ShortCode)
			assert.Len(t, room.ShortCode, 6) // Short codes are 6 hex chars
			assert.Equal(t, domain.RoomStatusWaiting, room.Status)
		})
	}
}

func TestRoomService_GetRoom(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	roomService := service.NewRoomService(repos.Room, repos.DraftState)
	ctx := context.Background()

	// Create a user and room
	user, _ := testutil.NewUserBuilder().Build(t, testDB.DB)
	room, err := roomService.CreateRoom(ctx, service.CreateRoomInput{
		CreatedBy:     user.ID,
		DraftMode:     domain.DraftModeProPlay,
		TimerDuration: 30,
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		idOrCode string
		wantRoom bool
		wantErr  bool
	}{
		{
			name:     "get by UUID",
			idOrCode: room.ID.String(),
			wantRoom: true,
			wantErr:  false,
		},
		{
			name:     "get by short code",
			idOrCode: room.ShortCode,
			wantRoom: true,
			wantErr:  false,
		},
		{
			name:     "get by lowercase short code",
			idOrCode: string([]rune(room.ShortCode)), // Should be case-insensitive
			wantRoom: true,
			wantErr:  false,
		},
		{
			name:     "non-existent UUID",
			idOrCode: uuid.New().String(),
			wantRoom: false,
			wantErr:  true,
		},
		{
			name:     "non-existent short code",
			idOrCode: "INVALID",
			wantRoom: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := roomService.GetRoom(ctx, tt.idOrCode)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.wantRoom {
				assert.Equal(t, room.ID, got.ID)
			}
		})
	}
}

func TestRoomService_JoinRoom(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	roomService := service.NewRoomService(repos.Room, repos.DraftState)
	ctx := context.Background()

	// Create users
	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	user1, _ := testutil.NewUserBuilder().WithDisplayName("user1").Build(t, testDB.DB)
	user2, _ := testutil.NewUserBuilder().WithDisplayName("user2").Build(t, testDB.DB)

	tests := []struct {
		name        string
		setupRoom   func() *domain.Room
		userID      uuid.UUID
		side        domain.Side
		wantErr     error
		checkResult func(*testing.T, *domain.Room)
	}{
		{
			name: "join blue side",
			setupRoom: func() *domain.Room {
				room, _ := roomService.CreateRoom(ctx, service.CreateRoomInput{
					CreatedBy:     creator.ID,
					DraftMode:     domain.DraftModeProPlay,
					TimerDuration: 30,
				})
				return room
			},
			userID: user1.ID,
			side:   domain.SideBlue,
			checkResult: func(t *testing.T, room *domain.Room) {
				assert.NotNil(t, room.BlueSideUserID)
				assert.Equal(t, user1.ID, *room.BlueSideUserID)
			},
		},
		{
			name: "join red side",
			setupRoom: func() *domain.Room {
				room, _ := roomService.CreateRoom(ctx, service.CreateRoomInput{
					CreatedBy:     creator.ID,
					DraftMode:     domain.DraftModeProPlay,
					TimerDuration: 30,
				})
				return room
			},
			userID: user1.ID,
			side:   domain.SideRed,
			checkResult: func(t *testing.T, room *domain.Room) {
				assert.NotNil(t, room.RedSideUserID)
				assert.Equal(t, user1.ID, *room.RedSideUserID)
			},
		},
		{
			name: "blue side already taken by different user",
			setupRoom: func() *domain.Room {
				room, _ := roomService.CreateRoom(ctx, service.CreateRoomInput{
					CreatedBy:     creator.ID,
					DraftMode:     domain.DraftModeProPlay,
					TimerDuration: 30,
				})
				// User1 takes blue side
				roomService.JoinRoom(ctx, room.ID, user1.ID, domain.SideBlue)
				return room
			},
			userID:  user2.ID,
			side:    domain.SideBlue,
			wantErr: service.ErrSideTaken,
		},
		{
			name: "same user can rejoin their side",
			setupRoom: func() *domain.Room {
				room, _ := roomService.CreateRoom(ctx, service.CreateRoomInput{
					CreatedBy:     creator.ID,
					DraftMode:     domain.DraftModeProPlay,
					TimerDuration: 30,
				})
				// User1 takes blue side
				roomService.JoinRoom(ctx, room.ID, user1.ID, domain.SideBlue)
				return room
			},
			userID: user1.ID,
			side:   domain.SideBlue,
			checkResult: func(t *testing.T, room *domain.Room) {
				assert.NotNil(t, room.BlueSideUserID)
				assert.Equal(t, user1.ID, *room.BlueSideUserID)
			},
		},
		{
			name: "non-existent room",
			setupRoom: func() *domain.Room {
				return &domain.Room{ID: uuid.New()}
			},
			userID:  user1.ID,
			side:    domain.SideBlue,
			wantErr: service.ErrRoomNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := tt.setupRoom()

			result, err := roomService.JoinRoom(ctx, room.ID, tt.userID, tt.side)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestRoomService_GetUserRooms(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	roomService := service.NewRoomService(repos.Room, repos.DraftState)
	ctx := context.Background()

	// Create users
	user1, _ := testutil.NewUserBuilder().WithDisplayName("user1").Build(t, testDB.DB)
	user2, _ := testutil.NewUserBuilder().WithDisplayName("user2").Build(t, testDB.DB)

	// Create rooms for user1
	for i := 0; i < 5; i++ {
		roomService.CreateRoom(ctx, service.CreateRoomInput{
			CreatedBy:     user1.ID,
			DraftMode:     domain.DraftModeProPlay,
			TimerDuration: 30,
		})
	}

	// Create room for user2
	roomService.CreateRoom(ctx, service.CreateRoomInput{
		CreatedBy:     user2.ID,
		DraftMode:     domain.DraftModeProPlay,
		TimerDuration: 30,
	})

	tests := []struct {
		name      string
		userID    uuid.UUID
		limit     int
		offset    int
		wantCount int
	}{
		{
			name:      "get all rooms for user1",
			userID:    user1.ID,
			limit:     10,
			offset:    0,
			wantCount: 5,
		},
		{
			name:      "get limited rooms for user1",
			userID:    user1.ID,
			limit:     3,
			offset:    0,
			wantCount: 3,
		},
		{
			name:      "get rooms with offset",
			userID:    user1.ID,
			limit:     10,
			offset:    3,
			wantCount: 2,
		},
		{
			name:      "get rooms for user2",
			userID:    user2.ID,
			limit:     10,
			offset:    0,
			wantCount: 1,
		},
		{
			name:      "get rooms for non-existent user",
			userID:    uuid.New(),
			limit:     10,
			offset:    0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rooms, err := roomService.GetUserRooms(ctx, tt.userID, tt.limit, tt.offset)
			require.NoError(t, err)
			assert.Len(t, rooms, tt.wantCount)
		})
	}
}

func TestRoomService_ShortCodeGeneration(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	roomService := service.NewRoomService(repos.Room, repos.DraftState)
	ctx := context.Background()

	user, _ := testutil.NewUserBuilder().Build(t, testDB.DB)

	// Create multiple rooms and verify unique short codes
	shortCodes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		room, err := roomService.CreateRoom(ctx, service.CreateRoomInput{
			CreatedBy:     user.ID,
			DraftMode:     domain.DraftModeProPlay,
			TimerDuration: 30,
		})
		require.NoError(t, err)

		assert.Len(t, room.ShortCode, 6)
		assert.False(t, shortCodes[room.ShortCode], "duplicate short code generated")
		shortCodes[room.ShortCode] = true
	}
}
