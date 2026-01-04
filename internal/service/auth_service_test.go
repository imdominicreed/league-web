package service_test

import (
	"context"
	"testing"

	"github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Register(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	cfg := testutil.TestConfig()
	authService := service.NewAuthService(repos.User, repos.Session, cfg)
	ctx := context.Background()

	tests := []struct {
		name      string
		input     service.RegisterInput
		setup     func()
		wantErr   error
		checkUser bool
	}{
		{
			name: "successful registration",
			input: service.RegisterInput{
				DisplayName: "newuser",
				Password:    "password123",
			},
			checkUser: true,
		},
		{
			name: "duplicate display name",
			input: service.RegisterInput{
				DisplayName: "existinguser",
				Password:    "password123",
			},
			setup: func() {
				// Create existing user
				testutil.NewUserBuilder().
					WithDisplayName("existinguser").
					Build(t, testDB.DB)
			},
			wantErr: service.ErrDisplayNameExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up between tests
			testDB.Truncate(t)

			if tt.setup != nil {
				tt.setup()
			}

			result, err := authService.Register(ctx, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.checkUser {
				assert.NotNil(t, result.User)
				assert.Equal(t, tt.input.DisplayName, result.User.DisplayName)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	cfg := testutil.TestConfig()
	authService := service.NewAuthService(repos.User, repos.Session, cfg)
	ctx := context.Background()

	// Create a user for login tests
	user, rawPassword := testutil.NewUserBuilder().
		WithDisplayName("loginuser").
		WithPassword("correctpassword").
		Build(t, testDB.DB)

	tests := []struct {
		name    string
		input   service.LoginInput
		wantErr error
	}{
		{
			name: "successful login",
			input: service.LoginInput{
				DisplayName: user.DisplayName,
				Password:    rawPassword,
			},
		},
		{
			name: "wrong password",
			input: service.LoginInput{
				DisplayName: user.DisplayName,
				Password:    "wrongpassword",
			},
			wantErr: service.ErrInvalidCredentials,
		},
		{
			name: "non-existent user",
			input: service.LoginInput{
				DisplayName: "nonexistent",
				Password:    "anypassword",
			},
			wantErr: service.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := authService.Login(ctx, tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result.User)
			assert.Equal(t, user.ID, result.User.ID)
			assert.NotEmpty(t, result.AccessToken)
			assert.NotEmpty(t, result.RefreshToken)
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	cfg := testutil.TestConfig()
	authService := service.NewAuthService(repos.User, repos.Session, cfg)
	ctx := context.Background()

	// Register a user to get a valid token
	result, err := authService.Register(ctx, service.RegisterInput{
		DisplayName: "tokenuser",
		Password:    "password123",
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   result.AccessToken,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "notavalidjwt",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := authService.ValidateToken(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, claims)
		})
	}
}

func TestAuthService_GetUserByID(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	cfg := testutil.TestConfig()
	authService := service.NewAuthService(repos.User, repos.Session, cfg)
	ctx := context.Background()

	user, _ := testutil.NewUserBuilder().
		WithDisplayName("getuserbyid").
		Build(t, testDB.DB)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      user.ID,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := authService.GetUserByID(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, user.ID, got.ID)
			assert.Equal(t, user.DisplayName, got.DisplayName)
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repos := postgres.NewRepositories(testDB.DB)
	cfg := testutil.TestConfig()
	authService := service.NewAuthService(repos.User, repos.Session, cfg)
	ctx := context.Background()

	// Register a user to create a session
	result, err := authService.Register(ctx, service.RegisterInput{
		DisplayName: "logoutuser",
		Password:    "password123",
	})
	require.NoError(t, err)

	// Logout should succeed
	err = authService.Logout(ctx, result.User.ID)
	require.NoError(t, err)

	// Logout again should not error (no sessions to delete)
	err = authService.Logout(ctx, result.User.ID)
	require.NoError(t, err)
}
