import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import type { LocalImage } from "../api/upload";

const MAX_EDGE = 1080;

/**
 * Lets the user pick a photo, then crops/orients and compresses it on the device so
 * uploads stay small. The server re-encodes and strips metadata regardless.
 */
export async function pickPhoto(): Promise<LocalImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    throw new Error("Autorisez l'accès à vos photos pour en ajouter.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 1,
    exif: false
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  const resize = asset.width >= asset.height ? { width: MAX_EDGE } : { height: MAX_EDGE };
  const processed = await ImageManipulator.manipulateAsync(asset.uri, [{ resize }], {
    compress: 0.82,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return { uri: processed.uri, mimeType: "image/jpeg", fileName: "photo.jpg" };
}
