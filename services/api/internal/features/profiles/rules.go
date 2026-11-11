package profiles

import (
	"math"
	"time"
)

const (
	MinAge           = 18
	MaxAge           = 99
	MaxInterests     = 10
	MaxPhotos        = 6
	MinDistanceKm    = 1
	MaxDistanceKm    = 500
	coordinateDigits = 100.0 // 2 decimals ≈ 1.1 km grid
)

// AgeOn returns the age in full years at the given date.
func AgeOn(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// RoundCoordinate snaps a coordinate to a ~1 km grid. Exact GPS positions are
// never stored.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*coordinateDigits) / coordinateDigits
}

// HaversineKm returns the great-circle distance between two points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// DisplayDistanceKm rounds a distance for display. Values under 2 km are
// reported as 1 km and larger ones are rounded to whole kilometres (to 5 km
// beyond 20 km) so the exact position cannot be triangulated.
func DisplayDistanceKm(km float64) int {
	switch {
	case km < 2:
		return 1
	case km < 20:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

func isGender(v string) bool {
	for _, g := range Genders {
		if g == v {
			return true
		}
	}
	return false
}
