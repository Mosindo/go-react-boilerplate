package profiles

import "math"

// gridStep snaps coordinates to a ~1.1 km grid (0.01° of latitude). Only the
// snapped value is stored, so the exact position of a member is never known
// to the server, and distances shown to others cannot be used to pinpoint them.
const gridStep = 0.01

const earthRadiusKm = 6371.0

func SnapCoordinate(value float64) float64 {
	return math.Round(value/gridStep) * gridStep
}

func ValidCoordinates(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKm * 2 * math.Asin(math.Min(1, math.Sqrt(a)))
}

// ApproximateKm rounds a distance for display: never below 1 km, and in
// 5 km steps beyond 10 km.
func ApproximateKm(km float64) int {
	switch {
	case km < 1.5:
		return 1
	case km < 10:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}
