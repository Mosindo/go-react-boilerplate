import * as Location from "expo-location";

export type LocationResult =
  { status: "ok"; latitude: number; longitude: number; city: string | null } | { status: "denied" } | { status: "unavailable" };

/**
 * Reads a coarse position. Only low-accuracy fixes are requested, and the
 * server rounds coordinates to about 1 km before storing them.
 */
export async function getApproximateLocation(): Promise<LocationResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
    let city: string | null = null;
    try {
      const [place] = await Location.reverseGeocodeAsync(position.coords);
      city = place?.city ?? place?.subregion ?? null;
    } catch {
      city = null;
    }
    return { status: "ok", latitude: position.coords.latitude, longitude: position.coords.longitude, city };
  } catch {
    return { status: "unavailable" };
  }
}
