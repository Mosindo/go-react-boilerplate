import * as Location from "expo-location";
import { profileApi } from "../api/profile";

export class LocationDeniedError extends Error {}

/**
 * Asks for foreground location only when the user chooses to share it, reads
 * a low-accuracy fix and sends it. The server further rounds it to ~1 km and
 * never shows it to anyone: other people only see an approximate distance.
 */
export async function shareApproximateLocation(): Promise<void> {
  const perm = await Location.requestForegroundPermissionsAsync();
  if (!perm.granted) {
    throw new LocationDeniedError("Localisation refusée. Tu peux l'activer plus tard dans les réglages.");
  }
  const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  await profileApi.updateLocation(pos.coords.latitude, pos.coords.longitude);
}
