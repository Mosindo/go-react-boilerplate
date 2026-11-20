import * as ImagePicker from "expo-image-picker";
import type { LocalImage } from "../../api/endpoints";
import { showToast } from "../../ui";

/**
 * Opens the photo library with on-device cropping (3:4) and JPEG compression.
 * The server re-encodes again and strips metadata (including GPS EXIF).
 */
export async function pickPhoto(): Promise<LocalImage | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    showToast("Autorise l’accès à tes photos dans les réglages pour en ajouter.", "error");
    return null;
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 0.8,
    exif: false
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  return { uri: asset.uri, mimeType: asset.mimeType ?? "image/jpeg", fileName: asset.fileName ?? "photo.jpg" };
}
