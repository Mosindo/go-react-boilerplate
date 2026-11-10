import * as ImagePicker from "expo-image-picker";
import { SaveFormat, manipulateAsync } from "expo-image-manipulator";

/**
 * Lets the user pick a photo, crops to portrait 4:5 when the platform
 * supports it, and compresses it client side (max 1440 px, JPEG 80%) to keep
 * uploads small. The server re-validates and re-encodes everything anyway.
 */
export async function pickAndCompressPhoto(): Promise<string | null> {
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
  if (result.canceled || !result.assets[0]) return null;
  const asset = result.assets[0];
  const longest = Math.max(asset.width || 0, asset.height || 0);
  const actions = longest > 1440 ? [asset.width >= asset.height ? { resize: { width: 1440 } } : { resize: { height: 1440 } }] : [];
  const out = await manipulateAsync(asset.uri, actions, { compress: 0.8, format: SaveFormat.JPEG });
  return out.uri;
}
