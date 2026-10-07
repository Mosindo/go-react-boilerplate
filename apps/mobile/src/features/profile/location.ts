import * as Location from "expo-location";

export type LocationResult =
  | { ok: true; latitude: number; longitude: number; city: string | null }
  | { ok: false; reason: "denied" | "blocked" | "unavailable" };

/**
 * Asks for foreground permission and reads an approximate position. Never throws: callers show
 * the manual-city fallback for any `ok: false`. `blocked` means the OS will not prompt again
 * (the user must enable it in system settings).
 */
export async function requestDeviceLocation(): Promise<LocationResult> {
  try {
    const permission = await Location.requestForegroundPermissionsAsync();
    if (permission.status !== Location.PermissionStatus.GRANTED) {
      return { ok: false, reason: permission.canAskAgain ? "denied" : "blocked" };
    }
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const { latitude, longitude } = position.coords;
    return { ok: true, latitude, longitude, city: await reverseCity(latitude, longitude) };
  } catch {
    return { ok: false, reason: "unavailable" };
  }
}

async function reverseCity(latitude: number, longitude: number): Promise<string | null> {
  try {
    const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
    return place?.city ?? place?.subregion ?? place?.region ?? null;
  } catch {
    return null;
  }
}

/** Resolves a typed city name to coordinates using the platform geocoder (best effort). */
export async function geocodeCity(city: string): Promise<{ latitude: number; longitude: number } | null> {
  try {
    const [match] = await Location.geocodeAsync(city);
    return match ? { latitude: match.latitude, longitude: match.longitude } : null;
  } catch {
    return null;
  }
}
