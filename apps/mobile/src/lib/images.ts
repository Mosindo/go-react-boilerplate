import { Platform } from "react-native";
import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";

const MAX_EDGE = 1440;

export type PickedImage = { uri: string; width: number; height: number };

/** Opens the library, then downsizes/compresses the image before upload (saves data and time). */
export async function pickPhoto(): Promise<PickedImage | null> {
  if (Platform.OS !== "web") {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      throw new Error("Autorisez l'accès à vos photos pour en ajouter.");
    }
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 1
  });
  if (result.canceled || !result.assets[0]) return null;
  const asset = result.assets[0];
  return compress(asset.uri, asset.width, asset.height);
}

async function compress(uri: string, width: number, height: number): Promise<PickedImage> {
  const context = ImageManipulator.manipulate(uri);
  if (width > MAX_EDGE || height > MAX_EDGE) {
    if (width >= height) context.resize({ width: MAX_EDGE });
    else context.resize({ height: MAX_EDGE });
  }
  const rendered = await context.renderAsync();
  const saved = await rendered.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
  return { uri: saved.uri, width: saved.width, height: saved.height };
}

export async function photoFormData(image: PickedImage): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors.
    form.append("photo", { uri: image.uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}
