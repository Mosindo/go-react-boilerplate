package geo

import (
	"math"
	"testing"
)

func TestRound2(t *testing.T) {
	cases := map[float64]float64{
		48.856613: 48.86, 2.352222: 2.35, -0.004: 0, 179.999: 180, -33.8688: -33.87, 12.345: 12.35, 1.0: 1.0,
	}
	for in, want := range cases {
		if got := Round2(in); math.Abs(got-want) > 1e-9 {
			t.Errorf("Round2(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestBucketKm(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{{0, 5}, {0.2, 5}, {4.99, 5}, {5, 5}, {5.01, 10}, {22, 25}, {49.9, 50}, {50, 50}, {50.1, 55}, {-3, 5}, {math.NaN(), 5}}
	for _, c := range cases {
		if got := BucketKm(c.in); got != c.want {
			t.Errorf("BucketKm(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestHaversine(t *testing.T) {
	// Paris -> London is ~344 km
	d := HaversineKm(48.8566, 2.3522, 51.5074, -0.1278)
	if d < 340 || d > 348 {
		t.Errorf("Paris-London = %v", d)
	}
	if HaversineKm(10, 10, 10, 10) != 0 {
		t.Error("same point")
	}
	// antipodes stay finite
	if d := HaversineKm(0, 0, 0, 180); math.IsNaN(d) || math.Abs(d-math.Pi*EarthRadiusKm) > 1 {
		t.Errorf("antipodes: %v", d)
	}
	// symmetric
	if a, b := HaversineKm(1, 2, 30, 40), HaversineKm(30, 40, 1, 2); math.Abs(a-b) > 1e-9 {
		t.Error("not symmetric")
	}
}

func TestValidateCoordinates(t *testing.T) {
	if err := ValidateCoordinates(90, -180); err != nil {
		t.Error(err)
	}
	for _, c := range [][2]float64{{91, 0}, {-91, 0}, {0, 181}, {0, -181}, {math.NaN(), 0}, {0, math.Inf(1)}} {
		if err := ValidateCoordinates(c[0], c[1]); err == nil {
			t.Errorf("%v should be invalid", c)
		}
	}
}

func TestBoundingBoxContainsEverythingInRadius(t *testing.T) {
	centers := [][2]float64{{48.86, 2.35}, {0, 0}, {-33.87, 151.21}, {64.1, -21.9}, {35.7, 139.7}}
	for _, c := range centers {
		for _, radius := range []float64{5, 50, 500} {
			minLat, maxLat, minLon, maxLon, useLon := BoundingBox(c[0], c[1], radius)
			for bearing := 0.0; bearing < 360; bearing += 15 {
				// walk `radius` km along the bearing (small-angle approximation is fine at these radii)
				lat2 := c[0] + radius*0.999/111.0*math.Cos(bearing*math.Pi/180)
				lon2 := c[1] + radius*0.999/(111.0*math.Cos(c[0]*math.Pi/180))*math.Sin(bearing*math.Pi/180)
				if HaversineKm(c[0], c[1], lat2, lon2) > radius {
					continue
				}
				if lat2 < minLat || lat2 > maxLat {
					t.Errorf("center %v r=%v bearing %v: lat %v outside [%v,%v]", c, radius, bearing, lat2, minLat, maxLat)
				}
				if useLon && (lon2 < minLon || lon2 > maxLon) {
					t.Errorf("center %v r=%v bearing %v: lon %v outside [%v,%v]", c, radius, bearing, lon2, minLon, maxLon)
				}
			}
		}
	}
	// near the pole or the antimeridian the longitude filter is disabled
	if _, _, _, _, useLon := BoundingBox(89.9, 0, 50); useLon {
		t.Error("polar box must not use longitude")
	}
	if _, _, _, _, useLon := BoundingBox(0, 179.9, 50); useLon {
		t.Error("antimeridian box must not use longitude")
	}
}
