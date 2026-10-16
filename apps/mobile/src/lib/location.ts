import * as Location from "expo-location";

export type DeviceLocation = { latitude: number; longitude: number; city: string };

export type LocationResult = { status: "ok"; location: DeviceLocation } | { status: "denied" } | { status: "unavailable" };

// ~1 km grid: matching never needs more, and the server rounds again.
const round = (value: number) => Math.round(value * 100) / 100;

/** Asks for foreground location once, only when the user chooses to share it. */
export async function readDeviceLocation(): Promise<LocationResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const latitude = round(position.coords.latitude);
    const longitude = round(position.coords.longitude);
    let city = "";
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = place?.city ?? place?.subregion ?? place?.region ?? "";
    } catch {
      // The city label is optional; coordinates are what matching uses.
    }
    return { status: "ok", location: { latitude, longitude, city } };
  } catch {
    return { status: "unavailable" };
  }
}
