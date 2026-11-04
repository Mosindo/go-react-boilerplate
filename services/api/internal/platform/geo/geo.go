// Package geo holds the location privacy rules of the application.
package geo

import "math"

// CoarseDecimals is the number of decimals kept for stored coordinates (2 ≈ 1.1 km).
const CoarseDecimals = 2

// Coarsen rounds a coordinate so the exact position is never persisted.
func Coarsen(v float64) float64 {
	p := math.Pow(10, CoarseDecimals)
	return math.Round(v*p) / p
}

// ValidLatLng reports whether the pair is a legal coordinate.
func ValidLatLng(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// DistanceKm is the great-circle distance between two points.
func DistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := rad(lat2 - lat1)
	dLng := rad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKm * 2 * math.Asin(math.Min(1, math.Sqrt(a)))
}

// ApproxKm turns a distance into the coarse value shown to other users:
// buckets of 5 km (minimum 5), so nobody can be triangulated.
func ApproxKm(km float64) int {
	if km < 5 {
		return 5
	}
	return int(math.Round(km/5)) * 5
}
