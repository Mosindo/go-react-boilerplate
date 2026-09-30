// Package geo contains the privacy-preserving location helpers.
//
// Exact coordinates are never stored nor returned: incoming positions are
// snapped to a ~1 km grid before persistence, and distances shown to other
// users are rounded into coarse buckets so they cannot be used to triangulate
// someone's position.
package geo

import "math"

const earthRadiusKm = 6371.0

// CoarsenCoordinate rounds a coordinate to 2 decimals (~1.1 km at the equator).
func CoarsenCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

func ValidLatitude(v float64) bool {
	return !math.IsNaN(v) && v >= -90 && v <= 90
}

func ValidLongitude(v float64) bool {
	return !math.IsNaN(v) && v >= -180 && v <= 180
}

// HaversineKm returns the great-circle distance between two points.
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// ApproximateDistanceKm converts a raw distance into the value displayed to
// users: never below 2 km, whole kilometres up to 10 km, then steps of 5 km.
func ApproximateDistanceKm(km float64) int {
	if math.IsNaN(km) || km < 0 {
		return 0
	}
	if km <= 2 {
		return 2
	}
	if km <= 10 {
		return int(math.Round(km))
	}
	return int(math.Round(km/5) * 5)
}
