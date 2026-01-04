package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewUserRepository(testDB.DB)
	ctx := context.Background()

	tests := []struct {
		name    string
		user    *domain.User
		wantErr bool
	}{
		{
			name: "successful creation",
			user: &domain.User{
				ID:           uuid.New(),
				DisplayName:  "testuser",
				PasswordHash: "hashedpassword",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "duplicate display name",
			user: &domain.User{
				ID:           uuid.New(),
				DisplayName:  "testuser", // Same as above
				PasswordHash: "hashedpassword2",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.user)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewUserRepository(testDB.DB)
	ctx := context.Background()

	// Create a test user
	user, _ := testutil.NewUserBuilder().
		WithDisplayName("getbyid_user").
		Build(t, testDB.DB)

	tests := []struct {
		name    string
		id      uuid.UUID
		want    *domain.User
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      user.ID,
			want:    user,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      uuid.New(),
			want:    nil,
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
			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.DisplayName, got.DisplayName)
		})
	}
}

func TestUserRepository_GetByDisplayName(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewUserRepository(testDB.DB)
	ctx := context.Background()

	// Create a test user
	user, _ := testutil.NewUserBuilder().
		WithDisplayName("displayname_user").
		Build(t, testDB.DB)

	tests := []struct {
		name        string
		displayName string
		want        *domain.User
		wantErr     bool
	}{
		{
			name:        "existing user",
			displayName: "displayname_user",
			want:        user,
			wantErr:     false,
		},
		{
			name:        "non-existent user",
			displayName: "nonexistent",
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByDisplayName(ctx, tt.displayName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.DisplayName, got.DisplayName)
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewUserRepository(testDB.DB)
	ctx := context.Background()

	// Create a test user
	user, _ := testutil.NewUserBuilder().
		WithDisplayName("update_user").
		Build(t, testDB.DB)

	// Update the user
	user.DisplayName = "updated_user"
	err := repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify the update
	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_user", got.DisplayName)
}
