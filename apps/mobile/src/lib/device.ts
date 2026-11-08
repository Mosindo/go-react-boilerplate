import { Platform } from "react-native";
import * as ImagePicker from "expo-image-picker";
import * as Location from "expo-location";
import type { UploadFile } from "../api/endpoints";

export class DeviceError extends Error {}

/** Opens the photo library and returns the chosen image (JPEG/PNG), or null if cancelled. */
export async function pickPhoto(): Promise<UploadFile | null> {
  if (Platform.OS !== "web") {
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted)
      throw new DeviceError("Autorise l'accès à tes photos dans les réglages pour en ajouter.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    quality: 0.9,
    exif: false,
    allowsMultipleSelection: false,
    // Asks iOS to hand over a JPEG instead of HEIC, which the server does not accept.
    preferredAssetRepresentationMode: ImagePicker.UIImagePickerPreferredAssetRepresentationMode.Compatible
  });
  if (result.canceled || result.assets.length === 0) return null;
  const asset = result.assets[0];
  const type = asset.mimeType === "image/png" ? "image/png" : "image/jpeg";
  return { uri: asset.uri, name: asset.fileName ?? `photo.${type === "image/png" ? "png" : "jpg"}`, type };
}

export type Located = { latitude: number; longitude: number; city?: string };

/** Current position, used once to set the profile location. The server rounds it to ~1 km. */
export async function locateMe(): Promise<Located> {
  const perm = await Location.requestForegroundPermissionsAsync();
  if (!perm.granted)
    throw new DeviceError("Autorise la localisation dans les réglages, ou saisis ta ville manuellement.");
  const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  const { latitude, longitude } = pos.coords;
  let city: string | undefined;
  try {
    const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
    city = place?.city ?? place?.subregion ?? place?.region ?? undefined;
  } catch {
    city = undefined; // reverse geocoding is unavailable on some platforms (web)
  }
  return { latitude, longitude, city };
}

/** Resolves a typed city name to coordinates. */
export async function geocodeCity(name: string): Promise<Located> {
  try {
    const [hit] = await Location.geocodeAsync(name);
    if (hit) return { latitude: hit.latitude, longitude: hit.longitude, city: name.trim() };
  } catch {
    // fall through
  }
  throw new DeviceError("Ville introuvable. Essaie avec la position de l'appareil.");
}
