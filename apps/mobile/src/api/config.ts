import { Platform } from "react-native";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Android emulators reach the host machine through 10.0.2.2.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Turns API-relative media paths (signed URLs) into absolute URLs. */
export function mediaUrl(path: string | null | undefined): string | undefined {
  if (!path) return undefined;
  return path.startsWith("http") ? path : `${API_BASE_URL}${path}`;
}

export function websocketUrl(path: string): string {
  return API_BASE_URL.replace(/^http/, "ws") + path;
}
