package profiles

import "math"

const earthRadiusKm = 6371.0088

// DistanceKm is the great-circle distance between two coordinates (haversine).
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// BucketKm blurs a distance: ceil to 1 km under 10 km, ceil to 5 km from there on, minimum 1.
func BucketKm(d float64) int {
	d = math.Round(d*1e6) / 1e6 // absorb float noise so 2.0000000001 does not become 3
	if d < 10 {
		return max(1, int(math.Ceil(d)))
	}
	return int(math.Ceil(d/5) * 5)
}
