import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";

export type PickResult = { status: "picked"; uri: string } | { status: "cancelled" } | { status: "denied" };

/**
 * Lets the member pick a photo, then resizes and re-encodes it on device
 * (bakes in the EXIF orientation, drops metadata, ~300 KB uploads). The server
 * re-validates and re-encodes everything anyway.
 */
export async function pickPhoto(): Promise<PickResult> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { status: "denied" };
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 1,
    exif: false
  });
  if (result.canceled || !result.assets[0]) {
    return { status: "cancelled" };
  }
  const asset = result.assets[0];
  const width = asset.width > 0 ? Math.min(asset.width, 1440) : 1440;
  const prepared = await ImageManipulator.manipulateAsync(asset.uri, [{ resize: { width } }], {
    compress: 0.82,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return { status: "picked", uri: prepared.uri };
}
