package discovery

import "math"

const distanceBucketKm = 5

// BucketDistance turns an exact distance into a deliberately coarse figure
// (5, 10, 15 ... km). Combined with the ~2 km storage grid it prevents
// trilateration of another user's position.
func BucketDistance(km float64) int {
	if km <= 0 || math.IsNaN(km) {
		return distanceBucketKm
	}
	return int(math.Ceil(km/distanceBucketKm)) * distanceBucketKm
}
