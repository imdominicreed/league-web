package testutil

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/api"
	"github.com/dom/league-draft-website/internal/config"
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	repoPostgres "github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/websocket"
	"github.com/testcontainers/testcontainers-go"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestDB manages a testcontainers PostgreSQL instance
type TestDB struct {
	Container testcontainers.Container
	DB        *gorm.DB
	DSN       string
}

// NewTestDB creates a new PostgreSQL testcontainer and returns a connection
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	container, err := tcPostgres.Run(ctx,
		"postgres:15-alpine",
		tcPostgres.WithDatabase("test_league_draft"),
		tcPostgres.WithUsername("test"),
		tcPostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := gorm.Open(gormPostgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Run migrations
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
		t.Fatalf("failed to run migrations: %v", err)
	}

	testDB := &TestDB{
		Container: container,
		DB:        db,
		DSN:       dsn,
	}

	t.Cleanup(func() {
		testDB.Cleanup()
	})

	return testDB
}

// Cleanup terminates the container
func (tdb *TestDB) Cleanup() {
	if tdb.Container != nil {
		ctx := context.Background()
		tdb.Container.Terminate(ctx)
	}
}

// Truncate clears all tables for test isolation
func (tdb *TestDB) Truncate(t *testing.T) {
	t.Helper()

	tables := []string{
		"match_option_assignments",
		"match_options",
		"lobby_players",
		"lobbies",
		"room_players",
		"user_role_profiles",
		"fearless_bans",
		"draft_actions",
		"draft_states",
		"rooms",
		"user_sessions",
		"users",
		"champions",
	}

	for _, table := range tables {
		if err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			t.Logf("warning: failed to truncate %s: %v", table, err)
		}
	}
}

// TestConfig returns a configuration suitable for testing
func TestConfig() *config.Config {
	return &config.Config{
		Port:                 "0", // Random port
		Environment:          "test",
		JWTSecret:            "test-jwt-secret-key-for-testing-only",
		JWTExpirationHours:   1,
		DefaultTimerDuration: 2 * time.Second, // Fast timer for tests
		DataDragonVersion:    "14.1.1",
	}
}

// TestServer holds all components for integration testing
type TestServer struct {
	Server   *httptest.Server
	DB       *TestDB
	Repos    *repository.Repositories
	Services *service.Services
	Hub      *websocket.Hub
	Config   *config.Config
}

// NewTestServer creates a complete test server with all dependencies
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	testDB := NewTestDB(t)
	cfg := TestConfig()

	repos := repoPostgres.NewRepositories(testDB.DB)
	hub := websocket.NewHub(repos.User, repos.RoomPlayer)
	go hub.Run()

	services := service.NewServices(repos, cfg)
	router := api.NewRouter(services, hub, cfg)

	server := httptest.NewServer(router)

	ts := &TestServer{
		Server:   server,
		DB:       testDB,
		Repos:    repos,
		Services: services,
		Hub:      hub,
		Config:   cfg,
	}

	t.Cleanup(func() {
		server.Close()
	})

	return ts
}

// BaseURL returns the test server's base URL
func (ts *TestServer) BaseURL() string {
	return ts.Server.URL
}

// APIURL returns the full API URL for a given path
func (ts *TestServer) APIURL(path string) string {
	return fmt.Sprintf("%s/api/v1%s", ts.Server.URL, path)
}

// WebSocketURL returns the WebSocket URL with token
func (ts *TestServer) WebSocketURL(token string) string {
	wsURL := "ws" + ts.Server.URL[4:] // Replace "http" with "ws"
	return fmt.Sprintf("%s/api/v1/ws?token=%s", wsURL, token)
}
