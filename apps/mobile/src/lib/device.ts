import { Platform } from "react-native";
import * as ImagePicker from "expo-image-picker";
import * as Location from "expo-location";
import { CITIES } from "./cities";

export type Coordinates = { latitude: number; longitude: number; city: string };

/** Asks for approximate (low accuracy) location and resolves a city name when possible. */
export async function locateDevice(): Promise<Coordinates | null> {
  try {
    const perm = await Location.requestForegroundPermissionsAsync();
    if (perm.status !== "granted") return null;
    const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
    const { latitude, longitude } = pos.coords;
    let city = "";
    try {
      const places = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = places[0]?.city ?? places[0]?.subregion ?? "";
    } catch {
      // Reverse geocoding is a nicety (unsupported on web): fall back to the nearest known city.
    }
    if (!city) city = nearestCity(latitude, longitude);
    return { latitude, longitude, city };
  } catch {
    return null;
  }
}

function nearestCity(latitude: number, longitude: number): string {
  let best = "";
  let bestDistance = Infinity;
  for (const c of CITIES) {
    const d = (c.latitude - latitude) ** 2 + (c.longitude - longitude) ** 2;
    if (d < bestDistance) {
      best = c.name;
      bestDistance = d;
    }
  }
  return bestDistance < 0.5 ? best : "";
}

/** Opens the photo library and returns multipart data ready for the upload endpoints. */
export async function pickPhotoForm(): Promise<FormData | "denied" | null> {
  if (Platform.OS !== "web") {
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) return "denied";
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    quality: 0.85,
    allowsMultipleSelection: false
  });
  const asset = result.canceled ? undefined : result.assets[0];
  if (!asset) return null;
  const form = new FormData();
  const mime = asset.mimeType ?? "image/jpeg";
  if (Platform.OS === "web") {
    const blob = await (await fetch(asset.uri)).blob();
    form.append("file", blob, asset.fileName ?? "photo.jpg");
  } else {
    form.append("file", { uri: asset.uri, name: asset.fileName ?? "photo.jpg", type: mime } as unknown as Blob);
  }
  return form;
}
