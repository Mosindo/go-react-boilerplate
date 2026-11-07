package profiles

import (
	"math"
	"time"
)

// RoundCoordinate keeps ~1.1 km precision (2 decimals). We deliberately never store exact positions.
func RoundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

// DistanceKm is the haversine great-circle distance.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 12742 * math.Asin(math.Min(1, math.Sqrt(a)))
}

// ApproxDistance coarsens a distance for display: 1 km floor, whole km under 10 km, then steps of 5 km.
func ApproxDistance(km float64) int {
	switch {
	case km <= 1:
		return 1
	case km < 10:
		return int(math.Ceil(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

// AgeOn returns completed years between birth and now.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}
