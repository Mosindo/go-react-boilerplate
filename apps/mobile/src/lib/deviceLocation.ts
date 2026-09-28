import * as Location from "expo-location";

export type DeviceLocationResult =
  | { ok: true; latitude: number; longitude: number; label: string }
  | { ok: false; reason: "denied" | "unavailable"; canAskAgain: boolean };

function labelFrom(address: Location.LocationGeocodedAddress | undefined): string {
  if (!address) {
    return "";
  }
  const place = address.city ?? address.subregion ?? address.district ?? "";
  return [place, address.region].filter((part): part is string => Boolean(part)).join(", ");
}

/** Asks for foreground permission, reads the position and derives a coarse city label (may be empty). */
export async function requestDeviceLocation(): Promise<DeviceLocationResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { ok: false, reason: "denied", canAskAgain: permission.canAskAgain };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const { latitude, longitude } = position.coords;
    let label = "";
    try {
      const addresses = await Location.reverseGeocodeAsync({ latitude, longitude });
      label = labelFrom(addresses[0]);
    } catch {
      // Reverse geocoding is best effort; the user can type a city name instead.
    }
    return { ok: true, latitude, longitude, label };
  } catch {
    return { ok: false, reason: "unavailable", canAskAgain: true };
  }
}
