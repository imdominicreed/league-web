package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PauseManager handles draft pause, resume-ready, and countdown logic.
type PauseManager struct {
	// Pause state
	isPaused           bool
	pausedBy           *uuid.UUID
	pausedBySide       string
	pausedAt           time.Time
	frozenTimerMs      int
	maxPauseDurationMs int
	pauseTimer         *time.Timer // For auto-resume

	// Resume ready state
	blueResumeReady       bool
	redResumeReady        bool
	resumeCountdown       int
	resumeCountdownCancel chan struct{}

	// Dependencies
	emitter    *EventEmitter
	timerMgr   *TimerManager
	onResume   func(remainingMs int) // Callback when draft resumes
	getUserName func(uuid.UUID) string

	mu sync.RWMutex
}

// NewPauseManager creates a new pause manager.
func NewPauseManager(emitter *EventEmitter, timerMgr *TimerManager, getUserName func(uuid.UUID) string, onResume func(remainingMs int)) *PauseManager {
	return &PauseManager{
		maxPauseDurationMs: 300000, // 5 minutes
		emitter:            emitter,
		timerMgr:           timerMgr,
		getUserName:        getUserName,
		onResume:           onResume,
	}
}

// IsPaused returns whether the draft is currently paused.
func (pm *PauseManager) IsPaused() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isPaused
}

// GetPausedBy returns the user ID of who paused.
func (pm *PauseManager) GetPausedBy() *uuid.UUID {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.pausedBy
}

// GetPausedBySide returns the side of who paused.
func (pm *PauseManager) GetPausedBySide() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.pausedBySide
}

// GetFrozenTimerMs returns the timer value when paused.
func (pm *PauseManager) GetFrozenTimerMs() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.frozenTimerMs
}

// GetResumeReady returns the resume-ready status for both sides.
func (pm *PauseManager) GetResumeReady() (blue bool, red bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.blueResumeReady, pm.redResumeReady
}

// GetResumeCountdown returns the current countdown value.
func (pm *PauseManager) GetResumeCountdown() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.resumeCountdown
}

// Pause pauses the draft.
func (pm *PauseManager) Pause(userID uuid.UUID, side string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isPaused {
		return ErrAlreadyPaused
	}

	// Pause the timer and get remaining time
	pm.frozenTimerMs = pm.timerMgr.Pause()

	// Set pause state
	pm.isPaused = true
	pm.pausedBy = &userID
	pm.pausedBySide = side
	pm.pausedAt = time.Now()

	// Reset resume-ready state
	pm.blueResumeReady = false
	pm.redResumeReady = false
	pm.resumeCountdown = 0

	// Start auto-resume timer (5 minutes)
	pm.pauseTimer = time.AfterFunc(
		time.Duration(pm.maxPauseDurationMs)*time.Millisecond,
		pm.handleAutoResume,
	)

	log.Printf("Draft paused by %s (%s side), timer frozen at %dms", userID, side, pm.frozenTimerMs)

	// Broadcast pause event
	pm.emitter.DraftPaused(pm.getUserName(userID), side, pm.frozenTimerMs, pm.maxPauseDurationMs)

	return nil
}

// SetResumeReady updates the resume-ready status for a player.
func (pm *PauseManager) SetResumeReady(userID uuid.UUID, side string, ready bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isPaused {
		return ErrNotPaused
	}

	// If countdown is in progress and someone un-readies, cancel it
	if pm.resumeCountdown > 0 && !ready {
		pm.cancelCountdownLocked(userID)
		return nil
	}

	// Update ready state
	if side == "blue" {
		pm.blueResumeReady = ready
	} else {
		pm.redResumeReady = ready
	}

	// Broadcast ready update
	pm.emitter.ResumeReadyUpdate(pm.blueResumeReady, pm.redResumeReady)

	// Check if both ready - start countdown
	if pm.blueResumeReady && pm.redResumeReady && pm.resumeCountdown == 0 {
		pm.startCountdownLocked()
	}

	return nil
}

