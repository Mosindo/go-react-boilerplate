// Package geo holds privacy-preserving location helpers.
package geo

import "math"

// RoundCoord truncates a coordinate to 2 decimals (~1.1 km). Only rounded values are ever stored.
func RoundCoord(v float64) float64 { return math.Round(v*100) / 100 }

// ValidCoords reports whether lat/lon are within range.
func ValidCoords(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

// ApproxKm converts an exact distance into the coarse value shown to other users:
// 1 km steps below 10 km, 5 km steps above. Never returns less than 1.
func ApproxKm(km float64) int {
	if km <= 1 {
		return 1
	}
	if km < 10 {
		return int(math.Ceil(km))
	}
	return int(math.Ceil(km/5) * 5)
}

// HaversineKm is the great-circle distance in kilometres.
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return r * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
