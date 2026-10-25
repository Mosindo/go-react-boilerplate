import * as Location from "expo-location";
import type { LocationInput } from "../../api";

export type LocationOutcome =
  { status: "granted"; location: LocationInput } | { status: "denied" } | { status: "unavailable" };

const MAX_CITY_LENGTH = 80;

/** Asks for the foreground permission, then resolves the current position and its city name. */
export async function detectLocation(): Promise<LocationOutcome> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== Location.PermissionStatus.GRANTED) {
    return { status: "denied" };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const { latitude, longitude } = position.coords;
    let city = "";
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = place?.city ?? place?.subregion ?? place?.region ?? "";
    } catch {
      // Reverse geocoding is optional: the coordinates alone are enough.
    }
    return { status: "granted", location: { latitude, longitude, city: city.slice(0, MAX_CITY_LENGTH) } };
  } catch {
    return { status: "unavailable" };
  }
}
