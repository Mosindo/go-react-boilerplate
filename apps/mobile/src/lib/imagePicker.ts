import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import type { FileUpload } from "../api/client";

export type PickResult = { status: "ok"; file: FileUpload } | { status: "cancelled" } | { status: "denied" };

const MAX_SIDE = 1600;
const QUALITY = 0.82;

/**
 * Lets the user choose a photo and shrinks it before upload: the server accepts
 * large files but mobile data is better spent on a 1600 px JPEG than a 12 MP original.
 * Re-encoding also bakes in the orientation, so photos are never sideways.
 */
export async function pickPhoto(source: "library" | "camera"): Promise<PickResult> {
  const permission =
    source === "camera" ? await ImagePicker.requestCameraPermissionsAsync() : await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }

  const options: ImagePicker.ImagePickerOptions = {
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 1
  };
  const picked = source === "camera" ? await ImagePicker.launchCameraAsync(options) : await ImagePicker.launchImageLibraryAsync(options);
  if (picked.canceled || picked.assets.length === 0) {
    return { status: "cancelled" };
  }

  const asset = picked.assets[0];
  const resize = asset.width >= asset.height ? { width: Math.min(asset.width, MAX_SIDE) } : { height: Math.min(asset.height, MAX_SIDE) };
  const result = await ImageManipulator.manipulateAsync(asset.uri, [{ resize }], {
    compress: QUALITY,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return { status: "ok", file: { uri: result.uri, name: `photo-${Date.now()}.jpg`, type: "image/jpeg" } };
}
