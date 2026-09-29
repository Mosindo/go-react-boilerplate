import { Platform } from "react-native";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { Photo } from "./models";

export type UploadableFile = {
  uri: string;
  name: string;
  type: string;
};

/** React Native's FormData accepts a `{uri, name, type}` descriptor in place of a Blob. */
type NativeFilePart = UploadableFile;

async function buildForm(file: UploadableFile): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(file.uri)).blob();
    form.append("file", blob, file.name);
  } else {
    const part: NativeFilePart = file;
    form.append("file", part as unknown as Blob);
  }
  return form;
}

export async function uploadPhoto(file: UploadableFile): Promise<Photo> {
  return apiRequest<Photo>(endpoints.photos.upload, {
    method: "POST",
    body: await buildForm(file)
  });
}

/** `photoIds` must be a permutation of the caller's photos; index 0 becomes the main photo. */
export async function reorderPhotos(photoIds: string[]): Promise<Photo[]> {
  const payload = await apiRequest<{ photos: Photo[] }>(endpoints.photos.order, {
    method: "PUT",
    body: JSON.stringify({ photoIds })
  });
  return payload.photos;
}

export async function deletePhoto(photoId: string): Promise<void> {
  await apiRequest<void>(endpoints.photos.remove(photoId), { method: "DELETE" });
}
