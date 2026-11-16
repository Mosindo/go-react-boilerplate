import { Platform } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import type { UploadablePhoto } from "../api/profile";

export class PhotoPermissionError extends Error {}

const MAX_SIDE = 1440;

/**
 * Lets the user pick a photo, then downsizes and re-encodes it as JPEG on the
 * device. This keeps uploads small on mobile data; the server still validates
 * and re-encodes everything (and strips metadata).
 */
export async function pickPhoto(): Promise<UploadablePhoto | null> {
  if (Platform.OS !== "web") {
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) {
      throw new PhotoPermissionError("Autorise l'accès à tes photos dans les réglages pour en ajouter.");
    }
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: Platform.OS !== "web",
    aspect: [4, 5],
    quality: 1,
    exif: false
  });
  if (result.canceled || !result.assets[0]) {
    return null;
  }
  const asset = result.assets[0];
  const ctx = ImageManipulator.manipulate(asset.uri);
  if (asset.width > MAX_SIDE || asset.height > MAX_SIDE) {
    ctx.resize(asset.width >= asset.height ? { width: MAX_SIDE } : { height: MAX_SIDE });
  }
  const rendered = await ctx.renderAsync();
  const saved = await rendered.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
  return { uri: saved.uri, mimeType: "image/jpeg" };
}
