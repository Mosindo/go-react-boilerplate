import { Platform } from "react-native";

function normalizeBaseUrl(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }
  if (__DEV__) {
    // Emulators only. A physical phone needs EXPO_PUBLIC_API_URL set to the LAN IP.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Turns a relative signed photo path returned by the API into a loadable URL. */
export function mediaUrl(path: string): string {
  return path.startsWith("http") ? path : `${API_BASE_URL}${path}`;
}