// startCountdownLocked starts the 5-second countdown before resuming.
// Must be called with lock held.
func (pm *PauseManager) startCountdownLocked() {
	pm.resumeCountdown = 5
	pm.resumeCountdownCancel = make(chan struct{})

	// Broadcast initial countdown
	pm.emitter.ResumeCountdown(5, "")

	log.Printf("Resume countdown started (5 seconds)")

	// Start countdown in goroutine
	go pm.runCountdownTicker()
}

// runCountdownTicker ticks down the resume countdown.
func (pm *PauseManager) runCountdownTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.resumeCountdownCancel:
			return
		case <-ticker.C:
			pm.mu.Lock()

			if pm.resumeCountdown <= 0 {
				pm.mu.Unlock()
				return
			}

			pm.resumeCountdown--

			if pm.resumeCountdown <= 0 {
				// Countdown complete - resume draft
				pm.doResumeLocked()
				pm.mu.Unlock()
				return
			}

			// Broadcast countdown tick
			pm.emitter.ResumeCountdown(pm.resumeCountdown, "")
			pm.mu.Unlock()
		}
	}
}

// cancelCountdownLocked cancels an ongoing resume countdown.
// Must be called with lock held.
func (pm *PauseManager) cancelCountdownLocked(cancelledBy uuid.UUID) {
	// Stop countdown goroutine
	if pm.resumeCountdownCancel != nil {
		close(pm.resumeCountdownCancel)
		pm.resumeCountdownCancel = nil
	}

	// Reset state
	pm.resumeCountdown = 0
	pm.blueResumeReady = false
	pm.redResumeReady = false

	log.Printf("Resume countdown cancelled by %s", cancelledBy)

	// Broadcast cancellation
	pm.emitter.ResumeCountdown(0, pm.getUserName(cancelledBy))

	// Broadcast ready update (both false)
	pm.emitter.ResumeReadyUpdate(false, false)
}

// doResumeLocked actually resumes the draft after countdown completes.
// Must be called with lock held.
func (pm *PauseManager) doResumeLocked() {
	// Stop auto-resume timer
	if pm.pauseTimer != nil {
		pm.pauseTimer.Stop()
		pm.pauseTimer = nil
	}

	// Save remaining time
	remainingMs := pm.frozenTimerMs

	// Clear pause and resume-ready state
	pm.isPaused = false
	pm.pausedBy = nil
	pm.pausedBySide = ""
	pm.blueResumeReady = false
	pm.redResumeReady = false
	pm.resumeCountdown = 0

	log.Printf("Draft resumed after countdown, timer restarting from %dms", remainingMs)

	// Broadcast resume
	pm.emitter.DraftResumed("Both players ready", remainingMs)

	// Callback to room to restart timer
	if pm.onResume != nil {
		pm.onResume(remainingMs)
	}
}

// handleAutoResume is called when the pause timer expires.
func (pm *PauseManager) handleAutoResume() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isPaused {
		return
	}

	// Cancel any ongoing countdown
	if pm.resumeCountdownCancel != nil {
		close(pm.resumeCountdownCancel)
		pm.resumeCountdownCancel = nil
	}

	// Save remaining time before clearing pause state
	remainingMs := pm.frozenTimerMs

	// Clear pause state and resume-ready state
	pm.isPaused = false
	pm.pausedBy = nil
	pm.pausedBySide = ""
	pm.blueResumeReady = false
	pm.redResumeReady = false
	pm.resumeCountdown = 0

	log.Printf("Draft auto-resumed after pause timeout, timer restarting from %dms", remainingMs)

	// Broadcast resume
	pm.emitter.DraftResumed("System (timeout)", remainingMs)

	// Callback to room to restart timer
	if pm.onResume != nil {
		pm.onResume(remainingMs)
	}
}

// ClearForEdit is called when clearing pending edit (needed for resume).
func (pm *PauseManager) ClearForEdit() {
	// This is a hook for the room to clear pending edits when resuming
	// The actual edit clearing is done by EditManager
}

// Errors
var (
	ErrAlreadyPaused = &PauseError{"already_paused", "Draft is already paused"}
	ErrNotPaused     = &PauseError{"not_paused", "Draft is not paused"}
)

type PauseError struct {
	Code    string
	Message string
}

func (e *PauseError) Error() string {
	return e.Message
}
