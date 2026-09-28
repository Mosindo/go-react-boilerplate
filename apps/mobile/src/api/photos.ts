import { API_BASE_URL } from "./config";
import type { Photo } from "./types";

export type PhotoSource = { uri: string; headers: Record<string, string>; cacheKey: string };

/** Photos need the bearer token; the cache key ignores it so token rotation never refetches images. */
export function photoSource(photo: Photo, accessToken: string | null): PhotoSource {
  return {
    uri: `${API_BASE_URL}${photo.url}`,
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
    cacheKey: photo.url
  };
}
