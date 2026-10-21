// Package geo holds the small amount of geographic maths the product needs.
package geo

import "math"

const earthRadiusKm = 6371.0

// Round2 rounds a coordinate to 2 decimals (~1.1 km), the precision at which locations are stored.
func Round2(v float64) float64 { return math.Round(v*100) / 100 }

// DistanceKm is the great-circle (haversine) distance.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// Display coarsens a distance before it leaves the server: whole kilometres under 50 km
// (never less than 1), steps of 5 km beyond. Exact distances are never exposed.
func Display(km float64) int {
	if km <= 1 {
		return 1
	}
	if km < 50 {
		return int(math.Ceil(km))
	}
	return int(math.Round(km/5) * 5)
}

// SQLDistance is the haversine expression for two lat/lng column pairs, in kilometres.
func SQLDistance(lat1, lon1, lat2, lon2 string) string {
	return "(2 * 6371.0 * asin(LEAST(1.0, sqrt(" +
		"power(sin(radians(" + lat2 + " - " + lat1 + ") / 2), 2) + " +
		"cos(radians(" + lat1 + ")) * cos(radians(" + lat2 + ")) * power(sin(radians(" + lon2 + " - " + lon1 + ") / 2), 2)))))"
}
