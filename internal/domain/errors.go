package domain

import "errors"

// Profile validation errors
var (
	ErrInvalidRole          = errors.New("invalid role")
	ErrInvalidRank          = errors.New("invalid rank")
	ErrInvalidComfortRating = errors.New("comfort rating must be between 1 and 5")
	ErrInvalidMMR           = errors.New("MMR must be non-negative")
)

// Lobby errors
var (
	ErrLobbyFull        = errors.New("lobby is full")
	ErrLobbyNotFound    = errors.New("lobby not found")
	ErrAlreadyInLobby   = errors.New("user is already in lobby")
	ErrNotInLobby       = errors.New("user is not in lobby")
	ErrNotLobbyCreator  = errors.New("only lobby creator can perform this action")
	ErrNotEnoughPlayers = errors.New("not enough players in lobby")
	ErrPlayersNotReady  = errors.New("not all players are ready")
	ErrInvalidLobbyState = errors.New("invalid lobby state for this action")
)
