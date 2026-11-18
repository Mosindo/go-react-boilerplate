// Package people holds the privacy-sensitive helpers shared by every feature
// that shows a member to someone else: age computation and approximate
// location handling.
package people

import (
	"math"
	"time"
)

const (
	MinimumAge = 18
	MaximumAge = 99

	// CoordinatePrecision rounds stored coordinates to 2 decimals (~1.1 km),
	// so the database never holds a member's precise position.
	CoordinatePrecision = 100.0
)

// Age returns the age in full years at the given instant.
func Age(birthdate, now time.Time) int {
	by, bm, bd := birthdate.Date()
	ny, nm, nd := now.Date()
	age := ny - by
	if nm < bm || (nm == bm && nd < bd) {
		age--
	}
	return age
}

// RoundCoordinate degrades a coordinate to the stored precision.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*CoordinatePrecision) / CoordinatePrecision
}

// DisplayDistanceKm turns a raw distance into what members are allowed to see:
// at least 1 km, rounded to the km under 10 km and to 5 km above, which keeps
// trilateration from recovering a precise position.
func DisplayDistanceKm(km float64) int {
	if math.IsNaN(km) || km < 1 {
		return 1
	}
	if km < 10 {
		return int(math.Round(km))
	}
	return int(math.Round(km/5) * 5)
}

// HaversineKm returns the great-circle distance between two points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}
