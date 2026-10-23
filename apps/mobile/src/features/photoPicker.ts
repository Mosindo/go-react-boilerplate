import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import { computeResize, isTooSmall } from "../lib/photoResize";

export type PickResult =
  | { status: "ok"; uri: string }
  | { status: "cancelled" }
  | { status: "denied" }
  | { status: "too_small" }
  | { status: "error" };

/** Opens the gallery, then resizes (longest side <= 1080) and re-encodes as JPEG q=0.8 before upload. */
export async function pickPhoto(): Promise<PickResult> {
  try {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted && permission.accessPrivileges !== "limited") {
      return { status: "denied" };
    }
    const picked = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"],
      allowsEditing: false,
      quality: 1
    });
    if (picked.canceled || !picked.assets[0]) return { status: "cancelled" };
    const asset = picked.assets[0];
    if (isTooSmall(asset.width, asset.height)) return { status: "too_small" };
    const resize = computeResize(asset.width, asset.height);
    const result = await ImageManipulator.manipulateAsync(asset.uri, resize ? [{ resize }] : [], {
      compress: 0.8,
      format: ImageManipulator.SaveFormat.JPEG
    });
    return { status: "ok", uri: result.uri };
  } catch {
    return { status: "error" };
  }
}
