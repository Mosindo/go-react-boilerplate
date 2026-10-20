package profiles

import "fmt"

// DistanceKmSQL returns a SQL expression for the great-circle distance in km
// between two aliases that expose latitude/longitude columns.
func DistanceKmSQL(a, b string) string {
	return fmt.Sprintf(`(2 * 6371 * asin(least(1, sqrt(
		power(sin(radians(%[1]s.latitude - %[2]s.latitude) / 2), 2) +
		cos(radians(%[2]s.latitude)) * cos(radians(%[1]s.latitude)) *
		power(sin(radians(%[1]s.longitude - %[2]s.longitude) / 2), 2)))))`, a, b)
}

var distanceBuckets = []int{1, 2, 3, 5, 10, 15, 25, 50, 100}

// BucketDistance coarsens a distance so clients never learn an exact position.
func BucketDistance(km float64) int {
	for _, bucket := range distanceBuckets {
		if km <= float64(bucket) {
			return bucket
		}
	}
	// Beyond 100 km: round up to the next 50 km.
	return (int(km/50) + 1) * 50
}
