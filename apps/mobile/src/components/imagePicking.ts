import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import type { LocalImage } from "../api/profile";

const MAX_SIDE = 1440;

/**
 * Lets the user pick a photo, then resizes and re-encodes it as JPEG on the
 * device. This keeps uploads small (usually < 400 KB) and normalises the
 * orientation; the server re-encodes again and strips any metadata.
 */
export async function pickAndCompressPhoto(): Promise<LocalImage | null> {
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
  const context = ImageManipulator.manipulate(asset.uri);
  const longest = Math.max(asset.width ?? 0, asset.height ?? 0);
  if (longest > MAX_SIDE) {
    if ((asset.width ?? 0) >= (asset.height ?? 0)) {
      context.resize({ width: MAX_SIDE });
    } else {
      context.resize({ height: MAX_SIDE });
    }
  }
  const rendered = await context.renderAsync();
  const saved = await rendered.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
  return { uri: saved.uri, mimeType: "image/jpeg" };
}
