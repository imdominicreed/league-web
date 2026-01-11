package websocket

import (
	"context"
	"log"
	"math/rand"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
)

// DraftState holds the current state of the draft.
type DraftState struct {
	CurrentPhase int
	BlueBans     []string
	RedBans      []string
	BluePicks    []string
	RedPicks     []string
	IsComplete   bool
	BlueReady    bool
	RedReady     bool
	Started      bool
}

// DraftStateManager handles draft phase transitions and champion selections.
type DraftStateManager struct {
	state        *DraftState
	currentHover map[string]*string // side -> championId
	championRepo repository.ChampionRepository
	emitter      *EventEmitter
	timerMgr     *TimerManager
	timerDuration int

	// Callbacks
	onDraftComplete func()
	sendStateSync   func() // Called to sync state after draft start/complete
}

// NewDraftStateManager creates a new draft state manager.
func NewDraftStateManager(
	championRepo repository.ChampionRepository,
	emitter *EventEmitter,
	timerDuration int,
) *DraftStateManager {
	return &DraftStateManager{
		state: &DraftState{
			CurrentPhase: 0,
			BlueBans:     []string{},
			RedBans:      []string{},
			BluePicks:    []string{},
			RedPicks:     []string{},
		},
		currentHover:  make(map[string]*string),
		championRepo:  championRepo,
		emitter:       emitter,
		timerDuration: timerDuration,
	}
}

// SetTimerManager sets the timer manager (for circular dependency resolution).
func (dm *DraftStateManager) SetTimerManager(tm *TimerManager) {
	dm.timerMgr = tm
}

// SetCallbacks sets the callbacks for draft events.
func (dm *DraftStateManager) SetCallbacks(onComplete func(), sendStateSync func()) {
	dm.onDraftComplete = onComplete
	dm.sendStateSync = sendStateSync
}

// GetState returns the current draft state.
func (dm *DraftStateManager) GetState() *DraftState {
	return dm.state
}

// IsStarted returns whether the draft has started.
func (dm *DraftStateManager) IsStarted() bool {
	return dm.state.Started
}

// IsComplete returns whether the draft is complete.
func (dm *DraftStateManager) IsComplete() bool {
	return dm.state.IsComplete
}

// GetCurrentPhase returns the current phase.
func (dm *DraftStateManager) GetCurrentPhase() int {
	return dm.state.CurrentPhase
}

// GetCurrentSide returns the current team's side ("blue" or "red").
func (dm *DraftStateManager) GetCurrentSide() string {
	phase := domain.GetPhase(dm.state.CurrentPhase)
	if phase == nil {
		return ""
	}
	return string(phase.Team)
}

// SetReady sets the ready status for a side.
func (dm *DraftStateManager) SetReady(side string, ready bool) {
	switch side {
	case "blue":
		dm.state.BlueReady = ready
	case "red":
		dm.state.RedReady = ready
	}
}

// BothReady returns whether both sides are ready.
func (dm *DraftStateManager) BothReady() bool {
	return dm.state.BlueReady && dm.state.RedReady
}

// Start starts the draft.
func (dm *DraftStateManager) Start() {
	dm.state.Started = true

	phase := domain.GetPhase(0)

	// Broadcast draft started
	dm.emitter.DraftStarted(
		"0",
		string(phase.Team),
		string(phase.ActionType),
		dm.timerDuration,
	)

	// Send state sync to update room status
	if dm.sendStateSync != nil {
		dm.sendStateSync()
	}

	// Start the timer
	if dm.timerMgr != nil {
		dm.timerMgr.Start()
	}
}

