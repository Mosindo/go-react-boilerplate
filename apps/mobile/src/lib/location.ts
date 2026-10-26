import * as Location from "expo-location";

export type Position = { latitude: number; longitude: number; city: string };

/** Asks for the foreground position once; the API stores only a ~1 km approximation. */
export async function getCurrentPosition(): Promise<Position> {
  const perm = await Location.requestForegroundPermissionsAsync();
  if (!perm.granted) {
    throw new Error("Autorisez la localisation pour voir des personnes proches de vous.");
  }
  const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  let city = "";
  try {
    const [place] = await Location.reverseGeocodeAsync(pos.coords);
    city = place?.city || place?.subregion || place?.region || "";
  } catch {
    // reverse geocoding is unavailable on some platforms: the user can type the city
  }
  return { latitude: pos.coords.latitude, longitude: pos.coords.longitude, city };
}

/** Resolves a typed city name to coordinates (manual alternative to GPS). */
export async function geocodeCity(city: string): Promise<Position> {
  const results = await Location.geocodeAsync(city);
  const first = results[0];
  if (!first) throw new Error("Ville introuvable. Essayez un autre nom.");
  return { latitude: first.latitude, longitude: first.longitude, city };
}
