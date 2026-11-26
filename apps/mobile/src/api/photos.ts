import { Platform } from "react-native";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import { api } from "./client";
import type { Photo } from "./types";

const MAX_EDGE = 1440;

/** Downscales and re-encodes a picked image on the device to save bandwidth before upload. */
async function compress(uri: string, width: number, height: number): Promise<string> {
  const context = ImageManipulator.manipulate(uri);
  if (Math.max(width, height) > MAX_EDGE) {
    context.resize(width >= height ? { width: MAX_EDGE } : { height: MAX_EDGE });
  }
  const image = await context.renderAsync();
  const result = await image.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
  return result.uri;
}

async function buildForm(uri: string, width: number, height: number): Promise<FormData> {
  const compressed = await compress(uri, width, height).catch(() => uri);
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(compressed)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts a file descriptor object.
    form.append("photo", { uri: compressed, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export type PickedImage = { uri: string; width: number; height: number };

export async function uploadPhoto(image: PickedImage) {
  const form = await buildForm(image.uri, image.width, image.height);
  const res = await api<{ photos: Photo[] }>("/photos", { method: "POST", body: form });
  return res.photos;
}

export async function replacePhoto(photoId: string, image: PickedImage) {
  const form = await buildForm(image.uri, image.width, image.height);
  const res = await api<{ photos: Photo[] }>(`/photos/${photoId}`, { method: "PUT", body: form });
  return res.photos;
}

export async function deletePhoto(photoId: string) {
  const res = await api<{ photos: Photo[] }>(`/photos/${photoId}`, { method: "DELETE" });
  return res.photos;
}

export async function reorderPhotos(photoIds: string[]) {
  const res = await api<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { photoIds } });
  return res.photos;
}
