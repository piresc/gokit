// Package circuitbreaker wraps sony/gobreaker to prevent cascade failures.
package circuitbreaker

import (
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

// State mirrors gobreaker states.
type State int

const (
	StateClosed   State = State(gobreaker.StateClosed)
	StateHalfOpen State = State(gobreaker.StateHalfOpen)
	StateOpen     State = State(gobreaker.StateOpen)
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit is open.
var ErrCircuitOpen = gobreaker.ErrOpenState

const (
	defaultOpenTimeout      = 30 * time.Second
	defaultInterval         = 60 * time.Second
	defaultFailureThreshold = 5
	defaultMaxRequests      = 1
)

// Config holds circuit breaker configuration.
type Config struct {
	Name             string
	FailureThreshold uint32
	OpenTimeout      time.Duration
	MaxRequests      uint32
}

// CircuitBreaker wraps gobreaker.
type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

// New creates a CircuitBreaker from cfg.
func New(cfg Config) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = defaultFailureThreshold
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = defaultOpenTimeout
	}
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = defaultMaxRequests
	}

	settings := gobreaker.Settings{
		Name:        cfg.Name,
		Interval:    defaultInterval,
		Timeout:     cfg.OpenTimeout,
		MaxRequests: cfg.MaxRequests,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.FailureThreshold
		},
	}

	return &CircuitBreaker{cb: gobreaker.NewCircuitBreaker[any](settings)}
}

// Execute calls fn through the circuit breaker.
func (c *CircuitBreaker) Execute(fn func() error) error {
	_, err := c.cb.Execute(func() (any, error) {
		return nil, fn()
	})
	if errors.Is(err, gobreaker.ErrTooManyRequests) {
		return ErrCircuitOpen
	}
	return err
}

// State returns the current circuit state.
func (c *CircuitBreaker) State() State {
	return State(c.cb.State())
}
