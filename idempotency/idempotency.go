// Package idempotency provides a generic idempotency check wrapper.
package idempotency

import (
	"context"
	"errors"
	"fmt"
)

// LookupFunc looks up an existing record by idempotency key.
type LookupFunc[T any] func(ctx context.Context, key string) (*T, error)

// Result holds the outcome of an idempotency check.
type Result[T any] struct {
	Found  bool
	Record *T
}

// Check looks up key via lookupFn. If found, returns Result{Found:true, Record:...}.
// If not found (errors.Is(err, errNotFound)), returns empty Result with nil error.
// Any other lookup error is wrapped with operation context and returned.
func Check[T any](ctx context.Context, key string, lookupFn LookupFunc[T], errNotFound error, operation string) (Result[T], error) {
	existing, err := lookupFn(ctx, key)
	if err == nil && existing != nil {
		return Result[T]{Found: true, Record: existing}, nil
	}
	if err != nil && !errors.Is(err, errNotFound) {
		return Result[T]{}, fmt.Errorf("%s idempotency check: %w", operation, err)
	}
	return Result[T]{}, nil
}
