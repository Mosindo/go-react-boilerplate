import { Platform } from "react-native";

export type LocalImage = { uri: string; mimeType?: string | null; fileName?: string | null };

/**
 * Builds the multipart body for a picked image. React Native accepts a
 * {uri,name,type} descriptor; the web needs a real Blob.
 */
export async function photoFormData(image: LocalImage): Promise<FormData> {
  const form = new FormData();
  const name = image.fileName ?? "photo.jpg";
  const type = image.mimeType ?? "image/jpeg";
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("photo", blob, name);
  } else {
    form.append("photo", { uri: image.uri, name, type } as unknown as Blob);
  }
  return form;
}
