import { Platform } from "react-native";
import * as Location from "expo-location";

export type ApproxLocation = { latitude: number; longitude: number; city?: string };

/**
 * Requests a single, approximate position (only when the user asks for it).
 * Coordinates are rounded on-device to ~1 km before leaving the phone; the server
 * rounds again and never exposes them.
 */
export async function getApproximateLocation(): Promise<ApproxLocation> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Autorisez la localisation pour voir des profils près de chez vous.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  const round = (v: number) => Math.round(v * 100) / 100;
  const result: ApproxLocation = {
    latitude: round(position.coords.latitude),
    longitude: round(position.coords.longitude)
  };
  if (Platform.OS !== "web") {
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude: result.latitude, longitude: result.longitude });
      if (place?.city) result.city = place.city;
    } catch {
      // City is optional.
    }
  }
  return result;
}
