import * as Location from "expo-location";
import { Platform } from "react-native";

export type ResolvedLocation = { latitude: number; longitude: number; city: string };

export class LocationError extends Error {}

/**
 * Asks for foreground permission only when the member taps the button, reads
 * a balanced-accuracy fix and derives a city label. The API rounds the
 * coordinates to ~1 km before storing them.
 */
export async function locateMe(): Promise<ResolvedLocation> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    throw new LocationError("Autorisez la localisation, ou saisissez votre ville.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  let city = "";
  if (Platform.OS !== "web") {
    try {
      const [place] = await Location.reverseGeocodeAsync(position.coords);
      city = place?.city ?? place?.subregion ?? place?.region ?? "";
    } catch {
      city = "";
    }
  }
  return { latitude: position.coords.latitude, longitude: position.coords.longitude, city };
}

/** Resolves a typed city name with the platform geocoder (native only). */
export async function locateCity(query: string): Promise<ResolvedLocation> {
  if (Platform.OS === "web") {
    throw new LocationError("La recherche de ville n'est disponible que sur l'application mobile.");
  }
  const results = await Location.geocodeAsync(query);
  if (results.length === 0) {
    throw new LocationError("Ville introuvable. Vérifiez l'orthographe.");
  }
  return { latitude: results[0].latitude, longitude: results[0].longitude, city: query.trim() };
}
