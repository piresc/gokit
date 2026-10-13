package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistanceKm_SamePoint(t *testing.T) {
	p := Point{Lat: -6.2088, Lon: 106.8456}
	assert.InDelta(t, 0.0, DistanceKm(p, p), 0.001)
}

func TestDistanceKm_KnownDistance(t *testing.T) {
	// Jakarta to Bandung: approximately 120 km
	jakarta := Point{Lat: -6.2088, Lon: 106.8456}
	bandung := Point{Lat: -6.9175, Lon: 107.6191}
	d := DistanceKm(jakarta, bandung)
	assert.InDelta(t, 120.0, d, 5.0) // within 5 km
}

func TestDistanceM_KnownDistance(t *testing.T) {
	jakarta := Point{Lat: -6.2088, Lon: 106.8456}
	bandung := Point{Lat: -6.9175, Lon: 107.6191}
	d := DistanceM(jakarta, bandung)
	assert.InDelta(t, 120_000.0, d, 5_000.0) // within 5 km in metres
}

func TestDistanceKm_NorthPoleToEquator(t *testing.T) {
	np := Point{Lat: 90.0, Lon: 0.0}
	eq := Point{Lat: 0.0, Lon: 0.0}
	d := DistanceKm(np, eq)
	// quarter circumference ≈ 10_007 km
	assert.InDelta(t, 10_007.0, d, 10.0)
}
