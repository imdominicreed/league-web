package websocket

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PendingEdit represents an edit proposal awaiting confirmation.
type PendingEdit struct {
	ProposedBy    uuid.UUID
	ProposedSide  string
	SlotType      string // "ban" or "pick"
	Team          string // "blue" or "red"
	SlotIndex     int
	OldChampionID string
	NewChampionID string
	ExpiresAt     time.Time
}

// EditManager handles edit proposal, confirmation, and rejection logic.
type EditManager struct {
	pendingEdit  *PendingEdit
	editTimeout  *time.Timer

	// Dependencies
	emitter        *EventEmitter
	getUserName    func(uuid.UUID) string
	getDraftState  func() *DraftState
	applyEdit      func(edit *PendingEdit)
	isChampionUsed func(championID string, exceptSlotType, exceptTeam string, exceptSlotIndex int) bool

	mu sync.RWMutex
}

// NewEditManager creates a new edit manager.
func NewEditManager(
	emitter *EventEmitter,
	getUserName func(uuid.UUID) string,
	getDraftState func() *DraftState,
	applyEdit func(edit *PendingEdit),
	isChampionUsed func(championID string, exceptSlotType, exceptTeam string, exceptSlotIndex int) bool,
) *EditManager {
	return &EditManager{
		emitter:        emitter,
		getUserName:    getUserName,
		getDraftState:  getDraftState,
		applyEdit:      applyEdit,
		isChampionUsed: isChampionUsed,
	}
}

// HasPendingEdit returns whether there's a pending edit.
func (em *EditManager) HasPendingEdit() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.pendingEdit != nil
}

// GetPendingEdit returns the current pending edit.
func (em *EditManager) GetPendingEdit() *PendingEdit {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.pendingEdit
}

// ProposeEdit creates a new edit proposal.
func (em *EditManager) ProposeEdit(userID uuid.UUID, side string, payload ProposeEditPayload) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// No pending edit already
	if em.pendingEdit != nil {
		return &EditError{"edit_pending", "An edit is already pending"}
	}

	// Get current draft state to validate slot
	draftState := em.getDraftState()

	// Validate slot exists and get old champion
	oldChampionID, err := em.getChampionAtSlot(draftState, payload.SlotType, payload.Team, payload.SlotIndex)
	if err != nil {
		return &EditError{"invalid_slot", err.Error()}
	}

	// Validate new champion is available (not already picked/banned elsewhere)
	if em.isChampionUsed(payload.ChampionID, payload.SlotType, payload.Team, payload.SlotIndex) {
		return &EditError{"champion_unavailable", "Champion already used"}
	}

	// Create pending edit
	em.pendingEdit = &PendingEdit{
		ProposedBy:    userID,
		ProposedSide:  side,
		SlotType:      payload.SlotType,
		Team:          payload.Team,
		SlotIndex:     payload.SlotIndex,
		OldChampionID: oldChampionID,
		NewChampionID: payload.ChampionID,
		ExpiresAt:     time.Now().Add(30 * time.Second),
	}

	// Start timeout timer
	em.editTimeout = time.AfterFunc(30*time.Second, em.handleTimeout)

	log.Printf("Edit proposed by %s: %s %s slot %d: %s -> %s",
		userID, payload.Team, payload.SlotType, payload.SlotIndex,
		oldChampionID, payload.ChampionID)

	// Broadcast proposal
	em.emitter.EditProposed(
		em.getUserName(userID),
		side,
		payload.SlotType,
		payload.Team,
		payload.SlotIndex,
		oldChampionID,
		payload.ChampionID,
		em.pendingEdit.ExpiresAt.UnixMilli(),
	)

	return nil
}

