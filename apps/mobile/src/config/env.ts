import Constants from "expo-constants";
import { Platform } from "react-native";

const API_PORT = 18080;

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}

/** Host of the machine running the Expo dev server (a LAN IP on a physical phone). */
function devServerHost(): string | null {
  const hostUri = Constants.expoConfig?.hostUri;
  if (!hostUri) return null;
  const host = hostUri.split(":")[0];
  return host || null;
}

/**
 * Resolves the API base URL.
 *  1. EXPO_PUBLIC_API_URL always wins (required for release builds).
 *  2. In development the API is assumed to run next to the Expo dev server on port 18080, so a physical
 *     phone reaches it through the LAN IP — never "localhost", which would be the phone itself.
 */
export function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return stripTrailingSlash(explicit);
  if (!__DEV__) return "";

  if (Platform.OS === "web" && typeof window !== "undefined") {
    return `${window.location.protocol}//${window.location.hostname}:${API_PORT}`;
  }
  const host = devServerHost();
  if (host) return `http://${host}:${API_PORT}`;
  return Platform.OS === "android" ? `http://10.0.2.2:${API_PORT}` : `http://localhost:${API_PORT}`;
}

export const API_BASE_URL = resolveApiBaseUrl();

export function toWebSocketUrl(baseUrl: string): string {
  return `${baseUrl.replace(/^http/, "ws")}/ws`;
}
