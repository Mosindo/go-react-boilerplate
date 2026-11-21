// Package geo contains the location privacy rules: coordinates are coarsened
// before storage and distances are only ever exposed rounded to whole km.
package geo

import "math"

const earthRadiusKm = 6371.0

// coarseStep is the grid size (in degrees) used to store locations.
// 0.01° is ~1.1 km in latitude: enough for matching, too coarse to locate a home.
const coarseStep = 0.01

// Valid reports whether lat/lng are real coordinates.
func Valid(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// Coarsen snaps coordinates to the storage grid.
func Coarsen(lat, lng float64) (float64, float64) {
	return roundTo(lat, coarseStep), roundTo(lng, coarseStep)
}

func roundTo(v, step float64) float64 {
	return math.Round(v/step) * step
}

// DistanceKm returns the great-circle distance between two points.
func DistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := radians(lat2 - lat1)
	dLng := radians(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(radians(lat1))*math.Cos(radians(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// DisplayDistanceKm converts a raw distance to the value shown to members:
// whole kilometres, never below 1 so that "0 km" can't reveal co-location.
func DisplayDistanceKm(km float64) int {
	rounded := int(math.Round(km))
	if rounded < 1 {
		return 1
	}
	return rounded
}

// BoundingBox returns latitude/longitude deltas covering radiusKm around lat.
// It is used as an index-friendly prefilter before the exact distance check.
func BoundingBox(lat, radiusKm float64) (latDelta, lngDelta float64) {
	latDelta = radiusKm / 111.0
	cosLat := math.Cos(radians(lat))
	if cosLat < 0.01 {
		cosLat = 0.01
	}
	lngDelta = radiusKm / (111.0 * cosLat)
	if lngDelta > 180 {
		lngDelta = 180
	}
	return latDelta, lngDelta
}

func radians(deg float64) float64 {
	return deg * math.Pi / 180
}
