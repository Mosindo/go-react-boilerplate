import { Platform } from "react-native";

export type PickedImage = { uri: string; fileName?: string | null; mimeType?: string | null };

/**
 * Builds the multipart body for a photo upload. React Native accepts a {uri,name,type} descriptor;
 * the web needs a real Blob.
 */
export async function buildPhotoForm(image: PickedImage): Promise<FormData> {
  const form = new FormData();
  const type = image.mimeType === "image/png" ? "image/png" : "image/jpeg";
  const name = image.fileName ?? (type === "image/png" ? "photo.png" : "photo.jpg");
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("photo", blob, name);
  } else {
    form.append("photo", { uri: image.uri, name, type } as unknown as Blob);
  }
  return form;
}
