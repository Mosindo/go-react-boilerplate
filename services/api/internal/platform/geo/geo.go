// Package geo holds the privacy-preserving location helpers used by profiles
// and discovery. Exact coordinates never leave the device unrounded and never
// leave the server at all: clients only ever see an approximate distance.
package geo

import "math"

const earthRadiusKm = 6371.0

// gridStep is the rounding step applied before persisting coordinates (~1.1 km of latitude).
const gridStep = 0.01

// Round snaps a coordinate to a ~1 km grid so stored positions are approximate.
func Round(value float64) float64 {
	return math.Round(value/gridStep) * gridStep
}

// ValidCoordinates reports whether lat/lng are within the WGS84 range.
func ValidCoordinates(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// DistanceKm returns the great-circle distance between two points.
func DistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// ApproximateKm turns a precise distance into a coarse, display-safe value so that
// repeated queries cannot be used to triangulate someone's position.
func ApproximateKm(distance float64) int {
	switch {
	case distance < 2:
		return 1
	case distance < 10:
		return int(math.Round(distance))
	case distance < 50:
		return int(math.Round(distance/5) * 5)
	default:
		return int(math.Round(distance/10) * 10)
	}
}

// BoundingBox returns the lat/lng window that contains every point within radiusKm
// of the origin. It is used as an index-friendly prefilter before the exact check.
func BoundingBox(lat, lng, radiusKm float64) (minLat, maxLat, minLng, maxLng float64) {
	latDelta := radiusKm / 111.0
	minLat = math.Max(-90, lat-latDelta)
	maxLat = math.Min(90, lat+latDelta)
	cosLat := math.Cos(lat * math.Pi / 180)
	if cosLat < 0.01 {
		return minLat, maxLat, -180, 180
	}
	lngDelta := radiusKm / (111.0 * cosLat)
	if lngDelta >= 180 {
		return minLat, maxLat, -180, 180
	}
	return minLat, maxLat, lng - lngDelta, lng + lngDelta
}
