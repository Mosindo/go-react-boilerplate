import * as Location from "expo-location";

export type CoarsePosition = { latitude: number; longitude: number; city: string };

/**
 * Asks for foreground permission and returns the current position plus a city
 * label. The server rounds coordinates to ~1 km before storing them.
 */
export async function readCoarsePosition(): Promise<CoarsePosition> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== "granted") {
    throw new Error("Autorisez la localisation pour trouver des personnes près de chez vous.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  let city = "";
  try {
    const [place] = await Location.reverseGeocodeAsync(position.coords);
    city = place?.city ?? place?.subregion ?? place?.region ?? "";
  } catch {
    // Reverse geocoding is optional (unavailable on web); the city stays empty.
  }
  return { latitude: position.coords.latitude, longitude: position.coords.longitude, city };
}