// SelectChampion handles champion selection (hover before lock).
func (dm *DraftStateManager) SelectChampion(side string, championID string) error {
	if !dm.state.Started || dm.state.IsComplete {
		return &DraftError{"invalid_state", "Draft not in progress"}
	}

	phase := domain.GetPhase(dm.state.CurrentPhase)
	if phase == nil {
		return &DraftError{"invalid_phase", "Invalid phase"}
	}

	currentSide := string(phase.Team)
	if side != currentSide {
		return &DraftError{"not_your_turn", "It's not your turn"}
	}

	// Check if champion is already picked/banned
	if dm.isChampionUsed(championID) {
		return &DraftError{"champion_unavailable", "Champion is already picked or banned"}
	}

	// Store selection (will be confirmed on lock in)
	dm.currentHover[currentSide] = &championID

	// Broadcast hover
	dm.emitter.ChampionHovered(currentSide, &championID)

	return nil
}

// LockIn locks in the currently selected champion.
func (dm *DraftStateManager) LockIn(side string) error {
	if !dm.state.Started || dm.state.IsComplete {
		return &DraftError{"invalid_state", "Draft not in progress"}
	}

	phase := domain.GetPhase(dm.state.CurrentPhase)
	if phase == nil {
		return &DraftError{"invalid_phase", "Invalid phase"}
	}

	currentSide := string(phase.Team)
	if side != currentSide {
		return &DraftError{"not_your_turn", "It's not your turn"}
	}

	championID := dm.currentHover[currentSide]
	if championID == nil {
		none := "None"
		championID = &none
	}

	// Apply the selection
	dm.applySelection(phase, *championID)

	// Stop current timer
	if dm.timerMgr != nil {
		dm.timerMgr.Stop()
	}

	// Broadcast selection
	dm.emitter.ChampionSelected(
		dm.state.CurrentPhase,
		string(phase.Team),
		string(phase.ActionType),
		*championID,
	)

	// Move to next phase
	dm.advancePhase()

	return nil
}

// HoverChampion handles hover preview (no validation, just broadcast).
func (dm *DraftStateManager) HoverChampion(side string, championID *string) {
	if !dm.state.Started {
		return
	}
	dm.emitter.ChampionHovered(side, championID)
}

// HandleTimerExpired handles timer expiration (auto-pick/ban).
func (dm *DraftStateManager) HandleTimerExpired() {
	if dm.state.IsComplete {
		return
	}

	phase := domain.GetPhase(dm.state.CurrentPhase)
	if phase == nil {
		return
	}

	var championID string

	if phase.ActionType == domain.ActionTypeBan {
		// Missed ban - use "None" (skip the ban)
		championID = "None"
	} else {
		// Missed pick - select a random available champion
		championID = dm.getRandomAvailableChampion()
	}

	dm.applySelection(phase, championID)

	// Broadcast selection
	dm.emitter.ChampionSelected(
		dm.state.CurrentPhase,
		string(phase.Team),
		string(phase.ActionType),
		championID,
	)

	dm.advancePhase()
}

// advancePhase moves to the next draft phase.
func (dm *DraftStateManager) advancePhase() {
	dm.state.CurrentPhase++

	// Clear hover for next phase
	dm.currentHover = make(map[string]*string)

	if dm.state.CurrentPhase >= domain.TotalPhases() {
		dm.state.IsComplete = true

		dm.emitter.DraftCompleted(
			dm.state.BlueBans,
			dm.state.RedBans,
			dm.state.BluePicks,
			dm.state.RedPicks,
		)

		// Send state sync to update room status
		if dm.sendStateSync != nil {
			dm.sendStateSync()
		}

		if dm.onDraftComplete != nil {
			dm.onDraftComplete()
		}

		return
	}

	phase := domain.GetPhase(dm.state.CurrentPhase)

	dm.emitter.PhaseChanged(
		dm.state.CurrentPhase,
		string(phase.Team),
		string(phase.ActionType),
		dm.timerDuration,
	)

	// Start timer for next phase
	if dm.timerMgr != nil {
		dm.timerMgr.Start()
	}
}

