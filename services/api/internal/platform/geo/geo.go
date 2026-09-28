// Package geo holds pure location helpers. Precision policy: coordinates are
// rounded to 2 decimals (~1 km) before they are stored, and only a bucketed
// distance is ever exposed to other users.
package geo

import (
	"errors"
	"math"
)

const (
	EarthRadiusKm = 6371.0
	BucketStepKm  = 5
)

// Round2 rounds a coordinate to 2 decimals.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ValidateCoordinates checks ranges and finiteness.
func ValidateCoordinates(lat, lon float64) error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) {
		return errors.New("coordinates must be finite numbers")
	}
	if lat < -90 || lat > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if lon < -180 || lon > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

// HaversineKm is the great-circle distance in kilometres.
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	if a > 1 {
		a = 1
	}
	return 2 * EarthRadiusKm * math.Asin(math.Sqrt(a))
}

// BucketKm rounds a distance up to the next 5 km step, with a minimum of 5.
func BucketKm(d float64) int {
	if math.IsNaN(d) || d < 0 {
		return BucketStepKm
	}
	b := int(math.Ceil(d/BucketStepKm)) * BucketStepKm
	if b < BucketStepKm {
		b = BucketStepKm
	}
	return b
}

// BoundingBox returns a lat/lon box that contains every point within radiusKm
// of the centre. useLon is false when the box would wrap the antimeridian or
// reach a pole, in which case only the latitude bounds are valid.
func BoundingBox(lat, lon, radiusKm float64) (minLat, maxLat, minLon, maxLon float64, useLon bool) {
	dLat := radiusKm / 111.0 * 1.01
	minLat, maxLat = lat-dLat, lat+dLat
	if minLat < -90 {
		minLat = -90
	}
	if maxLat > 90 {
		maxLat = 90
	}
	cos := math.Cos(lat * math.Pi / 180)
	if cos < 0.01 || maxLat >= 90 || minLat <= -90 {
		return minLat, maxLat, -180, 180, false
	}
	dLon := radiusKm / (111.0 * cos) * 1.01
	minLon, maxLon = lon-dLon, lon+dLon
	if minLon < -180 || maxLon > 180 || dLon >= 180 {
		return minLat, maxLat, -180, 180, false
	}
	return minLat, maxLat, minLon, maxLon, true
}
