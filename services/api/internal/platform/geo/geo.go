// Package geo holds the privacy-preserving location helpers: coordinates are stored
// coarsely and distances are only ever exposed in coarse buckets.
package geo

import "math"

const (
	earthRadiusKm = 6371.0
	// DistanceBucketKm is the granularity of any distance shown to another user.
	DistanceBucketKm = 5
	coordDecimals    = 2 // ~1.1 km
)

// Haversine returns the great-circle distance in kilometres.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKm * 2 * math.Asin(math.Min(1, math.Sqrt(a)))
}

// BucketKm rounds a distance up to the next DistanceBucketKm step (minimum one bucket),
// so exact distances never leave the server.
func BucketKm(km float64) int {
	if km < 0 {
		km = 0
	}
	steps := int(math.Ceil(km / DistanceBucketKm))
	if steps < 1 {
		steps = 1
	}
	return steps * DistanceBucketKm
}

// Coarsen rounds a coordinate to the precision the application is willing to store.
func Coarsen(v float64) float64 {
	p := math.Pow(10, coordDecimals)
	return math.Round(v*p) / p
}

func ValidCoordinates(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}
