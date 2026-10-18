package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTracingExcludePaths_ShouldDefaultToHealth_WhenEmpty(t *testing.T) {
	assert.Equal(t, []string{"/health"}, tracingExcludePaths(nil))
}

func TestTracingExcludePaths_ShouldReturnConfig_WhenSet(t *testing.T) {
	assert.Equal(t, []string{"/metrics"}, tracingExcludePaths([]string{"/metrics"}))
}
