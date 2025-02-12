package states

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents a service state
type State string

const (
	// StateUninitialized represents an uninitialized service
	StateUninitialized State = "uninitialized"
	// StateInitializing represents a service that is initializing
	StateInitializing State = "initializing"
	// StateReady represents a service that is ready to start
	StateReady State = "ready"
	// StateStarting represents a service that is starting
	StateStarting State = "starting"
	// StateRunning represents a running service
	StateRunning State = "running"
	// StateStopping represents a service that is stopping
	StateStopping State = "stopping"
	// StateStopped represents a stopped service
	StateStopped State = "stopped"
	// StateError represents a service in error state
	StateError State = "error"
)

// TransitionError represents a state transition error
type TransitionError struct {
	From      State
	To        State
	Message   string
	Timestamp time.Time
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("invalid state transition from %s to %s: %s", e.From, e.To, e.Message)
}

// StateMachine manages service state transitions
type StateMachine struct {
	mu            sync.RWMutex
	currentState  State
	previousState State
	logger        *zap.Logger
	transitions   map[State][]State
	handlers      map[State]StateHandler
	errors        []TransitionError
	ctx           context.Context
	cancel        context.CancelFunc
}

// StateHandler handles state transitions
type StateHandler interface {
	OnEnter(ctx context.Context) error
	OnExit(ctx context.Context) error
	Validate(ctx context.Context) error
}

// NewStateMachine creates a new state machine
func NewStateMachine(logger *zap.Logger) *StateMachine {
	ctx, cancel := context.WithCancel(context.Background())
	sm := &StateMachine{
		currentState: StateUninitialized,
		logger:       logger,
		transitions:  make(map[State][]State),
		handlers:     make(map[State]StateHandler),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Define valid state transitions
	sm.transitions[StateUninitialized] = []State{StateInitializing}
	sm.transitions[StateInitializing] = []State{StateReady, StateError}
	sm.transitions[StateReady] = []State{StateStarting, StateStopped}
	sm.transitions[StateStarting] = []State{StateRunning, StateError}
	sm.transitions[StateRunning] = []State{StateStopping, StateError}
	sm.transitions[StateStopping] = []State{StateStopped, StateError}
	sm.transitions[StateStopped] = []State{StateInitializing}
	sm.transitions[StateError] = []State{StateInitializing, StateStopped}

	return sm
}

// RegisterHandler registers a handler for a state
func (sm *StateMachine) RegisterHandler(state State, handler StateHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers[state] = handler
}

// CurrentState returns the current state
func (sm *StateMachine) CurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// TransitionTo attempts to transition to a new state
func (sm *StateMachine) TransitionTo(newState State) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate transition
	if !sm.isValidTransition(sm.currentState, newState) {
		err := &TransitionError{
			From:      sm.currentState,
			To:        newState,
			Message:   "invalid transition",
			Timestamp: time.Now(),
		}
		sm.errors = append(sm.errors, *err)
		return err
	}

	// Execute exit handler for current state
	if handler, exists := sm.handlers[sm.currentState]; exists {
		if err := handler.OnExit(sm.ctx); err != nil {
			sm.logger.Error("state exit handler failed",
				zap.String("from", string(sm.currentState)),
				zap.String("to", string(newState)),
				zap.Error(err))
			return err
		}
	}

	// Update state
	sm.previousState = sm.currentState
	sm.currentState = newState

	// Execute enter handler for new state
	if handler, exists := sm.handlers[newState]; exists {
		if err := handler.OnEnter(sm.ctx); err != nil {
			sm.logger.Error("state enter handler failed",
				zap.String("state", string(newState)),
				zap.Error(err))
			// Rollback on failure
			sm.currentState = sm.previousState
			return err
		}
	}

	sm.logger.Info("state transition successful",
		zap.String("from", string(sm.previousState)),
		zap.String("to", string(newState)))

	return nil
}

// Validate validates the current state
func (sm *StateMachine) Validate() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if handler, exists := sm.handlers[sm.currentState]; exists {
		return handler.Validate(sm.ctx)
	}
	return nil
}

// Stop stops the state machine
func (sm *StateMachine) Stop() {
	sm.cancel()
}

// isValidTransition checks if a transition is valid
func (sm *StateMachine) isValidTransition(from, to State) bool {
	validStates, exists := sm.transitions[from]
	if !exists {
		return false
	}
	for _, state := range validStates {
		if state == to {
			return true
		}
	}
	return false
}

// GetTransitionErrors returns all transition errors
func (sm *StateMachine) GetTransitionErrors() []TransitionError {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return append([]TransitionError{}, sm.errors...)
}
