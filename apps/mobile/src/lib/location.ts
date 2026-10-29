import * as Location from "expo-location";

export type ApproxPosition = { latitude: number; longitude: number; city: string };

/**
 * Reads a coarse position (no GPS lock needed) and the nearest city name.
 * The server rounds coordinates to ~1 km before storing them, and the exact
 * position is never shown to other users.
 */
export async function locateMe(): Promise<ApproxPosition> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== "granted") {
    throw new Error(
      "Autorisation de localisation refusée. Vous pouvez saisir votre ville manuellement."
    );
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  const { latitude, longitude } = position.coords;
  let city = "";
  try {
    const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
    city = place?.city ?? place?.subregion ?? place?.region ?? "";
  } catch {
    // The city label is optional: coordinates are enough.
  }
  return { latitude, longitude, city };
}

/** Resolves a typed city name to coordinates when the user prefers not to share their position. */
export async function geocodeCity(city: string): Promise<ApproxPosition> {
  const query = city.trim();
  if (!query) {
    throw new Error("Entrez le nom de votre ville.");
  }
  let results: Location.LocationGeocodedLocation[] = [];
  try {
    results = await Location.geocodeAsync(query);
  } catch {
    throw new Error("Impossible de trouver cette ville pour le moment.");
  }
  const first = results[0];
  if (!first) {
    throw new Error("Ville introuvable. Essayez un nom plus précis.");
  }
  return { latitude: first.latitude, longitude: first.longitude, city: query };
}
