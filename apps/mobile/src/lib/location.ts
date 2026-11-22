import { Platform } from "react-native";
import * as Location from "expo-location";

export type ApproximatePosition = {
  latitude: number;
  longitude: number;
  city: string;
};

/**
 * Reads the device position once, with coarse accuracy: the app only needs a
 * neighbourhood-level location, which the server further rounds to ~1 km.
 */
export async function getApproximatePosition(): Promise<ApproximatePosition> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Sans accès à la localisation, nous ne pouvons pas vous proposer de profils proches.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  const { latitude, longitude } = position.coords;
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
