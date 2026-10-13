package semver

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Valid(t *testing.T) {
	v, err := Parse("v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, 1, v.Major)
	assert.Equal(t, 2, v.Minor)
	assert.Equal(t, 3, v.Patch)
}

func TestParse_WithPrerelease(t *testing.T) {
	v, err := Parse("1.0.0-alpha.1")
	require.NoError(t, err)
	assert.Equal(t, "alpha.1", v.Prerelease)
}

func TestParse_WithBuild(t *testing.T) {
	v, err := Parse("1.0.0+build.42")
	require.NoError(t, err)
	assert.Equal(t, "build.42", v.Build)
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse("not-a-version")
	assert.Error(t, err)
}

func TestParse_Empty(t *testing.T) {
	_, err := Parse("")
	assert.Error(t, err)
}

func TestCompare_LessThan(t *testing.T) {
	r, err := Compare("1.0.0", "2.0.0")
	require.NoError(t, err)
	assert.Equal(t, -1, r)
}

func TestCompare_Equal(t *testing.T) {
	r, err := Compare("1.2.3", "1.2.3")
	require.NoError(t, err)
	assert.Equal(t, 0, r)
}

func TestCompare_GreaterThan(t *testing.T) {
	r, err := Compare("2.0.0", "1.9.9")
	require.NoError(t, err)
	assert.Equal(t, 1, r)
}

func TestLessThanOrEqual(t *testing.T) {
	ok, err := LessThanOrEqual("1.0.0", "1.0.0")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestString_RoundTrip(t *testing.T) {
	input := "1.2.3-beta+build.1"
	v := MustParse(input)
	assert.Equal(t, input, v.String())
}

func TestCache_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := LessThanOrEqual("1.2.3", "2.0.0")
			assert.NoError(t, err)
			assert.True(t, ok)
		}()
	}
	wg.Wait()
}

func TestPrereleaseOrdering(t *testing.T) {
	// release > prerelease
	r, err := Compare("1.0.0", "1.0.0-alpha")
	require.NoError(t, err)
	assert.Equal(t, 1, r)
}
