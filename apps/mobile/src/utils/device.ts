import { Platform } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { SaveFormat, manipulateAsync } from "expo-image-manipulator";
import * as Location from "expo-location";
import type { UploadablePhoto } from "../api/endpoints";

const MAX_SIDE = 1440;

/**
 * Lets the user pick a photo, then resizes and re-encodes it as JPEG on the device
 * (smaller uploads, EXIF orientation baked in). The server re-validates and strips metadata.
 */
export async function pickPhoto(): Promise<UploadablePhoto | null> {
  if (Platform.OS !== "web") {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      throw new Error("Autorise l'accès à tes photos dans les réglages pour en ajouter.");
    }
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 1,
    exif: false
  });
  if (result.canceled || result.assets.length === 0) return null;
  const asset = result.assets[0];

  const resize =
    Math.max(asset.width, asset.height) > MAX_SIDE
      ? [{ resize: asset.width >= asset.height ? { width: MAX_SIDE } : { height: MAX_SIDE } }]
      : [];
  const processed = await manipulateAsync(asset.uri, resize, { compress: 0.82, format: SaveFormat.JPEG });

  if (Platform.OS === "web") {
    const blob = await (await fetch(processed.uri)).blob();
    return { uri: processed.uri, file: blob };
  }
  return { uri: processed.uri };
}

export type ApproximatePosition = { latitude: number; longitude: number; city?: string };

/**
 * Requests a coarse ("when in use") position. Coordinates are rounded on the device to
 * ~1 km before being sent; the server rounds again and never returns them.
 */
export async function getApproximatePosition(): Promise<ApproximatePosition> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Accès à la position refusé. Tu peux choisir une ville à la place.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  const latitude = Math.round(position.coords.latitude * 100) / 100;
  const longitude = Math.round(position.coords.longitude * 100) / 100;
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
}

/** Privacy-friendly alternative to GPS: pick a city centre. */
export const CITIES: ApproximatePosition[] = [
  { city: "Paris", latitude: 48.86, longitude: 2.35 },
  { city: "Marseille", latitude: 43.3, longitude: 5.37 },
  { city: "Lyon", latitude: 45.76, longitude: 4.84 },
  { city: "Toulouse", latitude: 43.6, longitude: 1.44 },
  { city: "Nice", latitude: 43.7, longitude: 7.27 },
  { city: "Nantes", latitude: 47.22, longitude: -1.55 },
  { city: "Montpellier", latitude: 43.61, longitude: 3.88 },
  { city: "Strasbourg", latitude: 48.57, longitude: 7.75 },
  { city: "Bordeaux", latitude: 44.84, longitude: -0.58 },
  { city: "Lille", latitude: 50.63, longitude: 3.06 },
  { city: "Rennes", latitude: 48.11, longitude: -1.68 },
  { city: "Grenoble", latitude: 45.19, longitude: 5.72 },
  { city: "Bruxelles", latitude: 50.85, longitude: 4.35 },
  { city: "Genève", latitude: 46.2, longitude: 6.14 },
  { city: "Lausanne", latitude: 46.52, longitude: 6.63 },
  { city: "Montréal", latitude: 45.5, longitude: -73.57 },
  { city: "Québec", latitude: 46.81, longitude: -71.21 },
  { city: "Luxembourg", latitude: 49.61, longitude: 6.13 }
];
