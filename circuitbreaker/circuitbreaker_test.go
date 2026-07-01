package circuitbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/circuitbreaker"
)

var errService = errors.New("service error")

func TestNew_DefaultsApplied(t *testing.T) {
	cb := circuitbreaker.New(circuitbreaker.Config{Name: "test"})
	require.NotNil(t, cb)
	assert.Equal(t, circuitbreaker.StateClosed, cb.State())
}

func TestExecute_Success(t *testing.T) {
	cb := circuitbreaker.New(circuitbreaker.Config{Name: "ok"})
	err := cb.Execute(func() error { return nil })
	assert.NoError(t, err)
}

func TestExecute_PassthroughError(t *testing.T) {
	cb := circuitbreaker.New(circuitbreaker.Config{Name: "err"})
	err := cb.Execute(func() error { return errService })
	assert.ErrorIs(t, err, errService)
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := circuitbreaker.New(circuitbreaker.Config{
		Name:             "trip",
		FailureThreshold: 2,
		OpenTimeout:      500 * time.Millisecond,
		MaxRequests:      1,
	})

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return errService })
	}

	assert.Equal(t, circuitbreaker.StateOpen, cb.State())

	// Further calls return ErrCircuitOpen.
	err := cb.Execute(func() error { return nil })
	assert.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen)
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := circuitbreaker.New(circuitbreaker.Config{
		Name:             "halfopen",
		FailureThreshold: 1,
		OpenTimeout:      100 * time.Millisecond,
		MaxRequests:      1,
	})

	_ = cb.Execute(func() error { return errService })
	_ = cb.Execute(func() error { return errService })
	assert.Equal(t, circuitbreaker.StateOpen, cb.State())

	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, circuitbreaker.StateHalfOpen, cb.State())
}

func TestState_String(t *testing.T) {
	assert.Equal(t, "closed", circuitbreaker.StateClosed.String())
	assert.Equal(t, "half-open", circuitbreaker.StateHalfOpen.String())
	assert.Equal(t, "open", circuitbreaker.StateOpen.String())
}
