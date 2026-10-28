import * as Location from "expo-location";

export type DeviceLocation = { latitude: number; longitude: number; city?: string };

/** Asks for foreground permission only when called (never at startup). */
export async function requestDeviceLocation(): Promise<DeviceLocation> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Location permission denied. You can type your city instead.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  const { latitude, longitude } = position.coords;
  let city: string | undefined;
  try {
    const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
    city = place?.city ?? place?.subregion ?? place?.region ?? undefined;
  } catch {
    // City lookup is optional.
  }
  return { latitude, longitude, city };
}
