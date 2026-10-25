import { http } from "./client";
import type { UploadFile } from "./http";
import type { Photo } from "./types";

export const photosApi = {
  upload: (file: UploadFile) => http.upload<Photo>("/photos", "POST", file),
  replace: (photoId: string, file: UploadFile) =>
    http.upload<Photo>(`/photos/${encodeURIComponent(photoId)}`, "PUT", file),
  reorder: async (photoIds: string[]) => (await http.put<{ items: Photo[] }>("/photos/order", { photoIds })).items,
  remove: (photoId: string) => http.delete(`/photos/${encodeURIComponent(photoId)}`)
};
