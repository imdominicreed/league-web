package websocket

import (
	"encoding/json"
)

// EventEmitter provides centralized message broadcasting for the room.
// All managers use this to send messages to clients.
type EventEmitter struct {
	room *Room
}

// NewEventEmitter creates a new event emitter for the room.
func NewEventEmitter(room *Room) *EventEmitter {
	return &EventEmitter{room: room}
}

// Broadcast sends a message to all clients in the room.
// Must be called with room lock held.
func (e *EventEmitter) Broadcast(msg *Message) {
	data, _ := json.Marshal(msg)
	for client := range e.room.clients {
		e.trySend(client, data)
	}
}

// BroadcastAsync sends a message through the broadcast channel (for use without lock).
func (e *EventEmitter) BroadcastAsync(msg *Message) {
	e.room.broadcast <- msg
}

// SendTo sends a message to a specific client.
func (e *EventEmitter) SendTo(client *Client, msg *Message) {
	client.Send(msg)
}

// trySend attempts to send to a client, safely handling closed channels.
func (e *EventEmitter) trySend(client *Client, data []byte) {
	defer func() {
		if recover() != nil {
			// Channel closed, client is disconnecting - skip silently
		}
	}()

	select {
	case client.send <- data:
	default:
		// Buffer full, skip
	}
}

// --- Draft lifecycle events ---

// DraftStarted broadcasts that the draft has begun.
func (e *EventEmitter) DraftStarted(phase, team, actionType string, timerMs int) {
	msg, _ := NewMessage(MessageTypeDraftStarted, DraftStartedPayload{
		CurrentPhase:     0,
		CurrentTeam:      team,
		ActionType:       actionType,
		TimerRemainingMs: timerMs,
	})
	e.Broadcast(msg)
}

// PhaseChanged broadcasts a phase transition.
func (e *EventEmitter) PhaseChanged(currentPhase int, team, actionType string, timerMs int) {
	msg, _ := NewMessage(MessageTypePhaseChanged, PhaseChangedPayload{
		CurrentPhase:     currentPhase,
		CurrentTeam:      team,
		ActionType:       actionType,
		TimerRemainingMs: timerMs,
	})
	e.Broadcast(msg)
}

// DraftCompleted broadcasts that the draft has finished.
func (e *EventEmitter) DraftCompleted(blueBans, redBans, bluePicks, redPicks []string) {
	msg, _ := NewMessage(MessageTypeDraftCompleted, DraftCompletedPayload{
		BlueBans:  blueBans,
		RedBans:   redBans,
		BluePicks: bluePicks,
		RedPicks:  redPicks,
	})
	e.Broadcast(msg)
}

// --- Champion events ---

// ChampionSelected broadcasts a champion lock-in.
func (e *EventEmitter) ChampionSelected(phase int, team, actionType, championID string) {
	msg, _ := NewMessage(MessageTypeChampionSelected, ChampionSelectedPayload{
		Phase:      phase,
		Team:       team,
		ActionType: actionType,
		ChampionID: championID,
	})
	e.Broadcast(msg)
}

// ChampionHovered broadcasts a hover preview.
func (e *EventEmitter) ChampionHovered(side string, championID *string) {
	msg, _ := NewMessage(MessageTypeChampionHovered, ChampionHoveredPayload{
		Side:       side,
		ChampionID: championID,
	})
	e.Broadcast(msg)
}

// --- Player events ---

// PlayerUpdate broadcasts a player joined/left/ready change.
func (e *EventEmitter) PlayerUpdate(side string, player *PlayerInfo, action string) {
	msg, _ := NewMessage(MessageTypePlayerUpdate, PlayerUpdatePayload{
		Side:   side,
		Player: player,
		Action: action,
	})
	e.Broadcast(msg)
}

// --- Timer events ---

// TimerTick broadcasts a timer update.
func (e *EventEmitter) TimerTick(remainingMs int, isBufferPeriod bool) {
	msg, _ := NewMessage(MessageTypeTimerTick, TimerTickPayload{
		RemainingMs:    remainingMs,
		IsBufferPeriod: isBufferPeriod,
	})
	e.BroadcastAsync(msg)
}

// --- Pause/Resume events ---

// DraftPaused broadcasts that the draft has been paused.
func (e *EventEmitter) DraftPaused(pausedByName, side string, timerFrozenMs, maxPauseMs int) {
	msg, _ := NewMessage(MessageTypeDraftPaused, DraftPausedPayload{
		PausedBy:         pausedByName,
		PausedBySide:     side,
		TimerFrozenAt:    timerFrozenMs,
		MaxPauseDuration: maxPauseMs,
	})
	e.Broadcast(msg)
}

// DraftResumed broadcasts that the draft has resumed.
func (e *EventEmitter) DraftResumed(resumedBy string, timerRemainingMs int) {
	msg, _ := NewMessage(MessageTypeDraftResumed, DraftResumedPayload{
		ResumedBy:        resumedBy,
		TimerRemainingMs: timerRemainingMs,
	})
	e.Broadcast(msg)
}

// ResumeReadyUpdate broadcasts the resume-ready status of both sides.
func (e *EventEmitter) ResumeReadyUpdate(blueReady, redReady bool) {
	msg, _ := NewMessage(MessageTypeResumeReadyUpdate, ResumeReadyUpdatePayload{
		BlueReady: blueReady,
		RedReady:  redReady,
	})
	e.Broadcast(msg)
}

// ResumeCountdown broadcasts the countdown seconds before resume.
func (e *EventEmitter) ResumeCountdown(seconds int, cancelledBy string) {
	msg, _ := NewMessage(MessageTypeResumeCountdown, ResumeCountdownPayload{
		SecondsRemaining: seconds,
		CancelledBy:      cancelledBy,
	})
	e.Broadcast(msg)
}

// --- Edit events ---

// EditProposed broadcasts an edit proposal.
func (e *EventEmitter) EditProposed(proposedByName, proposedSide, slotType, team string, slotIndex int, oldChampionID, newChampionID string, expiresAt int64) {
	msg, _ := NewMessage(MessageTypeEditProposed, EditProposedPayload{
		ProposedBy:    proposedByName,
		ProposedSide:  proposedSide,
		SlotType:      slotType,
		Team:          team,
		SlotIndex:     slotIndex,
		OldChampionID: oldChampionID,
		NewChampionID: newChampionID,
		ExpiresAt:     expiresAt,
	})
	e.Broadcast(msg)
}

// EditApplied broadcasts that an edit has been applied.
func (e *EventEmitter) EditApplied(slotType, team string, slotIndex int, oldChampionID, newChampionID string, blueBans, redBans, bluePicks, redPicks []string) {
	msg, _ := NewMessage(MessageTypeEditApplied, EditAppliedPayload{
		SlotType:      slotType,
		Team:          team,
		SlotIndex:     slotIndex,
		OldChampionID: oldChampionID,
		NewChampionID: newChampionID,
		BlueBans:      blueBans,
		RedBans:       redBans,
		BluePicks:     bluePicks,
		RedPicks:      redPicks,
	})
	e.Broadcast(msg)
}

// EditRejected broadcasts that an edit has been rejected.
func (e *EventEmitter) EditRejected(rejectedByName, rejectedSide string) {
	msg, _ := NewMessage(MessageTypeEditRejected, EditRejectedPayload{
		RejectedBy:   rejectedByName,
		RejectedSide: rejectedSide,
	})
	e.Broadcast(msg)
}

// --- Error events ---

// SendError sends an error to a specific client.
func (e *EventEmitter) SendError(client *Client, code, message string) {
	client.sendError(code, message)
}
