package geo

import "math"

const earthRadiusKm = 6371.0

// Point represents a geographical point with latitude and longitude.
type Point struct {
	Lat float64
	Lon float64
}

// DistanceKm returns the great-circle distance between two points in kilometres
// using the Haversine formula.
func DistanceKm(a, b Point) float64 {
	lat1 := a.Lat * math.Pi / 180.0
	lon1 := a.Lon * math.Pi / 180.0
	lat2 := b.Lat * math.Pi / 180.0
	lon2 := b.Lon * math.Pi / 180.0

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))

	return earthRadiusKm * c
}

// DistanceM returns the great-circle distance between two points in metres.
func DistanceM(a, b Point) float64 {
	return DistanceKm(a, b) * 1000
}
