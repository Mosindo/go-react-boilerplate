import { Platform } from "react-native";

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  if (__DEV__) {
    // Android emulators reach the host machine through 10.0.2.2. Physical
    // devices must set EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** Turns an API-relative path (e.g. a signed photo URL) into an absolute URL. */
export function absoluteUrl(path: string): string {
  if (/^https?:\/\//.test(path)) {
    return path;
  }
  return `${API_BASE_URL}${path}`;
}
