import { Platform } from "react-native";

function normalize(value: string): string {
  return value.replace(/\/+$/, "");
}

export function resolveApiBaseUrl(explicit: string | undefined, os: string, dev: boolean): string {
  const trimmed = explicit?.trim();
  if (trimmed) {
    return normalize(trimmed);
  }
  if (dev) {
    return os === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export function toWebSocketUrl(baseUrl: string): string {
  return `${baseUrl.replace(/^http/i, "ws")}/ws`;
}

export const API_BASE_URL = resolveApiBaseUrl(
  process.env.EXPO_PUBLIC_API_URL,
  Platform.OS,
  typeof __DEV__ !== "undefined" && __DEV__
);
export const WS_URL = toWebSocketUrl(API_BASE_URL);
