package websocket

import (
	"sync"
	"time"
)

const bufferDurationMsConst = 5000 // 5 second buffer after timer hits 0

// TimerManager handles draft timer logic including ticking, expiry, and pause/resume.
type TimerManager struct {
	durationMs    int
	timer         *time.Timer
	timerStarted  time.Time
	tickerStop    chan struct{}
	onExpired     func()
	emitter       *EventEmitter

	// For pause support
	frozenMs      int
	isPaused      bool

	mu sync.RWMutex
}

// NewTimerManager creates a new timer manager.
func NewTimerManager(durationMs int, emitter *EventEmitter, onExpired func()) *TimerManager {
	return &TimerManager{
		durationMs: durationMs,
		emitter:    emitter,
		onExpired:  onExpired,
	}
}

// Start begins the timer for a new phase.
func (tm *TimerManager) Start() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.timerStarted = time.Now()
	tm.isPaused = false

	// Timer fires after main duration + buffer period
	totalDuration := tm.durationMs + bufferDurationMsConst
	tm.timer = time.AfterFunc(time.Duration(totalDuration)*time.Millisecond, func() {
		tm.onExpired()
	})

	// Start ticker for timer updates
	tm.tickerStop = make(chan struct{})
	go tm.runTicker()
}

// Stop stops the timer and ticker.
func (tm *TimerManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.timer != nil {
		tm.timer.Stop()
		tm.timer = nil
	}
	if tm.tickerStop != nil {
		close(tm.tickerStop)
		tm.tickerStop = nil
	}
}

// Pause pauses the timer and returns the remaining milliseconds.
func (tm *TimerManager) Pause() int {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.timer != nil {
		tm.timer.Stop()
	}
	if tm.tickerStop != nil {
		close(tm.tickerStop)
		tm.tickerStop = nil
	}

	elapsed := time.Since(tm.timerStarted)
	tm.frozenMs = tm.durationMs - int(elapsed.Milliseconds())
	if tm.frozenMs < 0 {
		tm.frozenMs = 0
	}
	tm.isPaused = true

	return tm.frozenMs
}

// Resume restarts the timer from the frozen position.
func (tm *TimerManager) Resume() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.durationMs = tm.frozenMs
	tm.isPaused = false
	tm.mu.Unlock()

	// Use Start() which will acquire lock again
	tm.mu.Lock()
	tm.timerStarted = time.Now()

	totalDuration := tm.durationMs + bufferDurationMsConst
	tm.timer = time.AfterFunc(time.Duration(totalDuration)*time.Millisecond, func() {
		tm.onExpired()
	})

	tm.tickerStop = make(chan struct{})
	go tm.runTicker()
}

// GetRemaining returns the remaining milliseconds.
func (tm *TimerManager) GetRemaining() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.isPaused {
		return tm.frozenMs
	}

	elapsed := time.Since(tm.timerStarted)
	remaining := tm.durationMs - int(elapsed.Milliseconds())
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// GetDuration returns the base timer duration.
func (tm *TimerManager) GetDuration() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.durationMs
}

// SetDuration sets the timer duration (for resuming with remaining time).
func (tm *TimerManager) SetDuration(ms int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.durationMs = ms
}

// IsPaused returns whether the timer is paused.
func (tm *TimerManager) IsPaused() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.isPaused
}

// GetFrozenMs returns the frozen timer value (when paused).
func (tm *TimerManager) GetFrozenMs() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.frozenMs
}

// runTicker sends timer tick messages every second.
func (tm *TimerManager) runTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tm.tickerStop:
			return
		case <-ticker.C:
			tm.mu.RLock()
			if tm.isPaused {
				tm.mu.RUnlock()
				continue
			}

			elapsed := time.Since(tm.timerStarted)
			remaining := tm.durationMs - int(elapsed.Milliseconds())

			// Check if we're in the buffer period (past main timer but before auto-lock)
			isBufferPeriod := remaining <= 0

			// Display 0 during buffer period (don't show negative)
			displayRemaining := remaining
			if displayRemaining < 0 {
				displayRemaining = 0
			}
			tm.mu.RUnlock()

			tm.emitter.TimerTick(displayRemaining, isBufferPeriod)

			// Stop ticker after buffer period expires
			if remaining <= -bufferDurationMsConst {
				return
			}
		}
	}
}
