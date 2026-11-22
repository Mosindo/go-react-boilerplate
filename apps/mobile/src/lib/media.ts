import { Platform } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";

const MAX_WIDTH = 1080;

/**
 * Lets the member pick a photo, crops it to the 4:5 card ratio on devices,
 * downsizes it and re-encodes it as JPEG before upload (saves bandwidth; the
 * server re-validates and re-encodes anyway, stripping metadata).
 * Returns null when the member cancels.
 */
export async function pickPhotoForUpload(): Promise<FormData | null> {
  if (Platform.OS !== "web") {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      throw new Error("Autorisez l'accès à vos photos dans les réglages pour en ajouter.");
    }
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: Platform.OS !== "web",
    aspect: [4, 5],
    quality: 1,
    exif: false
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  const context = ImageManipulator.manipulate(asset.uri);
  if (asset.width > MAX_WIDTH) {
    context.resize({ width: MAX_WIDTH });
  }
  const rendered = await context.renderAsync();
  const saved = await rendered.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });

  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(saved.uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors of this shape.
    form.append("photo", { uri: saved.uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}