// ConfirmEdit confirms the pending edit.
func (em *EditManager) ConfirmEdit(userID uuid.UUID, side string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.pendingEdit == nil {
		return &EditError{"no_edit", "No edit pending"}
	}

	// Confirmer must be opposite side
	if side == em.pendingEdit.ProposedSide {
		return &EditError{"invalid_confirm", "Cannot confirm your own edit"}
	}

	// Apply the edit
	em.applyEdit(em.pendingEdit)

	log.Printf("Edit confirmed by %s: %s %s slot %d: %s -> %s",
		userID, em.pendingEdit.Team, em.pendingEdit.SlotType, em.pendingEdit.SlotIndex,
		em.pendingEdit.OldChampionID, em.pendingEdit.NewChampionID)

	// Get updated draft state for broadcast
	draftState := em.getDraftState()

	// Broadcast edit applied
	em.emitter.EditApplied(
		em.pendingEdit.SlotType,
		em.pendingEdit.Team,
		em.pendingEdit.SlotIndex,
		em.pendingEdit.OldChampionID,
		em.pendingEdit.NewChampionID,
		draftState.BlueBans,
		draftState.RedBans,
		draftState.BluePicks,
		draftState.RedPicks,
	)

	// Clear pending edit
	em.clearLocked()

	return nil
}

// RejectEdit rejects the pending edit.
func (em *EditManager) RejectEdit(userID uuid.UUID, side string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.pendingEdit == nil {
		return &EditError{"no_edit", "No edit pending"}
	}

	// Rejecter must be opposite side
	if side == em.pendingEdit.ProposedSide {
		return &EditError{"invalid_reject", "Cannot reject your own edit"}
	}

	log.Printf("Edit rejected by %s", userID)

	// Broadcast rejection
	em.emitter.EditRejected(em.getUserName(userID), side)

	// Clear pending edit
	em.clearLocked()

	return nil
}

// Clear clears the pending edit (called externally, e.g., on resume).
func (em *EditManager) Clear() {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.clearLocked()
}

// clearLocked clears the pending edit and stops the timeout timer.
// Must be called with lock held.
func (em *EditManager) clearLocked() {
	if em.editTimeout != nil {
		em.editTimeout.Stop()
		em.editTimeout = nil
	}
	em.pendingEdit = nil
}

// handleTimeout is called when an edit proposal times out.
func (em *EditManager) handleTimeout() {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.pendingEdit == nil {
		return
	}

	log.Printf("Edit proposal timed out")

	// Broadcast rejection (timeout)
	em.emitter.EditRejected("System (timeout)", "")

	// Clear pending edit
	em.clearLocked()
}

// getChampionAtSlot returns the champion at the specified slot.
func (em *EditManager) getChampionAtSlot(draftState *DraftState, slotType, team string, slotIndex int) (string, error) {
	var arr []string
	switch {
	case slotType == "ban" && team == "blue":
		arr = draftState.BlueBans
	case slotType == "ban" && team == "red":
		arr = draftState.RedBans
	case slotType == "pick" && team == "blue":
		arr = draftState.BluePicks
	case slotType == "pick" && team == "red":
		arr = draftState.RedPicks
	default:
		return "", fmt.Errorf("invalid slot type or team")
	}

	if slotIndex < 0 || slotIndex >= len(arr) {
		return "", fmt.Errorf("slot index out of range")
	}

	return arr[slotIndex], nil
}

// BuildPendingEditInfo builds PendingEditInfo for state sync.
func (em *EditManager) BuildPendingEditInfo() *PendingEditInfo {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if em.pendingEdit == nil {
		return nil
	}

	return &PendingEditInfo{
		ProposedBy:    em.getUserName(em.pendingEdit.ProposedBy),
		ProposedSide:  em.pendingEdit.ProposedSide,
		SlotType:      em.pendingEdit.SlotType,
		Team:          em.pendingEdit.Team,
		SlotIndex:     em.pendingEdit.SlotIndex,
		OldChampionID: em.pendingEdit.OldChampionID,
		NewChampionID: em.pendingEdit.NewChampionID,
		ExpiresAt:     em.pendingEdit.ExpiresAt.UnixMilli(),
	}
}

// EditError represents an edit-related error.
type EditError struct {
	Code    string
	Message string
}

func (e *EditError) Error() string {
	return e.Message
}
