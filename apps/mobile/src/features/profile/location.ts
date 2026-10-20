import * as Location from "expo-location";

export type Coordinates = { latitude: number; longitude: number };

/** Asks for foreground permission and returns a balanced-accuracy fix; the API rounds it before storing. */
export async function detectLocation(): Promise<Coordinates & { city: string | null }> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== "granted") {
    throw new Error("Location permission was not granted.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  const { latitude, longitude } = position.coords;
  let city: string | null = null;
  try {
    const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
    city = place?.city ?? place?.subregion ?? place?.region ?? null;
  } catch {
    city = null;
  }
  return { latitude, longitude, city };
}

export async function geocodeCity(city: string): Promise<Coordinates> {
  const results = await Location.geocodeAsync(city);
  const first = results[0];
  if (!first) throw new Error("We could not find that place. Try a nearby larger city.");
  return { latitude: first.latitude, longitude: first.longitude };
}
