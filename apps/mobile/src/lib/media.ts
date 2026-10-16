import * as ImagePicker from "expo-image-picker";
import type { UploadableImage } from "../api/platform";

export type PickResult = { status: "picked"; image: UploadableImage } | { status: "cancelled" } | { status: "denied" };

/** Opens the photo library. The server re-validates and re-encodes whatever is sent. */
export async function pickPhoto(): Promise<PickResult> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.85,
    exif: false
  });
  const asset = result.assets?.[0];
  if (result.canceled || !asset) {
    return { status: "cancelled" };
  }
  return {
    status: "picked",
    image: { uri: asset.uri, name: asset.fileName ?? "photo.jpg", mimeType: asset.mimeType ?? "image/jpeg" }
  };
}
