package idempotency_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/idempotency"
)

var errNotFound = errors.New("not found")
var errDB = errors.New("db error")

type record struct{ ID string }

func TestCheck_Found(t *testing.T) {
	lookup := func(_ context.Context, key string) (*record, error) {
		return &record{ID: key}, nil
	}

	result, err := idempotency.Check(context.Background(), "key1", lookup, errNotFound, "create")
	require.NoError(t, err)
	assert.True(t, result.Found)
	assert.Equal(t, "key1", result.Record.ID)
}

func TestCheck_NotFound(t *testing.T) {
	lookup := func(_ context.Context, _ string) (*record, error) {
		return nil, errNotFound
	}

	result, err := idempotency.Check(context.Background(), "key1", lookup, errNotFound, "create")
	require.NoError(t, err)
	assert.False(t, result.Found)
	assert.Nil(t, result.Record)
}

func TestCheck_UnexpectedError(t *testing.T) {
	lookup := func(_ context.Context, _ string) (*record, error) {
		return nil, errDB
	}

	result, err := idempotency.Check(context.Background(), "key1", lookup, errNotFound, "create")
	require.Error(t, err)
	assert.ErrorIs(t, err, errDB)
	assert.False(t, result.Found)
}

func TestCheck_NilRecordNoError(t *testing.T) {
	lookup := func(_ context.Context, _ string) (*record, error) {
		return nil, nil
	}

	result, err := idempotency.Check(context.Background(), "key1", lookup, errNotFound, "create")
	require.NoError(t, err)
	assert.False(t, result.Found)
}
