package testkit

import (
	"strings"
	"testing"
)

// AssertError checks that err is not nil and contains wantSubstr in its error message.
// Uses t.Helper() to report failures at the caller's line.
func AssertError(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("expected error to contain %q, got: %v", wantSubstr, err)
	}
}

// AssertNoError checks that err is nil.
// Uses t.Helper() to report failures at the caller's line.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}
