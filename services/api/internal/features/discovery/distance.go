package discovery

import "math"

// DistanceBucket converts an exact distance into the coarse value shown to
// other users: "less than N km" with N a multiple of 5. Exact distances are
// never exposed, which prevents trilateration of a user's position.
func DistanceBucket(km float64) int {
	if km <= 5 {
		return 5
	}
	return int(math.Ceil(km/5.0)) * 5
}
