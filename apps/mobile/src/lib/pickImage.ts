import * as ImagePicker from "expo-image-picker";
import type { PickedImage } from "../api";

/** Opens the system gallery and returns the chosen picture (null if cancelled). */
export async function pickImage(): Promise<PickedImage | null> {
  const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!perm.granted && perm.canAskAgain === false) {
    throw new Error("L'accès aux photos est désactivé. Autorisez-le dans les réglages de votre téléphone.");
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.85
  });
  if (result.canceled || !result.assets[0]) return null;
  const asset = result.assets[0];
  return { uri: asset.uri, fileName: asset.fileName, mimeType: asset.mimeType, file: asset.file ?? null };
}
