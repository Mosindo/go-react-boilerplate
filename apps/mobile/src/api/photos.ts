import { Platform } from "react-native";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { Photo } from "./types";

export type LocalImage = { uri: string; fileName?: string | null; mimeType?: string | null };

async function toForm(image: LocalImage): Promise<FormData> {
  const form = new FormData();
  const name = image.fileName ?? "photo.jpg";
  if (Platform.OS === "web") {
    // Browsers need a real Blob; the picker hands back a blob:/data: URI.
    const blob = await (await fetch(image.uri)).blob();
    form.append("file", blob, name);
  } else {
    // React Native streams the file described by this {uri,name,type} triple.
    form.append("file", { uri: image.uri, name, type: image.mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

const UPLOAD_TIMEOUT_MS = 60_000;

export async function uploadPhoto(image: LocalImage): Promise<Photo> {
  return apiRequest<Photo>(endpoints.photos.list, { method: "POST", body: await toForm(image), timeoutMs: UPLOAD_TIMEOUT_MS, silent: true });
}

export async function replacePhoto(photoId: string, image: LocalImage): Promise<Photo> {
  return apiRequest<Photo>(endpoints.photos.item(photoId), {
    method: "PUT",
    body: await toForm(image),
    timeoutMs: UPLOAD_TIMEOUT_MS,
    silent: true
  });
}

export async function deletePhoto(photoId: string): Promise<void> {
  await apiRequest(endpoints.photos.item(photoId), { method: "DELETE", silent: true });
}

export async function reorderPhotos(ids: string[]): Promise<Photo[]> {
  const response = await apiRequest<{ photos: Photo[] }>(endpoints.photos.order, {
    method: "PUT",
    body: JSON.stringify({ ids }),
    silent: true
  });
  return response.photos;
}
