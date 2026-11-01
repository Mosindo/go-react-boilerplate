// Package geo regroupe les utilitaires de geolocalisation respectueux de la vie privee.
package geo

import "math"

// CoordPrecision: 2 decimales ~ 1.1 km. Les coordonnees stockees sont arrondies
// a cette precision: la position exacte n'est jamais conservee.
const CoordPrecision = 100.0

// RoundCoord arrondit une coordonnee a ~1 km.
func RoundCoord(v float64) float64 {
	return math.Round(v*CoordPrecision) / CoordPrecision
}

// ValidLatLng verifie des coordonnees WGS84.
func ValidLatLng(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// BucketKm convertit une distance en valeur approximative affichable
// (multiples de 5 km, minimum 5), pour eviter la triangulation.
func BucketKm(km float64) int {
	if km <= 5 {
		return 5
	}
	return int(math.Ceil(km/5.0)) * 5
}
