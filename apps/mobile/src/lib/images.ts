import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import type { UploadFile } from "../api/platform";

const MAX_SIDE = 1280;

export type PickResult = { status: "ok"; file: UploadFile } | { status: "cancelled" } | { status: "denied" };

/**
 * Lets the user choose a picture and prepares it for upload: orientation is
 * baked in, the longest side is capped and the result is JPEG-compressed. The
 * server re-validates and re-encodes everything regardless.
 */
export async function pickPhoto(): Promise<PickResult> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 1
  });
  const asset = result.canceled ? undefined : result.assets[0];
  if (!asset) {
    return { status: "cancelled" };
  }
  const longest = Math.max(asset.width, asset.height);
  const actions = longest > MAX_SIDE ? [{ resize: asset.width >= asset.height ? { width: MAX_SIDE } : { height: MAX_SIDE } }] : [];
  const processed = await ImageManipulator.manipulateAsync(asset.uri, actions, {
    compress: 0.82,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return { status: "ok", file: { uri: processed.uri, name: `photo-${Date.now()}.jpg`, type: "image/jpeg" } };
}
