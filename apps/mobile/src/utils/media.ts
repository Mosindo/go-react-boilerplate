import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import type { LocalImage } from "../api/endpoints";

const MAX_SIDE = 1440;

/**
 * Lets the member pick a photo, then downsizes and re-encodes it as JPEG
 * before upload to save bandwidth. The server re-encodes again and strips
 * all metadata, so this step is an optimisation, not a security boundary.
 */
export async function pickAndCompressPhoto(): Promise<LocalImage | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 1,
    exif: false
  });
  if (result.canceled || result.assets.length === 0) return null;
  const asset = result.assets[0];

  try {
    const context = ImageManipulator.manipulate(asset.uri);
    if (asset.width > MAX_SIDE || asset.height > MAX_SIDE) {
      context.resize(asset.width >= asset.height ? { width: MAX_SIDE } : { height: MAX_SIDE });
    }
    const rendered = await context.renderAsync();
    const saved = await rendered.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
    return { uri: saved.uri, mimeType: "image/jpeg" };
  } catch {
    // Fall back to the original file; the server will normalise it.
    return { uri: asset.uri, mimeType: asset.mimeType ?? "image/jpeg", file: asset.file };
  }
}
