import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import type { UploadFile } from "../api/profile";
import { computeResize } from "../domain/photos";

const JPEG_QUALITY = 0.8;

/** Lets the user pick a photo, then resizes (longest side <= 1280px) and re-encodes it as JPEG before upload. */
export async function pickPhotoForUpload(): Promise<UploadFile | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsMultipleSelection: false,
    quality: 1
  });
  if (result.canceled || result.assets.length === 0) {
    return null;
  }
  const asset = result.assets[0];
  const context = ImageManipulator.ImageManipulator.manipulate(asset.uri);
  const resize = computeResize(asset.width, asset.height);
  if (resize) {
    context.resize(resize);
  }
  const image = await context.renderAsync();
  const saved = await image.saveAsync({ format: ImageManipulator.SaveFormat.JPEG, compress: JPEG_QUALITY });
  return { uri: saved.uri, name: `photo-${Date.now()}.jpg`, type: "image/jpeg" };
}
