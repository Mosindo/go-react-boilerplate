import * as ImagePicker from "expo-image-picker";

export type PickedPhoto = { uri: string; mimeType?: string | null; fileName?: string | null };

/** Opens the system gallery. Returns null when the user cancels. */
export async function pickPhoto(): Promise<PickedPhoto | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [4, 5],
    quality: 0.85,
    exif: false
  });
  if (result.canceled || result.assets.length === 0) return null;
  const asset = result.assets[0];
  if (!asset) return null;
  return { uri: asset.uri, mimeType: asset.mimeType, fileName: asset.fileName };
}