// applySelection applies a selection to the draft state.
func (dm *DraftStateManager) applySelection(phase *domain.Phase, championID string) {
	switch phase.ActionType {
	case domain.ActionTypeBan:
		if phase.Team == domain.SideBlue {
			dm.state.BlueBans = append(dm.state.BlueBans, championID)
		} else {
			dm.state.RedBans = append(dm.state.RedBans, championID)
		}
	case domain.ActionTypePick:
		if phase.Team == domain.SideBlue {
			dm.state.BluePicks = append(dm.state.BluePicks, championID)
		} else {
			dm.state.RedPicks = append(dm.state.RedPicks, championID)
		}
	}
}

// isChampionUsed checks if a champion is already picked or banned.
func (dm *DraftStateManager) isChampionUsed(championID string) bool {
	for _, id := range dm.state.BlueBans {
		if id == championID {
			return true
		}
	}
	for _, id := range dm.state.RedBans {
		if id == championID {
			return true
		}
	}
	for _, id := range dm.state.BluePicks {
		if id == championID {
			return true
		}
	}
	for _, id := range dm.state.RedPicks {
		if id == championID {
			return true
		}
	}
	return false
}

// IsChampionUsedExcept checks if a champion is used anywhere except the specified slot.
// Used by EditManager for validation.
func (dm *DraftStateManager) IsChampionUsedExcept(championID, exceptSlotType, exceptTeam string, exceptSlotIndex int) bool {
	// Check blue bans
	for i, id := range dm.state.BlueBans {
		if id == championID {
			if exceptSlotType == "ban" && exceptTeam == "blue" && i == exceptSlotIndex {
				continue // This is the slot being edited
			}
			return true
		}
	}
	// Check red bans
	for i, id := range dm.state.RedBans {
		if id == championID {
			if exceptSlotType == "ban" && exceptTeam == "red" && i == exceptSlotIndex {
				continue
			}
			return true
		}
	}
	// Check blue picks
	for i, id := range dm.state.BluePicks {
		if id == championID {
			if exceptSlotType == "pick" && exceptTeam == "blue" && i == exceptSlotIndex {
				continue
			}
			return true
		}
	}
	// Check red picks
	for i, id := range dm.state.RedPicks {
		if id == championID {
			if exceptSlotType == "pick" && exceptTeam == "red" && i == exceptSlotIndex {
				continue
			}
			return true
		}
	}
	return false
}

// ApplyEdit applies an edit to the draft state.
func (dm *DraftStateManager) ApplyEdit(edit *PendingEdit) {
	var arr *[]string
	switch {
	case edit.SlotType == "ban" && edit.Team == "blue":
		arr = &dm.state.BlueBans
	case edit.SlotType == "ban" && edit.Team == "red":
		arr = &dm.state.RedBans
	case edit.SlotType == "pick" && edit.Team == "blue":
		arr = &dm.state.BluePicks
	case edit.SlotType == "pick" && edit.Team == "red":
		arr = &dm.state.RedPicks
	}

	if arr != nil && edit.SlotIndex < len(*arr) {
		(*arr)[edit.SlotIndex] = edit.NewChampionID
	}
}

// getRandomAvailableChampion returns a random champion that hasn't been picked or banned.
func (dm *DraftStateManager) getRandomAvailableChampion() string {
	if dm.championRepo == nil {
		log.Printf("Warning: championRepo is nil, cannot get random champion")
		return "None"
	}

	champions, err := dm.championRepo.GetAll(context.Background())
	if err != nil {
		log.Printf("Error getting champions: %v", err)
		return "None"
	}

	// Filter out used champions
	var available []string
	for _, c := range champions {
		if !dm.isChampionUsed(c.ID) {
			available = append(available, c.ID)
		}
	}

	if len(available) == 0 {
		log.Printf("Warning: no available champions for random pick")
		return "None"
	}

	// Pick a random one
	return available[rand.Intn(len(available))]
}

// DraftError represents a draft-related error.
type DraftError struct {
	Code    string
	Message string
}

func (e *DraftError) Error() string {
	return e.Message
}
