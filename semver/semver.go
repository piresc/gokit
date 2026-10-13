package semver

import (
	"errors"
	"strconv"
	"strings"
	"sync"
)

// Version represents a semantic version.
type Version struct {
	Major, Minor, Patch int
	Prerelease          string
	Build               string
}

// Parse parses a semantic version string.
func Parse(v string) (*Version, error) {
	if v == "" {
		return nil, errors.New("empty version string")
	}

	// Remove 'v' prefix if present
	v = strings.TrimPrefix(v, "v")

	// Find build metadata
	buildIdx := strings.IndexByte(v, '+')
	var build string
	if buildIdx != -1 {
		build = v[buildIdx+1:]
		v = v[:buildIdx]
	}

	// Find prerelease
	prereleaseIdx := strings.IndexByte(v, '-')
	var prerelease string
	if prereleaseIdx != -1 {
		prerelease = v[prereleaseIdx+1:]
		v = v[:prereleaseIdx]
	}

	// Parse major.minor.patch
	dot1 := strings.IndexByte(v, '.')
	if dot1 == -1 {
		return nil, errors.New("invalid version format")
	}
	dot2 := strings.IndexByte(v[dot1+1:], '.')
	if dot2 == -1 {
		return nil, errors.New("invalid version format")
	}
	dot2 += dot1 + 1

	major, err := strconv.Atoi(v[:dot1])
	if err != nil {
		return nil, errors.New("invalid major version")
	}

	minor, err := strconv.Atoi(v[dot1+1 : dot2])
	if err != nil {
		return nil, errors.New("invalid minor version")
	}

	patch, err := strconv.Atoi(v[dot2+1:])
	if err != nil {
		return nil, errors.New("invalid patch version")
	}

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Build:      build,
	}, nil
}

// MustParse parses a version string and panics on error.
func MustParse(v string) *Version {
	ver, err := Parse(v)
	if err != nil {
		panic(err)
	}
	return ver
}

// Compare compares this version with another. Returns -1, 0, or 1.
func (v *Version) Compare(other *Version) int {
	if result := compareInt(v.Major, other.Major); result != 0 {
		return result
	}
	if result := compareInt(v.Minor, other.Minor); result != 0 {
		return result
	}
	if result := compareInt(v.Patch, other.Patch); result != 0 {
		return result
	}
	return comparePrerelease(v.Prerelease, other.Prerelease)
}

// String returns the string representation of the version.
func (v *Version) String() string {
	version := strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
	if v.Prerelease != "" {
		version += "-" + v.Prerelease
	}
	if v.Build != "" {
		version += "+" + v.Build
	}
	return version
}

// versionCache is a thread-safe cache for parsed versions.
var versionCache sync.Map

// parseOrCache returns a cached *Version for v, parsing and storing it on first access.
func parseOrCache(v string) (*Version, error) {
	if cached, ok := versionCache.Load(v); ok {
		return cached.(*Version), nil
	}
	parsed, err := Parse(v)
	if err == nil {
		versionCache.Store(v, parsed)
	}
	return parsed, err
}

// Compare compares two version strings. Returns -1, 0, or 1.
func Compare(v1, v2 string) (int, error) {
	ver1, err := parseOrCache(v1)
	if err != nil {
		return 0, err
	}
	ver2, err := parseOrCache(v2)
	if err != nil {
		return 0, err
	}
	return ver1.Compare(ver2), nil
}

// LessThan returns true if v1 < v2.
func LessThan(v1, v2 string) (bool, error) {
	result, err := Compare(v1, v2)
	return result < 0, err
}

// LessThanOrEqual returns true if v1 <= v2.
func LessThanOrEqual(v1, v2 string) (bool, error) {
	result, err := Compare(v1, v2)
	return result <= 0, err
}

// GreaterThan returns true if v1 > v2.
func GreaterThan(v1, v2 string) (bool, error) {
	result, err := Compare(v1, v2)
	return result > 0, err
}

// GreaterThanOrEqual returns true if v1 >= v2.
func GreaterThanOrEqual(v1, v2 string) (bool, error) {
	result, err := Compare(v1, v2)
	return result >= 0, err
}

// Equal returns true if v1 == v2.
func Equal(v1, v2 string) (bool, error) {
	result, err := Compare(v1, v2)
	return result == 0, err
}

func compareInt(a, b int) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}

func comparePrerelease(a, b string) int {
	if a == "" && b != "" {
		return 1
	}
	if a != "" && b == "" {
		return -1
	}
	return strings.Compare(a, b)
}
