import * as Location from "expo-location";
import { profileApi } from "../api/endpoints";

export type LocationOutcome = "saved" | "denied" | "unavailable";

/**
 * Sends one approximate position to the API (the server rounds it to ~1 km and only ever
 * shows other users a distance in 5 km steps). Nothing is tracked in the background.
 */
export async function shareApproximateLocation(): Promise<LocationOutcome> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) return "denied";
  try {
    const position = await Location.getCurrentPositionAsync({
      accuracy: Location.Accuracy.Balanced,
    });
    await profileApi.setLocation(position.coords.latitude, position.coords.longitude);
    return "saved";
  } catch {
    return "unavailable";
  }
}
