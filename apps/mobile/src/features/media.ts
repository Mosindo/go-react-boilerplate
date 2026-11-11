import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import * as Location from "expo-location";
import type { LocalImage } from "../api/endpoints";

const MAX_EDGE = 1440;

/**
 * Lets the user pick a photo, then downsizes and re-encodes it as JPEG before
 * upload (saves bandwidth and bakes the EXIF orientation into the pixels).
 * Returns null when the user cancels.
 */
export async function pickPhoto(): Promise<LocalImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    throw new Error("L'accès à tes photos est nécessaire pour en ajouter.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 1
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  const context = ImageManipulator.manipulate(asset.uri);
  if (asset.width > MAX_EDGE || asset.height > MAX_EDGE) {
    context.resize(asset.width >= asset.height ? { width: MAX_EDGE } : { height: MAX_EDGE });
  }
  const rendered = await context.renderAsync();
  const saved = await rendered.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
  return { uri: saved.uri, mimeType: "image/jpeg", fileName: "photo.jpg" };
}

/** Rounds to ~1 km before anything leaves the device. */
function coarse(v: number): number {
  return Math.round(v * 100) / 100;
}

/**
 * Reads an approximate position once, only when the user asks for it.
 * The app never tracks location in the background.
 */
export async function getApproximatePosition(): Promise<{ latitude: number; longitude: number }> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Autorise la localisation pour voir des profils près de toi, ou continue sans.");
  }
  const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  return { latitude: coarse(position.coords.latitude), longitude: coarse(position.coords.longitude) };
}
