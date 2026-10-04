import { Platform } from "react-native";

function normalizeBaseUrl(value: string): string {
  return value.trim().replace(/\/+$/, "");
}

/** Pure for testability: decides the API base URL from env + platform. */
export function resolveApiBaseUrl(explicit: string | undefined, os: string, isDev: boolean): string {
  if (explicit && explicit.trim()) return normalizeBaseUrl(explicit);
  if (isDev) return os === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl(
  process.env.EXPO_PUBLIC_API_URL,
  Platform.OS,
  typeof __DEV__ !== "undefined" ? __DEV__ : false
);

/** http(s) base URL -> ws(s) URL of the realtime endpoint. */
export function toWebSocketUrl(baseUrl: string): string {
  return `${baseUrl.replace(/^http/i, "ws")}/ws`;
}
