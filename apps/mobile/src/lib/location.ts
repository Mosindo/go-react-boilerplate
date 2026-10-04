export type Coordinates = { latitude: number; longitude: number };

export class LocationError extends Error {}

/**
 * Asks for foreground location permission (only when called) and returns one approximate fix.
 * expo-location is imported lazily so nothing touches location APIs until the user asks.
 */
export async function requestApproximateLocation(): Promise<Coordinates> {
  const Location = await import("expo-location");
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== "granted") {
    throw new LocationError("Location permission was not granted. You can still type your city instead.");
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    return { latitude: position.coords.latitude, longitude: position.coords.longitude };
  } catch {
    throw new LocationError("We couldn't read your location. You can still type your city instead.");
  }
}
