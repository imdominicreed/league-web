package postgres_test

import (
	"context"
	"testing"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoomRepository_Create(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewRoomRepository(testDB.DB)
	ctx := context.Background()

	// Create a user first
	user, _ := testutil.NewUserBuilder().Build(t, testDB.DB)

	room := testutil.NewRoomBuilder().
		WithCreator(user).
		WithDraftMode(domain.DraftModeProPlay).
		WithTimerDuration(30).
		Build(t, testDB.DB)

	// Verify the room was created
	got, err := repo.GetByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, room.ID, got.ID)
	assert.Equal(t, room.ShortCode, got.ShortCode)
	assert.Equal(t, domain.DraftModeProPlay, got.DraftMode)
}

func TestRoomRepository_GetByID(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewRoomRepository(testDB.DB)
	ctx := context.Background()

	user, _ := testutil.NewUserBuilder().Build(t, testDB.DB)
	room := testutil.NewRoomBuilder().
		WithCreator(user).
		Build(t, testDB.DB)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing room",
			id:      room.ID,
			wantErr: false,
		},
		{
			name:    "non-existent room",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, room.ID, got.ID)
		})
	}
}

func TestRoomRepository_GetByShortCode(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewRoomRepository(testDB.DB)
	ctx := context.Background()

	user, _ := testutil.NewUserBuilder().Build(t, testDB.DB)
	room := testutil.NewRoomBuilder().
		WithCreator(user).
		Build(t, testDB.DB)

	tests := []struct {
		name      string
		shortCode string
		wantErr   bool
	}{
		{
			name:      "existing room",
			shortCode: room.ShortCode,
			wantErr:   false,
		},
		{
			name:      "non-existent room",
			shortCode: "INVALID",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByShortCode(ctx, tt.shortCode)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, room.ID, got.ID)
		})
	}
}

func TestRoomRepository_Update(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewRoomRepository(testDB.DB)
	ctx := context.Background()

	user, _ := testutil.NewUserBuilder().Build(t, testDB.DB)
	room := testutil.NewRoomBuilder().
		WithCreator(user).
		Build(t, testDB.DB)

	// Update the room status
	room.Status = domain.RoomStatusInProgress
	err := repo.Update(ctx, room)
	require.NoError(t, err)

	// Verify the update
	got, err := repo.GetByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.RoomStatusInProgress, got.Status)
}

func TestRoomRepository_GetByUserID(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewRoomRepository(testDB.DB)
	ctx := context.Background()

	// Create users
	user1, _ := testutil.NewUserBuilder().WithDisplayName("user1").Build(t, testDB.DB)
	user2, _ := testutil.NewUserBuilder().WithDisplayName("user2").Build(t, testDB.DB)

	// Create rooms where user1 is involved
	testutil.NewRoomBuilder().WithCreator(user1).Build(t, testDB.DB)
	testutil.NewRoomBuilder().WithCreator(user2).WithBlueSide(user1).Build(t, testDB.DB)
	testutil.NewRoomBuilder().WithCreator(user2).WithRedSide(user1).Build(t, testDB.DB)

	// Create a room where user1 is not involved
	testutil.NewRoomBuilder().WithCreator(user2).Build(t, testDB.DB)

	tests := []struct {
		name      string
		userID    uuid.UUID
		limit     int
		offset    int
		wantCount int
	}{
		{
			name:      "user with multiple rooms",
			userID:    user1.ID,
			limit:     10,
			offset:    0,
			wantCount: 3,
		},
		{
			name:      "user with rooms - limited",
			userID:    user1.ID,
			limit:     2,
			offset:    0,
			wantCount: 2,
		},
		{
			name:      "user with rooms - offset",
			userID:    user1.ID,
			limit:     10,
			offset:    2,
			wantCount: 1,
		},
		{
			name:      "user with no rooms",
			userID:    uuid.New(),
			limit:     10,
			offset:    0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rooms, err := repo.GetByUserID(ctx, tt.userID, tt.limit, tt.offset)
			require.NoError(t, err)
			assert.Len(t, rooms, tt.wantCount)
		})
	}
}

func TestRoomRepository_WithSideUsers(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewRoomRepository(testDB.DB)
	ctx := context.Background()

	// Create users
	creator, _ := testutil.NewUserBuilder().WithDisplayName("creator").Build(t, testDB.DB)
	blueUser, _ := testutil.NewUserBuilder().WithDisplayName("blueuser").Build(t, testDB.DB)
	redUser, _ := testutil.NewUserBuilder().WithDisplayName("reduser").Build(t, testDB.DB)

	// Create room with both sides
	room := testutil.NewRoomBuilder().
		WithCreator(creator).
		WithBlueSide(blueUser).
		WithRedSide(redUser).
		Build(t, testDB.DB)

	// Verify the room has correct side users
	got, err := repo.GetByID(ctx, room.ID)
	require.NoError(t, err)

	assert.NotNil(t, got.BlueSideUserID)
	assert.Equal(t, blueUser.ID, *got.BlueSideUserID)

	assert.NotNil(t, got.RedSideUserID)
	assert.Equal(t, redUser.ID, *got.RedSideUserID)
}
