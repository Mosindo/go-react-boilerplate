import { Platform } from "react-native";
import { request } from "./client";
import type { Photo } from "./types";

const UPLOAD_TIMEOUT_MS = 60_000;

async function buildPhotoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    form.append("photo", await (await fetch(uri)).blob(), "photo.jpg");
  } else {
    // React Native's FormData accepts a { uri, name, type } descriptor instead of a Blob.
    form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const listPhotos = async () => (await request<{ photos: Photo[] }>("/me/photos")).photos;

export const uploadPhoto = async (uri: string) =>
  request<Photo>("/me/photos", {
    method: "POST",
    formData: await buildPhotoForm(uri),
    timeoutMs: UPLOAD_TIMEOUT_MS
  });

export const replacePhoto = async (id: string, uri: string) =>
  request<Photo>(`/me/photos/${id}`, {
    method: "PUT",
    formData: await buildPhotoForm(uri),
    timeoutMs: UPLOAD_TIMEOUT_MS
  });

export const deletePhoto = (id: string) => request(`/me/photos/${id}`, { method: "DELETE" });

export const reorderPhotos = async (photoIds: string[]) =>
  (await request<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } }))
    .photos;
