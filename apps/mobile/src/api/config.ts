import { Platform } from "react-native";

function developmentFallback(): string {
  // Android emulators reach the host machine through 10.0.2.2.
  return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) {
    return explicit.replace(/\/+$/, "");
  }
  return __DEV__ ? developmentFallback() : "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export function absoluteUrl(path: string): string {
  if (/^https?:\/\//.test(path)) {
    return path;
  }
  return `${API_BASE_URL}${path}`;
}

export function realtimeUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/realtime?ticket=${encodeURIComponent(ticket)}`;
}
