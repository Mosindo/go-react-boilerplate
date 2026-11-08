import { Platform } from "react-native";

function normalize(url: string): string {
  return url.replace(/\/+$/, "");
}

function resolveApiBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL?.trim();
  if (explicit) return normalize(explicit);
  if (__DEV__) {
    // Android emulator reaches the host through 10.0.2.2. A physical phone needs EXPO_PUBLIC_API_URL
    // set to the computer's LAN IP — "localhost" would point at the phone itself.
    return Platform.OS === "android" ? "http://10.0.2.2:18080" : "http://localhost:18080";
  }
  return "";
}

export const API_BASE_URL = resolveApiBaseUrl();

export function websocketUrl(): string {
  return API_BASE_URL.replace(/^http/, "ws") + "/ws";
}
