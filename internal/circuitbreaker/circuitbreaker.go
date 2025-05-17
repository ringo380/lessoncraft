package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the current state of the circuit breaker
type State int

const (
	// StateClosed means the circuit is closed and requests are allowed to pass through
	StateClosed State = iota
	// StateOpen means the circuit is open and requests are not allowed to pass through
	StateOpen
	// StateHalfOpen means the circuit is half-open and a limited number of requests are allowed to pass through
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name                     string
	state                    State
	failureThreshold         int
	resetTimeout             time.Duration
	halfOpenSuccessThreshold int
	failureCount             int
	successCount             int
	lastStateChange          time.Time
	mutex                    sync.RWMutex
	onStateChange            func(name string, from, to State)
}

// Options configures a CircuitBreaker
type Options struct {
	// Name is a descriptive name for the circuit breaker
	Name string
	// FailureThreshold is the number of consecutive failures that will trip the circuit
	FailureThreshold int
	// ResetTimeout is the time to wait before transitioning from open to half-open
	ResetTimeout time.Duration
	// HalfOpenSuccessThreshold is the number of consecutive successes needed to close the circuit
	HalfOpenSuccessThreshold int
	// OnStateChange is called when the circuit breaker changes state
	OnStateChange func(name string, from, to State)
}

// DefaultOptions returns the default options for a CircuitBreaker
func DefaultOptions() Options {
	return Options{
		Name:                     "default",
		FailureThreshold:         5,
		ResetTimeout:             10 * time.Second,
		HalfOpenSuccessThreshold: 2,
		OnStateChange:            nil,
	}
}

// NewCircuitBreaker creates a new CircuitBreaker with the given options
func NewCircuitBreaker(options Options) *CircuitBreaker {
	if options.FailureThreshold <= 0 {
		options.FailureThreshold = DefaultOptions().FailureThreshold
	}
	if options.ResetTimeout <= 0 {
		options.ResetTimeout = DefaultOptions().ResetTimeout
	}
	if options.HalfOpenSuccessThreshold <= 0 {
		options.HalfOpenSuccessThreshold = DefaultOptions().HalfOpenSuccessThreshold
	}

	return &CircuitBreaker{
		name:                     options.Name,
		state:                    StateClosed,
		failureThreshold:         options.FailureThreshold,
		resetTimeout:             options.ResetTimeout,
		halfOpenSuccessThreshold: options.HalfOpenSuccessThreshold,
		failureCount:             0,
		successCount:             0,
		lastStateChange:          time.Now(),
		onStateChange:            options.OnStateChange,
	}
}

// ErrCircuitOpen is returned when the circuit is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Execute executes the given function if the circuit is closed or half-open
// It will record the result of the function and update the circuit state accordingly
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if the circuit is open
	if !cb.AllowRequest() {
		return ErrCircuitOpen
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.RecordResult(err)

	return err
}

// AllowRequest checks if a request should be allowed to pass through
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if the reset timeout has elapsed
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			// Transition to half-open state
			cb.mutex.RUnlock()
			cb.setState(StateHalfOpen)
			cb.mutex.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		// Allow a limited number of requests in half-open state
		// In this simple implementation, we allow only one request at a time
		return true
	default:
		return true
	}
}

// RecordResult records the result of a request and updates the circuit state
func (cb *CircuitBreaker) RecordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		// Record a failure
		cb.failureCount++
		cb.successCount = 0

		// Check if we need to trip the circuit
		if cb.state == StateClosed && cb.failureCount >= cb.failureThreshold {
			cb.setState(StateOpen)
		} else if cb.state == StateHalfOpen {
			cb.setState(StateOpen)
		}
	} else {
		// Record a success
		cb.successCount++
		cb.failureCount = 0

		// Check if we need to close the circuit
		if cb.state == StateHalfOpen && cb.successCount >= cb.halfOpenSuccessThreshold {
			cb.setState(StateClosed)
		}
	}
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Reset counters
	cb.failureCount = 0
	cb.successCount = 0

	// Notify state change
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, oldState, newState)
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Reset resets the circuit breaker to its initial closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.setState(StateClosed)
}
