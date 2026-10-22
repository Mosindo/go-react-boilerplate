// Package geo holds the location rules shared by profiles and discovery:
// coordinates are stored coarsely and only approximate distances are exposed.
package geo

import "math"

const (
	earthRadiusKm = 6371.0
	// coordinateStep is the storage resolution: 0.01 degree is roughly 1 km.
	coordinateStep = 100.0
	// distanceBucketKm is the granularity of the distance shown to other users.
	distanceBucketKm = 5
)

// RoundCoordinate reduces a coordinate to the stored resolution.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*coordinateStep) / coordinateStep
}

// DistanceKm is the great-circle distance between two points.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKm * 2 * math.Asin(math.Min(1, math.Sqrt(a)))
}

// DisplayDistance converts an exact distance into the approximate value exposed
// to clients: rounded up to the next 5 km, never below 5.
func DisplayDistance(km float64) int {
	if km <= distanceBucketKm {
		return distanceBucketKm
	}
	return int(math.Ceil(km/distanceBucketKm)) * distanceBucketKm
}

// SQLDistanceKm returns a SQL expression for the great-circle distance in km
// between two (lat, lon) column pairs.
func SQLDistanceKm(lat1, lon1, lat2, lon2 string) string {
	return "(" + "6371.0 * 2 * asin(LEAST(1.0, sqrt(" +
		"power(sin(radians(" + lat2 + " - " + lat1 + ") / 2), 2) + " +
		"cos(radians(" + lat1 + ")) * cos(radians(" + lat2 + ")) * power(sin(radians(" + lon2 + " - " + lon1 + ") / 2), 2)" +
		")))" + ")"
}
