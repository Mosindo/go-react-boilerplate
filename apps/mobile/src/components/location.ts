import * as Location from "expo-location";
import { Platform } from "react-native";

export type CoarseLocation = { latitude: number; longitude: number; city?: string };

/**
 * Asks for foreground permission only when the user taps the button, reads a
 * low-accuracy fix and rounds it on-device before it ever leaves the phone
 * (the server rounds again). Never tracks in the background.
 */
export async function getCoarseLocation(): Promise<CoarseLocation | { error: string }> {
  try {
    const { status } = await Location.requestForegroundPermissionsAsync();
    if (status !== "granted") {
      return { error: "Localisation refusée. Vous pouvez indiquer votre ville à la place." };
    }
    const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
    const latitude = Math.round(pos.coords.latitude * 100) / 100;
    const longitude = Math.round(pos.coords.longitude * 100) / 100;
    let city: string | undefined;
    if (Platform.OS !== "web") {
      try {
        const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
        city = place?.city ?? place?.subregion ?? undefined;
      } catch {
        city = undefined;
      }
    }
    return { latitude, longitude, city };
  } catch {
    return { error: "Impossible d'obtenir votre position pour le moment." };
  }
}
