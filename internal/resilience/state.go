package resilience

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// CircuitBreakerStates manages multiple circuit breakers with orchestration
type CircuitBreakerStates struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewCircuitBreakerStates creates a new circuit breaker state manager
func NewCircuitBreakerStates(logger *zap.Logger) *CircuitBreakerStates {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CircuitBreakerStates{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// AddBreaker adds a circuit breaker to the state manager
func (cs *CircuitBreakerStates) AddBreaker(name string, breaker *CircuitBreaker) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.breakers[name]; exists {
		return fmt.Errorf("circuit breaker %s already exists", name)
	}

	cs.breakers[name] = breaker
	cs.logger.Info("Added circuit breaker",
		zap.String("name", name))

	return nil
}

// GetBreaker retrieves a circuit breaker by name
func (cs *CircuitBreakerStates) GetBreaker(name string) (*CircuitBreaker, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	breaker, exists := cs.breakers[name]
	return breaker, exists
}

// GetAllBreakers returns all registered circuit breakers
func (cs *CircuitBreakerStates) GetAllBreakers() []*CircuitBreaker {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	breakers := make([]*CircuitBreaker, 0, len(cs.breakers))
	for _, breaker := range cs.breakers {
		breakers = append(breakers, breaker)
	}
	return breakers
}

// GetGlobalState returns the overall health state across all breakers
func (cs *CircuitBreakerStates) GetGlobalState() CircuitBreakerGlobalState {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var (
		totalOpen     int
		totalHalfOpen int
		totalClosed   int
		totalRequests uint64
		totalFailures uint64
	)

	for _, breaker := range cs.breakers {
		switch breaker.GetState() {
		case StateOpen:
			totalOpen++
		case StateHalfOpen:
			totalHalfOpen++
		case StateClosed:
			totalClosed++
		}

		stats := breaker.GetStats()
		totalRequests += stats.TotalRequests
		totalFailures += stats.TotalFailures
	}

	globalState := CircuitBreakerGlobalState{
		TotalBreakers: len(cs.breakers),
		OpenCount:     totalOpen,
		HalfOpenCount: totalHalfOpen,
		ClosedCount:   totalClosed,
		TotalRequests: totalRequests,
		TotalFailures: totalFailures,
		TimeStamp:     time.Now(),
	}

	// Determine overall health
	if totalOpen > 0 {
		globalState.OverallHealth = "critical"
	} else if totalHalfOpen > 0 {
		globalState.OverallHealth = "degraded"
	} else {
		globalState.OverallHealth = "healthy"
	}

	return globalState
}

// CircuitBreakerGlobalState represents the global state across all breakers
type CircuitBreakerGlobalState struct {
	TotalBreakers int
	OpenCount     int
	HalfOpenCount int
	ClosedCount   int
	TotalRequests uint64
	TotalFailures uint64
	OverallHealth string
	TimeStamp     time.Time
}

// String returns a string representation of the global state
func (gs CircuitBreakerGlobalState) String() string {
	return fmt.Sprintf("Breakers: %d (Open: %d, Half-Open: %d, Closed: %d) | Health: %s | Requests: %d | Failures: %d",
		gs.TotalBreakers, gs.OpenCount, gs.HalfOpenCount, gs.ClosedCount,
		gs.OverallHealth, gs.TotalRequests, gs.TotalFailures)
}

// BulkOperations provides bulk operations across all circuit breakers
type BulkOperations struct {
	States *CircuitBreakerStates
	logger *zap.Logger
}

// NewBulkOperations creates a new bulk operations handler
func NewBulkOperations(states *CircuitBreakerStates, logger *zap.Logger) *BulkOperations {
	return &BulkOperations{
		States: states,
		logger: logger,
	}
}

// OpenAll forces all circuit breakers to open state
func (bo *BulkOperations) OpenAll() {
	breakers := bo.States.GetAllBreakers()
	opened := 0

	for _, breaker := range breakers {
		if breaker.GetState() != StateOpen {
			breaker.ForceOpen()
			opened++
		}
	}

	bo.logger.Info("Bulk operation: opened circuit breakers",
		zap.Int("attempted", len(breakers)),
		zap.Int("opened", opened))
}

// CloseAll forces all circuit breakers to closed state
func (bo *BulkOperations) CloseAll() {
	breakers := bo.States.GetAllBreakers()
	closed := 0

	for _, breaker := range breakers {
		if breaker.GetState() != StateClosed {
			breaker.ForceClose()
			closed++
		}
	}

	bo.logger.Info("Bulk operation: closed circuit breakers",
		zap.Int("attempted", len(breakers)),
		zap.Int("closed", closed))
}

// ResetAll resets all circuit breakers to initial closed state
func (bo *BulkOperations) ResetAll() {
	breakers := bo.States.GetAllBreakers()

	for _, breaker := range breakers {
		breaker.Reset()
	}

	bo.logger.Info("Bulk operation: reset circuit breakers",
		zap.Int("reset", len(breakers)))
}

// CircuitBreakerEvents manages event logging for circuit breaker operations
type CircuitBreakerEvents struct {
	logger    *zap.Logger
	events    []CircuitBreakerEvent
	eventsMu  sync.RWMutex
	maxEvents int
}

// CircuitBreakerEvent represents a circuit breaker state change event
type CircuitBreakerEvent struct {
	Name         string              `json:"name"`
	OldState     CircuitBreakerState `json:"old_state"`
	NewState     CircuitBreakerState `json:"new_state"`
	FailureRate  float64             `json:"failure_rate"`
	Timestamp    time.Time           `json:"timestamp"`
	RequestCount uint64              `json:"request_count"`
}

// NewCircuitBreakerEvents creates a new event logger
func NewCircuitBreakerEvents(logger *zap.Logger, maxEvents int) *CircuitBreakerEvents {
	if maxEvents <= 0 {
		maxEvents = 1000 // Default max events
	}

	return &CircuitBreakerEvents{
		logger:    logger,
		events:    make([]CircuitBreakerEvent, 0, maxEvents),
		maxEvents: maxEvents,
	}
}

// LogEvent logs a circuit breaker state change event
func (ce *CircuitBreakerEvents) LogEvent(name string, oldState, newState CircuitBreakerState, failureRate float64, requestCount uint64) {
	event := CircuitBreakerEvent{
		Name:         name,
		OldState:     oldState,
		NewState:     newState,
		FailureRate:  failureRate,
		Timestamp:    time.Now(),
		RequestCount: requestCount,
	}

	ce.eventsMu.Lock()
	defer ce.eventsMu.Unlock()

	ce.events = append(ce.events, event)

	// Maintain max events limit (circular buffer)
	if len(ce.events) > ce.maxEvents {
		ce.events = ce.events[1:]
	}

	ce.logger.Info("Circuit breaker state change logged",
		zap.String("breaker", name),
		zap.String("from", ce.stateString(oldState)),
		zap.String("to", ce.stateString(newState)),
		zap.Float64("failure_rate", failureRate),
		zap.Uint64("request_count", requestCount))
}

// GetEvents returns recent circuit breaker events
func (ce *CircuitBreakerEvents) GetEvents(limit int) []CircuitBreakerEvent {
	ce.eventsMu.RLock()
	defer ce.eventsMu.RUnlock()

	count := len(ce.events)
	if limit <= 0 || limit > count {
		limit = count
	}

	// Return most recent events first
	result := make([]CircuitBreakerEvent, limit)
	copy(result, ce.events[count-limit:])
	return result
}

// ClearEvents clears all recorded events
func (ce *CircuitBreakerEvents) ClearEvents() {
	ce.eventsMu.Lock()
	defer ce.eventsMu.Unlock()

	ce.events = ce.events[:0]
	ce.logger.Info("Circuit breaker events cleared")
}

// stateString converts circuit breaker state to string
func (ce *CircuitBreakerEvents) stateString(state CircuitBreakerState) string {
	switch state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerMetrics provides monitoring and alerting integration
type CircuitBreakerMetrics struct {
	Breakers   *CircuitBreakerStates
	Events     *CircuitBreakerEvents
	alertCount int64
	lastCheck  time.Time
	checkMu    sync.Mutex
}

// NewCircuitBreakerMetrics creates a new metrics handler
func NewCircuitBreakerMetrics(breakers *CircuitBreakerStates, events *CircuitBreakerEvents) *CircuitBreakerMetrics {
	return &CircuitBreakerMetrics{
		Breakers: breakers,
		Events:   events,
	}
}

// CheckHealth performs health checks and returns current status
func (cm *CircuitBreakerMetrics) CheckHealth() CircuitBreakerHealthStatus {
	globalState := cm.Breakers.GetGlobalState()

	status := CircuitBreakerHealthStatus{
		GlobalState:   globalState,
		BreakerStates: make([]CircuitBreakerBreakerStatus, 0),
		Timestamp:     time.Now(),
	}

	breakers := cm.Breakers.GetAllBreakers()
	for _, breaker := range breakers {
		stats := breaker.GetStats()
		breakerStatus := CircuitBreakerBreakerStatus{
			Name:          breaker.config.Name,
			State:         breaker.GetState(),
			FailureRate:   breaker.GetFailureRate(),
			TotalRequests: stats.TotalRequests,
			TotalFailures: stats.TotalFailures,
		}
		status.BreakerStates = append(status.BreakerStates, breakerStatus)
	}

	return status
}

// CircuitBreakerHealthStatus represents current health status
type CircuitBreakerHealthStatus struct {
	GlobalState   CircuitBreakerGlobalState     `json:"global_state"`
	BreakerStates []CircuitBreakerBreakerStatus `json:"breaker_states"`
	Timestamp     time.Time                     `json:"timestamp"`
}

// CircuitBreakerBreakerStatus represents individual breaker status
type CircuitBreakerBreakerStatus struct {
	Name          string              `json:"name"`
	State         CircuitBreakerState `json:"state"`
	FailureRate   float64             `json:"failure_rate"`
	TotalRequests uint64              `json:"total_requests"`
	TotalFailures uint64              `json:"total_failures"`
}

// ShouldAlert checks if alerting conditions are met
func (cm *CircuitBreakerMetrics) ShouldAlert() bool {
	cm.checkMu.Lock()
	defer cm.checkMu.Unlock()

	now := time.Now()
	if now.Sub(cm.lastCheck) < 30*time.Second {
		// Only check every 30 seconds to avoid spam
		return false
	}

	globalState := cm.Breakers.GetGlobalState()

	// Alert if more than 20% of breakers are open
	openPercentage := float64(globalState.OpenCount) / float64(globalState.TotalBreakers)
	if openPercentage > 0.2 {
		atomic.AddInt64(&cm.alertCount, 1)
		cm.lastCheck = now
		return true
	}

	cm.lastCheck = now
	return false
}

// GetAlertCount returns the number of alerts triggered
func (cm *CircuitBreakerMetrics) GetAlertCount() int64 {
	return atomic.LoadInt64(&cm.alertCount)
}
