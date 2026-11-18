import { Platform } from "react-native";
import * as Location from "expo-location";

export type LocationResult =
  { status: "ok"; latitude: number; longitude: number; city?: string } | { status: "denied" } | { status: "unavailable" };

/**
 * Reads the current position once, at "balanced" accuracy. Coordinates are
 * rounded to ~1 km on device as well as on the server; only the city name is
 * kept for display.
 */
export async function getApproximateLocation(): Promise<LocationResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const latitude = Math.round(position.coords.latitude * 100) / 100;
    const longitude = Math.round(position.coords.longitude * 100) / 100;
    let city: string | undefined;
    if (Platform.OS !== "web") {
      try {
        const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
        city = place?.city ?? place?.subregion ?? undefined;
      } catch {
        city = undefined;
      }
    }
    return { status: "ok", latitude, longitude, city };
  } catch {
    return { status: "unavailable" };
  }
}
