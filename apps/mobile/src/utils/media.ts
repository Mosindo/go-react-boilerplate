import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import * as Location from "expo-location";

export type PickResult = { uri: string } | { error: string } | null;

/**
 * Lets the user pick a photo, then downsizes and recompresses it on device
 * (≤ 1600 px, JPEG 80%) to save bandwidth. The server re-validates and
 * re-encodes anyway.
 */
export async function pickPhoto(): Promise<PickResult> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { error: "Autorise l'accès à tes photos pour en ajouter." };
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 1
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  const resize =
    (asset.width ?? 0) >= (asset.height ?? 0)
      ? { width: Math.min(asset.width || 1600, 1600) }
      : { height: Math.min(asset.height || 1600, 1600) };
  const compressed = await ImageManipulator.manipulateAsync(asset.uri, [{ resize }], {
    compress: 0.8,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return { uri: compressed.uri };
}

export type LocationResult = { latitude: number; longitude: number; city?: string } | { error: string };

/**
 * Reads the current position once, only after an explicit user action.
 * Coordinates are rounded to ~1 km before leaving the device.
 */
export async function readApproximateLocation(): Promise<LocationResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { error: "Sans localisation, nous ne pouvons pas te proposer de personnes près de toi." };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const latitude = Math.round(position.coords.latitude * 100) / 100;
    const longitude = Math.round(position.coords.longitude * 100) / 100;
    let city: string | undefined;
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = place?.city ?? place?.subregion ?? undefined;
    } catch {
      // Reverse geocoding is optional (unavailable on some platforms).
    }
    return { latitude, longitude, city };
  } catch {
    return { error: "Position introuvable. Réessaie dans un endroit plus dégagé." };
  }
}
