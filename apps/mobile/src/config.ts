import { Platform } from "react-native";

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}

/**
 * Base URL of the API. Set EXPO_PUBLIC_API_URL (e.g. http://192.168.1.20:18080).
 * Physical phones can never reach "localhost": use the LAN IP of the machine
 * running the API. Emulators have a development fallback below.
 */
export function resolveApiBaseUrl(
  explicit: string | undefined = process.env.EXPO_PUBLIC_API_URL,
  platform: string = Platform.OS,
  isDev: boolean = __DEV__
): string {
  const trimmed = explicit?.trim();
  if (trimmed) {
    return stripTrailingSlash(trimmed);
  }
  if (isDev) {
    return platform === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** WebSocket endpoint derived from the API base URL. */
export function toWebSocketUrl(baseUrl: string): string {
  return baseUrl.replace(/^http/i, "ws") + "/ws";
}
