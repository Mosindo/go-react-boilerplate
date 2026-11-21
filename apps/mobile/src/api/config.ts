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
    // Android emulators reach the host through 10.0.2.2. Physical phones need
    // EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Builds an absolute URL for API-relative paths such as signed photo URLs. */
export function absoluteUrl(path: string): string {
  if (/^https?:\/\//i.test(path)) {
    return path;
  }
  return `${API_BASE_URL}${path}`;
}

export function realtimeUrl(): string {
  return `${API_BASE_URL.replace(/^http/i, "ws")}/realtime`;
}
