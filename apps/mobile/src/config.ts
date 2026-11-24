import { Platform } from "react-native";

function normalize(url: string): string {
  return url.trim().replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL;
  if (explicit && explicit.trim()) {
    return normalize(explicit);
  }
  if (__DEV__) {
    // Emulator defaults. Physical phones must set EXPO_PUBLIC_API_URL to the LAN IP.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export function absoluteUrl(path: string): string {
  if (/^https?:\/\//.test(path)) {
    return path;
  }
  return `${API_BASE_URL}${path}`;
}

export function websocketUrl(path: string): string {
  return absoluteUrl(path).replace(/^http/, "ws");
}
