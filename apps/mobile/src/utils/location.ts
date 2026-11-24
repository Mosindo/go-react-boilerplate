import * as Location from "expo-location";
import { roundCoordinate } from "./validation";

export type ApproximateLocation = { latitude: number; longitude: number; city?: string };

export class LocationDeniedError extends Error {}

/**
 * Reads a coarse position (low accuracy) only when the member asks for it.
 * Coordinates are rounded to ~1 km before leaving the device.
 */
export async function getApproximateLocation(): Promise<ApproximateLocation> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== "granted") {
    throw new LocationDeniedError("location permission denied");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  const latitude = roundCoordinate(position.coords.latitude);
  const longitude = roundCoordinate(position.coords.longitude);
  let city: string | undefined;
  try {
    const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
    city = place?.city ?? place?.subregion ?? undefined;
  } catch {
    // Reverse geocoding is unavailable on some platforms (web); the city stays optional.
  }
  return { latitude, longitude, city };
}
