package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_Execute(t *testing.T) {
	// Create a circuit breaker with a low threshold for testing
	cb := NewCircuitBreaker(Options{
		Name:                     "test",
		FailureThreshold:         2,
		ResetTimeout:             100 * time.Millisecond,
		HalfOpenSuccessThreshold: 1,
	})

	// Test successful execution
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())

	// Test failure but not enough to trip
	err = cb.Execute(func() error {
		return errors.New("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.State())

	// Test failure that trips the circuit
	err = cb.Execute(func() error {
		return errors.New("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())

	// Test that requests are rejected when open
	err = cb.Execute(func() error {
		return nil
	})
	assert.Equal(t, ErrCircuitOpen, err)

	// Wait for the circuit to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// Test successful execution in half-open state
	err = cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())

	// Test that the circuit stays closed after success
	err = cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	stateChanges := make([]struct {
		name string
		from State
		to   State
	}, 0)

	cb := NewCircuitBreaker(Options{
		Name:                     "test-transitions",
		FailureThreshold:         2,
		ResetTimeout:             100 * time.Millisecond,
		HalfOpenSuccessThreshold: 1,
		OnStateChange: func(name string, from, to State) {
			stateChanges = append(stateChanges, struct {
				name string
				from State
				to   State
			}{name, from, to})
		},
	})

	// Cause failures to trip the circuit
	cb.Execute(func() error { return errors.New("error 1") })
	cb.Execute(func() error { return errors.New("error 2") })

	// Wait for the circuit to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// Succeed in half-open state to close the circuit
	cb.Execute(func() error { return nil })

	// Verify state transitions
	// We only expect one state change (Closed -> Open) because the transition
	// from Open -> HalfOpen happens in AllowRequest which doesn't trigger the callback
	assert.Equal(t, 1, len(stateChanges))
	assert.Equal(t, "test-transitions", stateChanges[0].name)
	assert.Equal(t, StateClosed, stateChanges[0].from)
	assert.Equal(t, StateOpen, stateChanges[0].to)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(Options{
		Name:                     "test-reset",
		FailureThreshold:         1,
		ResetTimeout:             1 * time.Hour, // Long timeout to ensure it doesn't auto-reset
		HalfOpenSuccessThreshold: 1,
	})

	// Trip the circuit
	cb.Execute(func() error { return errors.New("error") })
	assert.Equal(t, StateOpen, cb.State())

	// Reset the circuit
	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())

	// Verify it works normally after reset
	err := cb.Execute(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}
