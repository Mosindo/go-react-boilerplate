import { Platform } from "react-native";
import * as Location from "expo-location";
import { roundCoordinate } from "./validation";

export type DeviceLocation = { latitude: number; longitude: number; city: string };

export class LocationDeniedError extends Error {
  constructor() {
    super("Autorisation de localisation refusée.");
    this.name = "LocationDeniedError";
  }
}

/**
 * Reads the device position once (no background tracking). Coordinates are rounded to
 * ~1 km before leaving this function, and the city name comes from on-device geocoding.
 */
export async function readDeviceLocation(): Promise<DeviceLocation> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== "granted") {
    throw new LocationDeniedError();
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  const latitude = roundCoordinate(position.coords.latitude);
  const longitude = roundCoordinate(position.coords.longitude);
  let city = "";
  if (Platform.OS !== "web") {
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = place?.city ?? place?.subregion ?? place?.region ?? "";
    } catch {
      city = "";
    }
  }
  return { latitude, longitude, city };
}
