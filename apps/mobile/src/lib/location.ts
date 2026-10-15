import * as Location from "expo-location";

export type PositionResult =
  { status: "ok"; latitude: number; longitude: number; city?: string } | { status: "denied" } | { status: "unavailable" };

/**
 * Reads the device position once, only when the user asks for it. A low
 * accuracy fix is enough: the API rounds it to about 1 km before storing it.
 */
export async function readApproximatePosition(): Promise<PositionResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
    const { latitude, longitude } = position.coords;
    let city: string | undefined;
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = place?.city ?? place?.subregion ?? place?.region ?? undefined;
    } catch {
      // The city label is optional: distance filtering works with coordinates alone.
    }
    return { status: "ok", latitude, longitude, city };
  } catch {
    return { status: "unavailable" };
  }
}
