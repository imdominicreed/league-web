package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dom/league-draft-website/internal/api"
	"github.com/dom/league-draft-website/internal/config"
	"github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/websocket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize database
	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Initialize repositories
	repos := postgres.NewRepositories(db)

	// Initialize WebSocket hub
	hub := websocket.NewHub(repos.User, repos.RoomPlayer)
	go hub.Run()

	// Initialize services
	services := service.NewServices(repos, cfg)

	// Initialize router
	router := api.NewRouter(services, hub, cfg)

	// Create server
	srv := &http.Server{
		Addr:         "0.0.0.0:" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
