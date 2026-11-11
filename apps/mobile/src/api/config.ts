import { Platform } from "react-native";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Emulators only. Physical phones must set EXPO_PUBLIC_API_URL to the LAN IP.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Turns a relative media path returned by the API into an absolute URL. */
export function mediaUrl(path: string): string {
  return path.startsWith("http") ? path : `${API_BASE_URL}${path}`;
}

export function websocketUrl(path: string): string {
  return API_BASE_URL.replace(/^http/, "ws") + path;
}
