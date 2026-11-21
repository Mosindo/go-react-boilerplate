import { Platform } from "react-native";
import * as Location from "expo-location";
import { profileApi } from "../../api/endpoints";

export type LocationResult = { ok: true; city: string | null } | { ok: false; reason: "denied" | "unavailable" };

/**
 * Asks for foreground permission and sends an approximate position. Only a
 * low-accuracy fix is requested, and the server coarsens it again (~1 km)
 * before storing: the exact position never leaves the device unrounded for long.
 */
export async function shareApproximateLocation(): Promise<LocationResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { ok: false, reason: "denied" };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
    const lat = Math.round(position.coords.latitude * 100) / 100;
    const lng = Math.round(position.coords.longitude * 100) / 100;
    await profileApi.setLocation(lat, lng);
    let city: string | null = null;
    if (Platform.OS !== "web") {
      try {
        const [place] = await Location.reverseGeocodeAsync({ latitude: lat, longitude: lng });
        city = place?.city ?? place?.subregion ?? null;
      } catch {
        city = null;
      }
    }
    return { ok: true, city };
  } catch {
    return { ok: false, reason: "unavailable" };
  }
}
