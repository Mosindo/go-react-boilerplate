import { apiRequest, UPLOAD_TIMEOUT_MS } from "./client";
import type { ListResponse, Photo } from "./types";

export type UploadableImage = {
  uri: string;
  mimeType?: string | null;
  fileName?: string | null;
};

/** React Native's FormData accepts `{ uri, name, type }` objects in place of Blobs. */
export function buildPhotoForm(image: UploadableImage): FormData {
  const type = image.mimeType === "image/png" ? "image/png" : "image/jpeg";
  const extension = type === "image/png" ? "png" : "jpg";
  const form = new FormData();
  form.append("file", {
    uri: image.uri,
    name: image.fileName || `photo.${extension}`,
    type
  } as unknown as Blob);
  return form;
}

export async function listPhotos(): Promise<Photo[]> {
  return (await apiRequest<ListResponse<Photo>>("/photos")).items;
}

export function uploadPhoto(image: UploadableImage): Promise<Photo> {
  return apiRequest<Photo>("/photos", {
    method: "POST",
    body: buildPhotoForm(image),
    timeoutMs: UPLOAD_TIMEOUT_MS
  });
}

export function replacePhoto(photoId: string, image: UploadableImage): Promise<Photo> {
  return apiRequest<Photo>(`/photos/${encodeURIComponent(photoId)}`, {
    method: "PUT",
    body: buildPhotoForm(image),
    timeoutMs: UPLOAD_TIMEOUT_MS
  });
}

export async function reorderPhotos(photoIds: string[]): Promise<Photo[]> {
  return (await apiRequest<ListResponse<Photo>>("/photos/order", { method: "PUT", body: { photoIds } }))
    .items;
}

export function deletePhoto(photoId: string): Promise<void> {
  return apiRequest<void>(`/photos/${encodeURIComponent(photoId)}`, { method: "DELETE" });
}
