import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";

/**
 * Lets the member pick a photo, then downsizes and re-encodes it on device
 * (max 1440 px, JPEG 80 %) so uploads stay small on mobile networks. The
 * server validates and re-encodes again; this step is only for bandwidth.
 */
export async function pickAndPreparePhoto(): Promise<string | null> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    throw new Error("permission");
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
  const longest = Math.max(asset.width ?? 0, asset.height ?? 0);
  const actions: ImageManipulator.Action[] =
    longest > 1440 ? [{ resize: (asset.width ?? 0) >= (asset.height ?? 0) ? { width: 1440 } : { height: 1440 } }] : [];
  const prepared = await ImageManipulator.manipulateAsync(asset.uri, actions, {
    compress: 0.8,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return prepared.uri;
}
