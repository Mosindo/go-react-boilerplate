// Package geo holds the privacy-preserving location helpers.
package geo

import "math"

const earthRadiusKm = 6371.0

// CoarsePrecision is the number of decimals kept for stored coordinates.
// 2 decimals ≈ 1.1 km of latitude: enough to compute "about N km away",
// never enough to locate a home.
const CoarsePrecision = 2

// Coarsen rounds a coordinate to CoarsePrecision decimals.
func Coarsen(v float64) float64 {
	p := math.Pow(10, CoarsePrecision)
	return math.Round(v*p) / p
}

// DistanceKm is the haversine great-circle distance.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

// DisplayKm turns a distance into the approximate value shown to users.
// Distances are bucketed (min 1 km, then whole km, then steps of 5 km beyond
// 20 km) so that repeated queries cannot be used to triangulate someone.
func DisplayKm(km float64) int {
	switch {
	case km < 1.5:
		return 1
	case km < 20:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

// BoundingBox returns lat/lon bounds containing every point within radiusKm.
// It is used as an index-friendly prefilter before the exact distance check.
func BoundingBox(lat, lon, radiusKm float64) (minLat, maxLat, minLon, maxLon float64) {
	latDelta := radiusKm / 111.0
	minLat, maxLat = math.Max(-90, lat-latDelta), math.Min(90, lat+latDelta)
	cos := math.Cos(lat * math.Pi / 180)
	if cos < 0.01 {
		return minLat, maxLat, -180, 180
	}
	lonDelta := radiusKm / (111.320 * cos)
	if lonDelta >= 180 {
		return minLat, maxLat, -180, 180
	}
	return minLat, maxLat, lon - lonDelta, lon + lonDelta
}
