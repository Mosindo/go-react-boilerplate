import * as Location from "expo-location";
import { updateLocation } from "../api/profile";

export type LocationOutcome = "shared" | "denied" | "unavailable";

/**
 * Asks for foreground permission, reads a *balanced* (not precise) position once and sends it.
 * The server rounds it to about 1 km before storing it; no background tracking is ever done.
 */
export async function shareApproximateLocation(): Promise<LocationOutcome> {
  try {
    const permission = await Location.requestForegroundPermissionsAsync();
    if (!permission.granted) return "denied";
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    await updateLocation(position.coords.latitude, position.coords.longitude);
    return "shared";
  } catch {
    return "unavailable";
  }
}
