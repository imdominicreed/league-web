package api

import (
	"net/http"

	"github.com/dom/league-draft-website/internal/api/handlers"
	"github.com/dom/league-draft-website/internal/api/middleware"
	"github.com/dom/league-draft-website/internal/config"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/websocket"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(services *service.Services, hub *websocket.Hub, repos *repository.Repositories, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.CORS)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(services.Auth)
	roomHandler := handlers.NewRoomHandler(services.Room, hub, repos.RoomPlayer)
	championHandler := handlers.NewChampionHandler(services.Champion)
	profileHandler := handlers.NewProfileHandler(services.Profile)
	lobbyHandler := handlers.NewLobbyHandler(services.Lobby, services.Matchmaking, hub)
	matchHistoryHandler := handlers.NewMatchHistoryHandler(repos.Room, repos.DraftState, repos.DraftAction, repos.RoomPlayer)
	simulationHandler := handlers.NewSimulationHandler(repos.Room, repos.DraftState, repos.DraftAction, repos.RoomPlayer, cfg)
	wsHandler := handlers.NewWebSocketHandler(hub, services.Auth)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)

			// Protected auth routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.Auth(services.Auth))
				r.Get("/me", authHandler.Me)
				r.Post("/logout", authHandler.Logout)
			})
		})

		// Champion routes (public for now)
		r.Route("/champions", func(r chi.Router) {
			r.Get("/", championHandler.GetAll)
			r.Get("/{id}", championHandler.Get)
			r.Post("/sync", championHandler.Sync) // Should be admin-only in production
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(services.Auth))

			// Room routes
			r.Route("/rooms", func(r chi.Router) {
				r.Post("/", roomHandler.Create)
				r.Get("/{idOrCode}", roomHandler.Get)
				r.Post("/{idOrCode}/join", roomHandler.Join)
				r.Get("/code/{code}", roomHandler.GetByCode)
			})

			// User routes
			r.Route("/users", func(r chi.Router) {
				r.Get("/me/drafts", roomHandler.GetUserRooms)
			})

			// Profile routes
			r.Route("/profile", func(r chi.Router) {
				r.Get("/", profileHandler.GetProfile)
				r.Get("/roles", profileHandler.GetRoleProfiles)
				r.Put("/roles/{role}", profileHandler.UpdateRoleProfile)
				r.Post("/roles/initialize", profileHandler.InitializeProfiles)
			})

			// Match history routes
			r.Route("/match-history", func(r chi.Router) {
				r.Get("/", matchHistoryHandler.List)
				r.Get("/{roomId}", matchHistoryHandler.GetDetail)
			})

			// Simulation endpoint (development only)
			r.Post("/simulate-match", simulationHandler.SimulateMatch)

			// Lobby routes
			r.Route("/lobbies", func(r chi.Router) {
				r.Post("/", lobbyHandler.Create)
				r.Get("/{idOrCode}", lobbyHandler.Get)
				r.Post("/{idOrCode}/join", lobbyHandler.Join)
				r.Post("/{idOrCode}/leave", lobbyHandler.Leave)
				r.Post("/{idOrCode}/ready", lobbyHandler.SetReady)
				r.Post("/{id}/generate-teams", lobbyHandler.GenerateTeams)
				r.Get("/{id}/match-options", lobbyHandler.GetMatchOptions)
				r.Post("/{id}/select-option", lobbyHandler.SelectOption)
				r.Post("/{id}/start-draft", lobbyHandler.StartDraft)

				// Captain management
				r.Post("/{id}/take-captain", lobbyHandler.TakeCaptain)
				r.Post("/{id}/promote-captain", lobbyHandler.PromoteCaptain)
				r.Post("/{id}/kick", lobbyHandler.KickPlayer)

				// Pending actions
				r.Post("/{id}/swap", lobbyHandler.ProposeSwap)
				r.Post("/{id}/propose-matchmake", lobbyHandler.ProposeMatchmake)
				r.Post("/{id}/propose-select-option", lobbyHandler.ProposeSelectOption)
				r.Post("/{id}/propose-start-draft", lobbyHandler.ProposeStartDraft)
				r.Get("/{id}/pending-action", lobbyHandler.GetPendingAction)
				r.Post("/{id}/pending-action/{actionId}/approve", lobbyHandler.ApprovePendingAction)
				r.Post("/{id}/pending-action/{actionId}/cancel", lobbyHandler.CancelPendingAction)

				// Team stats
				r.Get("/{id}/team-stats", lobbyHandler.GetTeamStats)

				// Voting
				r.Post("/{id}/vote", lobbyHandler.Vote)
				r.Get("/{id}/voting-status", lobbyHandler.GetVotingStatus)
				r.Post("/{id}/start-voting", lobbyHandler.StartVoting)
				r.Post("/{id}/end-voting", lobbyHandler.EndVoting)
			})
		})

		// WebSocket endpoint
		r.Get("/ws", wsHandler.Handle)
	})

	return r
}
