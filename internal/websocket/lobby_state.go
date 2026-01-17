package websocket

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// LobbyState holds the in-memory state for a single lobby
type LobbyState struct {
	lobbyID uuid.UUID

	mu            sync.RWMutex
	clients       map[*LobbyClient]bool
	votes         map[uuid.UUID]*InMemoryVote       // userID -> vote
	pendingAction *InMemoryPendingAction
}

// NewLobbyState creates a new lobby state
func NewLobbyState(lobbyID uuid.UUID) *LobbyState {
	return &LobbyState{
		lobbyID: lobbyID,
		clients: make(map[*LobbyClient]bool),
		votes:   make(map[uuid.UUID]*InMemoryVote),
	}
}

// AddClient adds a client to the lobby
func (s *LobbyState) AddClient(client *LobbyClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client] = true
}

// RemoveClient removes a client from the lobby
func (s *LobbyState) RemoveClient(client *LobbyClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client)
}

// ClientCount returns the number of connected clients
func (s *LobbyState) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// Broadcast sends a message to all clients in the lobby
func (s *LobbyState) Broadcast(msg *LobbyMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		client.Send(msg)
	}
}

// BroadcastExcept sends a message to all clients except the specified one
func (s *LobbyState) BroadcastExcept(msg *LobbyMessage, except *LobbyClient) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client != except {
			client.Send(msg)
		}
	}
}

// SendToUser sends a message to a specific user
func (s *LobbyState) SendToUser(userID uuid.UUID, msg *LobbyMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client.userID == userID {
			client.Send(msg)
		}
	}
}

// ============== Vote Management ==============

// CastVote records a vote (or updates an existing one)
func (s *LobbyState) CastVote(userID uuid.UUID, optionNumber int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.votes[userID] = &InMemoryVote{
		UserID:       userID,
		OptionNumber: optionNumber,
		CastAt:       time.Now(),
	}
}

// GetVote returns a user's vote
func (s *LobbyState) GetVote(userID uuid.UUID) *InMemoryVote {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.votes[userID]
}

// GetVotes returns all votes as userID -> optionNumber map
func (s *LobbyState) GetVotes() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int)
	for userID, vote := range s.votes {
		result[userID.String()] = vote.OptionNumber
	}
	return result
}

// GetVoteCounts returns vote counts per option
func (s *LobbyState) GetVoteCounts() map[int]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[int]int)
	for _, vote := range s.votes {
		counts[vote.OptionNumber]++
	}
	return counts
}

// GetTotalVotes returns the total number of votes cast
func (s *LobbyState) GetTotalVotes() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.votes)
}

// ClearVotes clears all votes
func (s *LobbyState) ClearVotes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.votes = make(map[uuid.UUID]*InMemoryVote)
}

// GetVotersByOption returns voters grouped by option number
func (s *LobbyState) GetVotersByOption() map[int][]uuid.UUID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[int][]uuid.UUID)
	for userID, vote := range s.votes {
		result[vote.OptionNumber] = append(result[vote.OptionNumber], userID)
	}
	return result
}

// ============== Pending Action Management ==============

// SetPendingAction sets the current pending action
func (s *LobbyState) SetPendingAction(action *InMemoryPendingAction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingAction = action
}

// GetPendingAction returns the current pending action
func (s *LobbyState) GetPendingAction() *InMemoryPendingAction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if expired
	if s.pendingAction != nil && s.pendingAction.IsExpired() {
		return nil
	}
	return s.pendingAction
}

// ClearPendingAction clears the current pending action
func (s *LobbyState) ClearPendingAction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingAction = nil
}

// ApprovePendingAction approves the pending action for a side
func (s *LobbyState) ApprovePendingAction(side string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pendingAction == nil || s.pendingAction.IsExpired() {
		return false
	}

	switch side {
	case "blue":
		s.pendingAction.ApprovedByBlue = true
	case "red":
		s.pendingAction.ApprovedByRed = true
	}

	return s.pendingAction.ApprovedByBlue && s.pendingAction.ApprovedByRed
}

// CreatePendingAction creates a new pending action
func (s *LobbyState) CreatePendingAction(
	actionType string,
	proposedByUser uuid.UUID,
	proposedBySide string,
	player1ID, player2ID *uuid.UUID,
	matchOptionNum *int,
) *InMemoryPendingAction {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if there's already a pending action
	if s.pendingAction != nil && !s.pendingAction.IsExpired() {
		return nil
	}

	action := &InMemoryPendingAction{
		ID:             uuid.New(),
		ActionType:     actionType,
		Status:         "pending",
		ProposedByUser: proposedByUser,
		ProposedBySide: proposedBySide,
		Player1ID:      player1ID,
		Player2ID:      player2ID,
		MatchOptionNum: matchOptionNum,
		ApprovedByBlue: proposedBySide == "blue",
		ApprovedByRed:  proposedBySide == "red",
		ExpiresAt:      time.Now().Add(60 * time.Second), // 60 second expiry
		CreatedAt:      time.Now(),
	}

	s.pendingAction = action
	return action
}

// HasPendingAction checks if there's an active pending action
func (s *LobbyState) HasPendingAction() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pendingAction != nil && !s.pendingAction.IsExpired()
}
