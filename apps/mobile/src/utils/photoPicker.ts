import { Platform } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import type { UploadablePhoto } from "../api/endpoints";

const MAX_SIDE = 1440;

export type PickResult = { photo: UploadablePhoto } | { error: string } | null;

/**
 * Lets the user pick a photo, crops it to portrait and compresses it on the
 * device before upload (smaller, faster uploads; the server re-validates and
 * re-encodes anyway).
 */
export async function pickPhoto(): Promise<PickResult> {
  if (Platform.OS !== "web") {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      return { error: "Autorise l'accès à tes photos dans les réglages pour en ajouter." };
    }
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: Platform.OS !== "web",
    aspect: [3, 4],
    quality: 1,
    exif: false
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];

  try {
    const context = ImageManipulator.manipulate(asset.uri);
    const longest = Math.max(asset.width || 0, asset.height || 0);
    if (longest > MAX_SIDE) {
      context.resize(asset.width >= asset.height ? { width: MAX_SIDE } : { height: MAX_SIDE });
    }
    const rendered = await context.renderAsync();
    const saved = await rendered.saveAsync({ compress: 0.8, format: SaveFormat.JPEG });
    const photo: UploadablePhoto = { uri: saved.uri, mimeType: "image/jpeg", fileName: "photo.jpg" };
    if (Platform.OS === "web") {
      photo.webFile = await (await fetch(saved.uri)).blob();
    }
    return { photo };
  } catch {
    return { error: "Impossible de préparer cette photo. Essaie une autre image." };
  }
}
